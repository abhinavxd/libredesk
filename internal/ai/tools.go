package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/ai/models"
	"github.com/abhinavxd/libredesk/internal/crypto"
	"github.com/jmoiron/sqlx/types"
	"github.com/zerodha/logf"
)

const (
	toolSearchArticles = "search_articles"
	// ToolContactHistory is reserved; the API layer injects this tool per request.
	ToolContactHistory = "get_contact_history"

	maxToolResponseBytes = 8000
)

var (
	// reservedToolNames are built-in tool names custom tools may not use.
	reservedToolNames = map[string]bool{toolSearchArticles: true, ToolContactHistory: true}

	toolHTTPClient = &http.Client{Timeout: 20 * time.Second}

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

// httpTool is an admin-defined custom tool that calls an external HTTP endpoint.
type httpTool struct {
	tool          models.Tool
	encryptionKey string
	lo            *logf.Logger
	client        *http.Client
}

func newHTTPTool(t models.Tool, encryptionKey string, lo *logf.Logger) *httpTool {
	return &httpTool{
		tool:          t,
		encryptionKey: encryptionKey,
		lo:            lo,
		client:        toolHTTPClient,
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
	method := strings.ToUpper(t.tool.Method)
	if method == "" {
		method = http.MethodPost
	}

	var bodyReader io.Reader
	if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch {
		bodyReader = bytes.NewBufferString(args)
	}

	req, err := http.NewRequestWithContext(ctx, method, t.tool.URL, bodyReader)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	if len(t.tool.Auth) > 0 {
		var auth models.ToolAuth
		if err := json.Unmarshal(t.tool.Auth, &auth); err == nil && auth.Header != "" {
			value, derr := crypto.Decrypt(auth.Value, t.encryptionKey)
			if derr != nil {
				t.lo.Error("error decrypting custom tool auth; skipping auth header", "tool", t.tool.Name, "error", derr)
			} else {
				req.Header.Set(auth.Header, value)
			}
		}
	}

	resp, err := t.client.Do(req)
	if err != nil {
		t.lo.Error("error calling custom tool", "tool", t.tool.Name, "error", err)
		return "", err
	}
	defer resp.Body.Close()

	out, _ := io.ReadAll(io.LimitReader(resp.Body, maxToolResponseBytes))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Sprintf("tool returned status %d: %s", resp.StatusCode, string(out)), nil
	}
	return string(out), nil
}

// buildToolRegistry returns the built-in, request-injected, and enabled custom tools plus their model-facing definitions.
func (m *Manager) buildToolRegistry(extra ...Tool) (map[string]Tool, []models.ToolDef, error) {
	registry := map[string]Tool{}
	var defs []models.ToolDef

	builtins := []Tool{&searchArticlesTool{m: m}}
	builtins = append(builtins, extra...)
	for _, t := range builtins {
		registry[t.Name()] = t
		defs = append(defs, toolDef(t))
	}

	customTools, err := m.GetEnabledTools()
	if err != nil {
		return nil, nil, err
	}
	for _, ct := range customTools {
		if _, exists := registry[ct.Name]; exists {
			m.lo.Warn("skipping custom tool that collides with a built-in tool", "name", ct.Name)
			continue
		}
		ht := newHTTPTool(ct, m.encryptionKey, m.lo)
		registry[ht.Name()] = ht
		defs = append(defs, toolDef(ht))
	}
	return registry, defs, nil
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
