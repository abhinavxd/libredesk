package ai

import (
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/abhinavxd/libredesk/internal/ai/models"
	"github.com/abhinavxd/libredesk/internal/crypto"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/jmoiron/sqlx/types"
)

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

// GetEnabledTools returns enabled custom tools for the agent registry (auth intact).
func (m *Manager) GetEnabledTools() ([]models.Tool, error) {
	tools := make([]models.Tool, 0)
	if err := m.q.GetEnabledTools.Select(&tools); err != nil {
		m.lo.Error("error fetching enabled tools", "error", err)
		return nil, err
	}
	return tools, nil
}

func (m *Manager) CreateTool(t models.Tool) (models.Tool, error) {
	if reservedToolNames[t.Name] {
		return t, envelope.NewError(envelope.InputError, m.i18n.T("admin.ai.tool.reservedName"), nil)
	}
	if t.Method = strings.ToUpper(t.Method); !allowedToolMethods[t.Method] {
		return t, envelope.NewError(envelope.InputError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	auth, err := m.prepareToolAuth(t.Auth, nil)
	if err != nil {
		return t, err
	}
	params := t.Parameters
	if len(params) == 0 {
		params = types.JSONText("{}")
	}
	var created models.Tool
	if err := m.q.InsertTool.Get(&created, t.Name, t.Description, t.URL, t.Method, auth, params, t.Enabled); err != nil {
		m.lo.Error("error creating tool", "error", err)
		return created, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	created.Auth = maskToolAuth(created.Auth)
	return created, nil
}

func (m *Manager) UpdateTool(id int, t models.Tool) (models.Tool, error) {
	if reservedToolNames[t.Name] {
		return t, envelope.NewError(envelope.InputError, m.i18n.T("admin.ai.tool.reservedName"), nil)
	}
	if t.Method = strings.ToUpper(t.Method); !allowedToolMethods[t.Method] {
		return t, envelope.NewError(envelope.InputError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
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
	params := t.Parameters
	if len(params) == 0 {
		params = types.JSONText("{}")
	}
	var updated models.Tool
	if err := m.q.UpdateTool.Get(&updated, id, t.Name, t.Description, t.URL, t.Method, auth, params, t.Enabled); err != nil {
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

// prepareToolAuth returns the auth JSON to store: a blank or dummy-masked value keeps the existing secret, a new plaintext value is encrypted.
func (m *Manager) prepareToolAuth(raw, existing types.JSONText) (types.JSONText, error) {
	if len(raw) == 0 {
		return types.JSONText("{}"), nil
	}
	var auth models.ToolAuth
	if err := json.Unmarshal(raw, &auth); err != nil {
		return raw, envelope.NewError(envelope.InputError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	switch {
	case auth.Value == "" || strings.Contains(auth.Value, stringutil.PasswordDummy):
		auth.Value = toolAuthValue(existing)
	case !crypto.IsEncrypted(auth.Value):
		enc, err := crypto.Encrypt(auth.Value, m.encryptionKey)
		if err != nil {
			m.lo.Error("error encrypting tool auth", "error", err)
			return raw, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
		}
		auth.Value = enc
	}
	b, err := json.Marshal(auth)
	if err != nil {
		return raw, err
	}
	return types.JSONText(b), nil
}

// maskToolAuth replaces the stored secret with a dummy value.
func maskToolAuth(raw types.JSONText) types.JSONText {
	if len(raw) == 0 {
		return types.JSONText("{}")
	}
	var auth models.ToolAuth
	if err := json.Unmarshal(raw, &auth); err != nil {
		return types.JSONText("{}")
	}
	if auth.Value != "" {
		auth.Value = strings.Repeat(stringutil.PasswordDummy, 10)
	}
	b, err := json.Marshal(auth)
	if err != nil {
		return types.JSONText("{}")
	}
	return types.JSONText(b)
}

// toolAuthValue returns the stored (encrypted) auth value from a raw auth JSON.
func toolAuthValue(raw types.JSONText) string {
	if len(raw) == 0 {
		return ""
	}
	var auth models.ToolAuth
	if err := json.Unmarshal(raw, &auth); err != nil {
		return ""
	}
	return auth.Value
}
