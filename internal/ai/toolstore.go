package ai

import (
	"database/sql"
	"encoding/json"
	neturl "net/url"
	"strings"

	"github.com/abhinavxd/libredesk/internal/ai/models"
	"github.com/abhinavxd/libredesk/internal/crypto"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/jmoiron/sqlx/types"
	"github.com/lib/pq"
)

var emptyToolAuth = types.JSONText(`{"headers":[]}`)

// GetTools returns all custom tools (auth values masked - never returned to the client).
func (m *Manager) GetTools() ([]models.Tool, error) {
	tools := make([]models.Tool, 0)
	if err := m.q.GetTools.Select(&tools); err != nil {
		m.lo.Error("error fetching tools", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	for i := range tools {
		tools[i].Auth = maskToolAuth(tools[i].Auth)
	}
	return tools, nil
}

// GetTool returns a single custom tool by id (auth secret masked).
func (m *Manager) GetTool(id int) (models.Tool, error) {
	var tool models.Tool
	if err := m.q.GetTool.Get(&tool, id); err != nil {
		if err == sql.ErrNoRows {
			return tool, envelope.NewError(envelope.NotFoundError, m.i18n.T("globals.messages.notFound"), nil)
		}
		m.lo.Error("error fetching tool", "error", err)
		return tool, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	tool.Auth = maskToolAuth(tool.Auth)
	return tool, nil
}

// GetEnabledToolsByIDs returns the enabled custom tools among ids for the agent registry (auth intact).
func (m *Manager) GetEnabledToolsByIDs(ids []int) ([]models.Tool, error) {
	tools := make([]models.Tool, 0)
	if err := m.q.GetEnabledToolsByIDs.Select(&tools, pq.Array(ids)); err != nil {
		m.lo.Error("error fetching enabled tools by ids", "error", err)
		return nil, err
	}
	return tools, nil
}

func (m *Manager) CreateTool(t models.Tool) (models.Tool, error) {
	if err := m.validateToolInput(&t); err != nil {
		return t, err
	}
	auth, err := m.prepareToolAuth(t.Auth, nil)
	if err != nil {
		return t, err
	}
	params := toolParametersOrEmpty(t)
	var created models.Tool
	if err := m.q.InsertTool.Get(&created, t.Name, t.Description, t.URL, t.Method, auth, params, t.Enabled, t.RequiresVerification); err != nil {
		m.lo.Error("error creating tool", "error", err)
		return created, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	created.Auth = maskToolAuth(created.Auth)
	return created, nil
}

func (m *Manager) UpdateTool(id int, t models.Tool) (models.Tool, error) {
	if err := m.validateToolInput(&t); err != nil {
		return t, err
	}
	var existing types.JSONText
	if err := m.q.GetToolAuth.Get(&existing, id); err != nil {
		if err == sql.ErrNoRows {
			return t, envelope.NewError(envelope.NotFoundError, m.i18n.T("globals.messages.notFound"), nil)
		}
		m.lo.Error("error fetching existing tool auth", "error", err)
		return t, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	auth, err := m.prepareToolAuth(t.Auth, existing)
	if err != nil {
		return t, err
	}
	params := toolParametersOrEmpty(t)
	var updated models.Tool
	if err := m.q.UpdateTool.Get(&updated, id, t.Name, t.Description, t.URL, t.Method, auth, params, t.Enabled, t.RequiresVerification); err != nil {
		if err == sql.ErrNoRows {
			return updated, envelope.NewError(envelope.NotFoundError, m.i18n.T("globals.messages.notFound"), nil)
		}
		m.lo.Error("error updating tool", "error", err)
		return updated, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	updated.Auth = maskToolAuth(updated.Auth)
	return updated, nil
}

func (m *Manager) DeleteTool(id int) error {
	if _, err := m.q.DeleteTool.Exec(id); err != nil {
		m.lo.Error("error deleting tool", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

// prepareToolAuth returns the auth JSON to store: per header, a blank or dummy-masked value keeps the
// existing secret stored under that key, a new plaintext value is encrypted.
func (m *Manager) prepareToolAuth(raw, existing types.JSONText) (types.JSONText, error) {
	if len(raw) == 0 {
		return emptyToolAuth, nil
	}
	var auth models.ToolAuth
	if err := json.Unmarshal(raw, &auth); err != nil {
		return raw, envelope.NewError(envelope.InputError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	existingHeaders := toolAuthHeaders(existing)
	for i, h := range auth.Headers {
		switch {
		case h.Value == "" || strings.Contains(h.Value, stringutil.PasswordDummy):
			// Matched by key only. Falling back to row position would hand a renamed header the secret
			// of whichever row now sits at its index, which is silently the wrong credential.
			existingValue := ""
			for _, e := range existingHeaders {
				if e.Key == h.Key && e.Value != "" {
					existingValue = e.Value
					break
				}
			}
			if existingValue == "" {
				return raw, envelope.NewError(envelope.InputError, m.i18n.T("admin.ai.tool.headerSecretMissing"), nil)
			}
			auth.Headers[i].Value = existingValue
		case crypto.IsEncrypted(h.Value):
			// crypto.Encrypt would store this user-supplied value as-is and Decrypt would then fail at call time.
			return raw, envelope.NewError(envelope.InputError, m.i18n.T("admin.ai.reservedSecretPrefix"), nil)
		default:
			enc, err := crypto.Encrypt(h.Value, m.encryptionKey)
			if err != nil {
				m.lo.Error("error encrypting tool auth", "error", err)
				return raw, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
			}
			auth.Headers[i].Value = enc
		}
	}
	b, err := json.Marshal(auth)
	if err != nil {
		return raw, err
	}
	return types.JSONText(b), nil
}

// validateToolInput runs the shared create/update checks: name format, reserved name, allowed HTTP method, valid definition.
func (m *Manager) validateToolInput(t *models.Tool) error {
	if !toolNameFormat.MatchString(t.Name) || len(t.Name) > maxToolNameLen {
		return envelope.NewError(envelope.InputError, m.i18n.T("admin.ai.tool.nameHint"), nil)
	}
	if reservedToolNames[t.Name] {
		return envelope.NewError(envelope.InputError, m.i18n.T("admin.ai.tool.reservedName"), nil)
	}
	if t.Method = strings.ToUpper(t.Method); !allowedToolMethods[t.Method] {
		return envelope.NewError(envelope.InputError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return m.validateToolDefinition(t)
}

// validateToolDefinition rejects a tool whose URL or parameters schema would fail every agent run at request time.
func (m *Manager) validateToolDefinition(t *models.Tool) error {
	t.URL = strings.TrimSpace(t.URL)
	u, err := neturl.Parse(t.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return envelope.NewError(envelope.InputError, m.i18n.T("ai.urlImport.invalidUrl"), nil)
	}
	params := strings.TrimSpace(string(t.Parameters))
	if params == "" || params == "null" {
		t.Parameters = nil
		return nil
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal([]byte(params), &obj); err != nil {
		return envelope.NewError(envelope.InputError, m.i18n.T("admin.ai.tool.parametersInvalid"), nil)
	}
	return nil
}

func toolParametersOrEmpty(t models.Tool) types.JSONText {
	if len(t.Parameters) == 0 {
		return types.JSONText("{}")
	}
	return t.Parameters
}

// maskToolAuth replaces every header's stored secret with a dummy value.
func maskToolAuth(raw types.JSONText) types.JSONText {
	headers := toolAuthHeaders(raw)
	if len(headers) == 0 {
		return emptyToolAuth
	}
	for i, h := range headers {
		if h.Value != "" {
			headers[i].Value = strings.Repeat(stringutil.PasswordDummy, 10)
		}
	}
	b, err := json.Marshal(models.ToolAuth{Headers: headers})
	if err != nil {
		return emptyToolAuth
	}
	return types.JSONText(b)
}

// toolAuthHeaders returns the stored headers, values still encrypted, from a raw auth JSON.
func toolAuthHeaders(raw types.JSONText) []models.ToolAuthHeader {
	if len(raw) == 0 {
		return nil
	}
	var auth models.ToolAuth
	if err := json.Unmarshal(raw, &auth); err != nil {
		return nil
	}
	return auth.Headers
}
