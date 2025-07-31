package user

import (
	"encoding/json"
	"slices"
	"strings"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/volatiletech/null/v9"
)

// GetAIAssistant retrieves an AI assistant by ID.
func (u *Manager) GetAIAssistant(id int) (models.User, error) {
	return u.Get(id, "", models.UserTypeAIAssistant)
}

// GetAIAssistants returns a list of all AI assistants.
func (u *Manager) GetAIAssistants() ([]models.User, error) {
	return u.GetAllUsers(1, 999999999, models.UserTypeAIAssistant, "desc", "users.updated_at", "")
}

// CreateAIAssistant creates a new AI assistant user.
func (u *Manager) CreateAIAssistant(user *models.User) error {
	// Validate and prepare meta data
	if err := u.validateAIAssistantMeta(user.Meta); err != nil {
		return err
	}

	// AI assistants do not have an email, so we set it to null
	user.Email = null.NewString("", false)
	if err := u.q.InsertAIAssistant.QueryRow(user.Email, user.FirstName, user.LastName, user.AvatarURL, user.Meta).Scan(&user.ID); err != nil {
		u.lo.Error("error creating AI assistant", "error", err)
		return envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.aiAssistant}"), nil)
	}
	return nil
}

// UpdateAIAssistant updates an AI assistant in the database.
func (u *Manager) UpdateAIAssistant(id int, user models.User) error {
	// Validate meta data
	if err := u.validateAIAssistantMeta(user.Meta); err != nil {
		return err
	}

	if _, err := u.q.UpdateAIAssistant.Exec(id, user.FirstName, user.LastName, user.Email, user.AvatarURL, user.Meta, user.Enabled); err != nil {
		u.lo.Error("error updating AI assistant", "error", err)
		return envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.aiAssistant}"), nil)
	}
	return nil
}

// SoftDeleteAIAssistant soft deletes an AI assistant by ID.
func (u *Manager) SoftDeleteAIAssistant(id int) error {
	if _, err := u.q.SoftDeleteAIAssistant.Exec(id); err != nil {
		u.lo.Error("error deleting AI assistant", "error", err)
		return envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.aiAssistant}"), nil)
	}
	return nil
}

// parseAIAssistantMeta parses the JSON meta field into AIAssistantMeta struct
func (u *Manager) parseAIAssistantMeta(metaBytes json.RawMessage) (models.AIAssistantMeta, error) {
	var meta models.AIAssistantMeta
	if len(metaBytes) == 0 {
		return meta, nil
	}

	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		u.lo.Error("error parsing AI assistant meta", "error", err)
		return meta, envelope.NewError(envelope.InputError, u.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.aiAssistant} meta"), nil)
	}
	return meta, nil
}

// validateAIAssistantMeta validates the AI assistant meta data
func (u *Manager) validateAIAssistantMeta(metaBytes json.RawMessage) error {
	meta, err := u.parseAIAssistantMeta(metaBytes)
	if err != nil {
		u.lo.Error("error parsing AI assistant meta", "error", err)
		return err
	}

	// Validate required fields
	if strings.TrimSpace(meta.ProductName) == "" {
		u.lo.Error("error validating product name", "invalid value", meta.ProductName)
		return envelope.NewError(envelope.InputError, u.i18n.Ts("globals.messages.errorValidating", "name", u.i18n.T("globals.terms.request")), nil)
	}

	if strings.TrimSpace(meta.ProductDescription) == "" {
		u.lo.Error("error validating product description", "invalid value", meta.ProductDescription)
		return envelope.NewError(envelope.InputError, u.i18n.Ts("globals.messages.errorValidating", "name", u.i18n.T("globals.terms.request")), nil)
	}

	// Validate answer length
	validAnswerLengths := []string{"concise", "medium", "long"}
	if !slices.Contains(validAnswerLengths, meta.AnswerLength) {
		u.lo.Error("error validating answer length", "invalid value", meta.AnswerLength)
		return envelope.NewError(envelope.InputError, u.i18n.Ts("globals.messages.errorValidating", "name", u.i18n.T("globals.terms.request")), nil)
	}

	// Validate answer tone
	validAnswerTones := []string{"neutral", "friendly", "professional", "humorous"}
	if !slices.Contains(validAnswerTones, meta.AnswerTone) {
		u.lo.Error("error validating answer tone", "invalid value", meta.AnswerTone)
		return envelope.NewError(envelope.InputError, u.i18n.Ts("globals.messages.errorValidating", "name", u.i18n.T("globals.terms.request")), nil)
	}

	return nil
}
