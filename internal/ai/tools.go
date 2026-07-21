package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"strconv"
	"strings"

	"github.com/abhinavxd/libredesk/internal/ai/models"
	"github.com/abhinavxd/libredesk/internal/crypto"
	"github.com/jmoiron/sqlx/types"
	"github.com/zerodha/logf"
)

const (
	toolSearchArticles = "search_articles"

	maxToolResponseBytes = 1 << 20

	// maxToolResultChars caps tool output fed to the model so a large response can't blow the context window.
	maxToolResultChars = 64 << 10
)

var (
	// reservedToolNames are built-in tool names (including the autonomous assistant's, see
	// internal/aiagent/tools.go) custom tools may not use: a colliding custom tool would be
	// silently replaced by the built-in in the agent loop's registry.
	reservedToolNames = map[string]bool{
		toolSearchArticles:           true,
		"search_knowledge_base":      true,
		"hand_off_to_human":          true,
		"resolve":                    true,
		"get_previous_conversations": true,
		"send_email_verification":    true,
		"check_email_verification":   true,
		"set_contact_email":          true,
	}

	// allowedToolMethods are the HTTP methods a custom tool may use: GET reads, POST writes.
	allowedToolMethods = map[string]bool{http.MethodGet: true, http.MethodPost: true}

	searchArticlesParams = types.JSONText(`{
		"type": "object",
		"properties": {
			"query": {
				"type": "string",
				"description": "The natural-language search query to find relevant knowledge base snippets."
			}
		},
		"required": ["query"]
	}`)

	// defaultToolParams is used when an admin defines a custom tool without a JSON schema.
	defaultToolParams = types.JSONText(`{
		"type": "object",
		"properties": {
			"input": {
				"type": "string",
				"description": "Input passed to the tool."
			}
		}
	}`)
)

// Tool is a callable the agent loop can invoke. Its schema is advertised to the model.
type Tool interface {
	Name() string
	Description() string
	Parameters() types.JSONText
	Execute(ctx context.Context, args string) (string, error)
}

// searchArticlesTool is the built-in retrieval tool over embedded articles + snippets.
type searchArticlesTool struct {
	m *Manager
}

func (t *searchArticlesTool) Name() string { return toolSearchArticles }

func (t *searchArticlesTool) Description() string {
	return "Search the knowledge base snippets for information relevant to the customer's question. Returns the most relevant content chunks."
}

func (t *searchArticlesTool) Parameters() types.JSONText { return searchArticlesParams }

func (t *searchArticlesTool) Execute(ctx context.Context, args string) (string, error) {
	var in struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(args), &in); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if strings.TrimSpace(in.Query) == "" {
		return "No query provided.", nil
	}

	results, err := t.m.Search(ctx, in.Query, 5)
	if err != nil {
		return "", err
	}
	if len(results) == 0 {
		return "No relevant articles found in the knowledge base.", nil
	}

	var b strings.Builder
	for i, r := range results {
		fmt.Fprintf(&b, "[%d] (relevance %.2f)\n%s\n\n", i+1, r.Score, r.ChunkText)
	}
	return b.String(), nil
}

// ToolContext is per-run context injected into custom tool calls server-side. It is never part
// of the model-facing schema, so the model cannot see or spoof it (e.g. the contact's identity).
type ToolContext struct {
	ContactID         int
	ContactExternalID string
	ContactType       string
	ConversationUUID  string
	InboxID           int
	// ContactEmail is evaluated live per tool call (never snapshotted) so a contact whose email is
	// set or corrected mid-turn is identified by the current address, not the one at run start.
	ContactEmail func() string
	// Verified is evaluated live per tool call (never snapshotted) so a contact who verifies
	// mid-turn passes on the same turn. A nil func is treated as unverified (fail closed).
	Verified func() bool
}

func (c ToolContext) verified() bool { return c.Verified != nil && c.Verified() }

func (c ToolContext) contactEmail() string {
	if c.ContactEmail == nil {
		return ""
	}
	return c.ContactEmail()
}

// httpTool is an admin-defined custom tool that calls an external HTTP endpoint.
type httpTool struct {
	tool          models.Tool
	encryptionKey string
	lo            *logf.Logger
	client        *http.Client
	tctx          ToolContext
}

func newHTTPTool(t models.Tool, encryptionKey string, lo *logf.Logger, client *http.Client, tctx ToolContext) *httpTool {
	return &httpTool{
		tool:          t,
		encryptionKey: encryptionKey,
		lo:            lo,
		client:        client,
		tctx:          tctx,
	}
}

func (t *httpTool) Name() string { return t.tool.Name }

func (t *httpTool) Description() string { return t.tool.Description }

func (t *httpTool) Parameters() types.JSONText {
	if len(t.tool.Parameters) == 0 || strings.TrimSpace(string(t.tool.Parameters)) == "{}" {
		return defaultToolParams
	}
	return t.tool.Parameters
}

func (t *httpTool) Execute(ctx context.Context, args string) (string, error) {
	if t.tool.RequiresVerification && !t.tctx.verified() {
		t.lo.Debug("custom tool blocked, contact not verified", "tool", t.tool.Name)
		return "The contact is not verified. Call send_email_verification first, then ask them for the code and call check_email_verification before retrying this tool.", nil
	}

	method := strings.ToUpper(t.tool.Method)
	if method == "" {
		method = http.MethodPost
	}

	// GET carries no body, so the model's arguments go on the query string; other methods send them as
	// the JSON body.
	url := t.tool.URL
	var bodyReader io.Reader
	if method == http.MethodGet {
		u, perr := neturl.Parse(url)
		if perr != nil {
			return "", fmt.Errorf("invalid tool URL: %w", perr)
		}
		q, err := argsToQuery(args, u.Query())
		if err != nil {
			return "", err
		}
		if q != "" {
			if u.RawQuery != "" {
				u.RawQuery += "&" + q
			} else {
				u.RawQuery = q
			}
			url = u.String()
		}
	} else {
		bodyReader = bytes.NewBufferString(args)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return "", err
	}
	if bodyReader != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if len(t.tool.Auth) > 0 {
		var auth models.ToolAuth
		if err := json.Unmarshal(t.tool.Auth, &auth); err == nil {
			for _, h := range auth.Headers {
				if h.Key == "" {
					continue
				}
				value, derr := crypto.Decrypt(h.Value, t.encryptionKey)
				if derr != nil {
					// A failed decrypt means a corrupted or re-keyed secret; sending the raw stored value
					// would leak ciphertext and fail auth confusingly. Skip this header and let the call 401.
					t.lo.Error("could not decrypt custom tool auth secret; sending request without it", "tool", t.tool.Name, "header", h.Key, "error", derr)
					continue
				}
				req.Header.Set(h.Key, value)
			}
		}
	}

	// Inject the contact's identity so tools (e.g. a CRM lookup) know who this is. Server-side
	// only, never advertised to the model. The verified header is always sent (defaulting to
	// false) so a tool can distinguish an OTP-verified contact from a self-claimed one.
	if t.tctx.ContactID != 0 {
		req.Header.Set("X-Libredesk-Contact-Id", strconv.Itoa(t.tctx.ContactID))
	}
	if t.tctx.ContactExternalID != "" {
		req.Header.Set("X-Libredesk-Contact-External-Id", t.tctx.ContactExternalID)
	}
	if t.tctx.ContactType != "" {
		req.Header.Set("X-Libredesk-Contact-Type", t.tctx.ContactType)
	}
	if email := t.tctx.contactEmail(); email != "" {
		req.Header.Set("X-Libredesk-Contact-Email", email)
	}
	req.Header.Set("X-Libredesk-Contact-Verified", strconv.FormatBool(t.tctx.verified()))
	if t.tctx.ConversationUUID != "" {
		req.Header.Set("X-Libredesk-Conversation-UUID", t.tctx.ConversationUUID)
	}
	if t.tctx.InboxID != 0 {
		req.Header.Set("X-Libredesk-Inbox-Id", strconv.Itoa(t.tctx.InboxID))
	}

	resp, err := t.client.Do(req)
	if err != nil {
		t.lo.Error("error calling custom tool", "tool", t.tool.Name, "error", err)
		return "", err
	}
	defer resp.Body.Close()

	out, _ := io.ReadAll(io.LimitReader(resp.Body, maxToolResponseBytes))
	body := string(out)
	if len(body) > maxToolResultChars {
		body = trimToRuneBoundary(body, maxToolResultChars) + "\n[tool response truncated]"
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Sprintf("tool returned status %d: %s", resp.StatusCode, body), nil
	}
	return body, nil
}

// buildToolRegistry assembles the tools advertised to the model: custom tools are restricted to allowedToolIDs (empty loads none).
func (m *Manager) buildToolRegistry(tctx ToolContext, allowedToolIDs []int, includeBuiltinSearch bool) (map[string]Tool, []models.ToolDef, error) {
	registry := map[string]Tool{}
	var defs []models.ToolDef

	if includeBuiltinSearch {
		builtin := &searchArticlesTool{m: m}
		registry[builtin.Name()] = builtin
		defs = append(defs, toolDef(builtin))
	}

	if len(allowedToolIDs) == 0 {
		return registry, defs, nil
	}

	customTools, err := m.GetEnabledTools()
	if err != nil {
		return nil, nil, err
	}
	allowed := make(map[int]bool, len(allowedToolIDs))
	for _, id := range allowedToolIDs {
		allowed[id] = true
	}
	for _, ct := range customTools {
		if !allowed[ct.ID] {
			continue
		}
		if _, exists := registry[ct.Name]; exists {
			m.lo.Warn("skipping custom tool that collides with a built-in tool", "name", ct.Name)
			continue
		}
		ht := newHTTPTool(ct, m.encryptionKey, m.lo, m.toolHTTPClient, tctx)
		registry[ht.Name()] = ht
		defs = append(defs, toolDef(ht))
	}
	return registry, defs, nil
}

// argsToQuery flattens the model's JSON argument object into a URL query string, JSON-encoding any
// non-scalar values and dropping keys already pinned in the tool URL. A blank or "{}" args string yields an empty query.
func argsToQuery(args string, pinned neturl.Values) (string, error) {
	args = strings.TrimSpace(args)
	if args == "" || args == "{}" {
		return "", nil
	}
	dec := json.NewDecoder(strings.NewReader(args))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	q := neturl.Values{}
	for k, v := range m {
		if _, ok := pinned[k]; ok {
			continue
		}
		switch val := v.(type) {
		case nil:
			continue
		case string:
			q.Set(k, val)
		case json.Number:
			q.Set(k, val.String())
		case bool:
			q.Set(k, fmt.Sprintf("%v", val))
		default:
			b, err := json.Marshal(val)
			if err != nil {
				return "", fmt.Errorf("encoding argument %q: %w", k, err)
			}
			q.Set(k, string(b))
		}
	}
	return q.Encode(), nil
}

func toolDef(t Tool) models.ToolDef {
	return models.ToolDef{
		Type: "function",
		Function: models.ToolFunction{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.Parameters(),
		},
	}
}
