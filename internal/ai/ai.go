// Package ai manages AI prompts and integrates with LLM providers.
package ai

import (
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"strings"

	"github.com/abhinavxd/libredesk/internal/ai/models"
	"github.com/abhinavxd/libredesk/internal/crypto"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS

	ErrInvalidAPIKey = errors.New("invalid API Key")
	ErrApiKeyNotSet  = errors.New("api Key not set")
)

type Manager struct {
	q             queries
	lo            *logf.Logger
	i18n          *i18n.I18n
	encryptionKey string
}

// Opts contains options for initializing the Manager.
type Opts struct {
	DB            *sqlx.DB
	I18n          *i18n.I18n
	Lo            *logf.Logger
	EncryptionKey string
}

// queries contains prepared SQL queries.
type queries struct {
	GetDefaultProvider   *sqlx.Stmt `query:"get-default-provider"`
	GetPrompt            *sqlx.Stmt `query:"get-prompt"`
	GetPrompts           *sqlx.Stmt `query:"get-prompts"`
	UpdateProviderConfig *sqlx.Stmt `query:"update-provider-config"`
}

// providerConfig represents the JSONB config stored in ai_providers.
type providerConfig struct {
	APIKey      string `json:"api_key"`
	EndpointURL string `json:"endpoint_url"`
	Model       string `json:"model"`
}

// ProviderInfo is the sanitized response for GET /ai/provider (no decrypted key).
type ProviderInfo struct {
	Provider    string `json:"provider"`
	EndpointURL string `json:"endpoint_url"`
	Model       string `json:"model"`
	HasAPIKey   bool   `json:"has_api_key"`
}

// New creates and returns a new instance of the Manager.
func New(opts Opts) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}
	return &Manager{
		q:             q,
		lo:            opts.Lo,
		i18n:          opts.I18n,
		encryptionKey: opts.EncryptionKey,
	}, nil
}

// Completion sends a prompt to the default provider and returns the response.
func (m *Manager) Completion(k string, prompt string) (string, error) {
	systemPrompt, err := m.getPrompt(k)
	if err != nil {
		return "", err
	}

	client, err := m.getDefaultProviderClient()
	if err != nil {
		m.lo.Error("error getting provider client", "error", err)
		return "", envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", m.i18n.Ts("globals.terms.provider")), nil)
	}

	payload := PromptPayload{
		SystemPrompt: systemPrompt,
		UserPrompt:   prompt,
	}

	response, err := client.SendPrompt(payload)
	if err != nil {
		if errors.Is(err, ErrInvalidAPIKey) {
			m.lo.Error("error invalid API key", "error", err)
			return "", envelope.NewError(envelope.InputError, m.i18n.Ts("globals.messages.invalid", "name", "OpenAI API Key"), nil)
		}
		if errors.Is(err, ErrApiKeyNotSet) {
			m.lo.Error("error API key not set", "error", err)
			return "", envelope.NewError(envelope.InputError, m.i18n.Ts("ai.apiKeyNotSet", "provider", "OpenAI"), nil)
		}
		m.lo.Error("error sending prompt to provider", "error", err)
		return "", envelope.NewError(envelope.GeneralError, err.Error(), nil)
	}

	return response, nil
}

// GetPrompts returns a list of prompts from the database.
func (m *Manager) GetPrompts() ([]models.Prompt, error) {
	var prompts = make([]models.Prompt, 0)
	if err := m.q.GetPrompts.Select(&prompts); err != nil {
		m.lo.Error("error fetching prompts", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", m.i18n.Ts("globals.terms.template")), nil)
	}
	return prompts, nil
}

// GetProvider returns the default provider's config (sanitized, no decrypted key).
func (m *Manager) GetProvider() (ProviderInfo, error) {
	var p models.Provider
	if err := m.q.GetDefaultProvider.Get(&p); err != nil {
		m.lo.Error("error fetching provider details", "error", err)
		return ProviderInfo{}, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", m.i18n.Ts("globals.terms.provider")), nil)
	}

	var cfg providerConfig
	if p.Config != "" {
		if err := json.Unmarshal([]byte(p.Config), &cfg); err != nil {
			m.lo.Error("error parsing provider config", "error", err)
			return ProviderInfo{}, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorParsing", "name", m.i18n.Ts("globals.terms.provider")), nil)
		}
	}

	return ProviderInfo{
		Provider:    p.Provider,
		EndpointURL: cfg.EndpointURL,
		Model:       cfg.Model,
		HasAPIKey:   cfg.APIKey != "",
	}, nil
}

// UpdateProvider updates a provider's config.
func (m *Manager) UpdateProvider(provider, apiKey, endpointURL, model string) error {
	// Validate endpoint URL if provided.
	if endpointURL != "" && !strings.HasPrefix(endpointURL, "http://") && !strings.HasPrefix(endpointURL, "https://") {
		return envelope.NewError(envelope.InputError, m.i18n.Ts("globals.messages.invalid", "name", "endpoint URL"), nil)
	}

	switch ProviderType(provider) {
	case ProviderOpenAI:
		return m.updateOpenAIProvider(apiKey, endpointURL, model)
	default:
		m.lo.Error("unsupported provider type", "provider", provider)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.invalid", "name", m.i18n.Ts("globals.terms.provider")), nil)
	}
}

// updateOpenAIProvider updates the OpenAI provider config in the database.
// If apiKey is empty, the existing key is preserved.
func (m *Manager) updateOpenAIProvider(apiKey, endpointURL, model string) error {
	// Read current config to preserve fields not being updated.
	var p models.Provider
	if err := m.q.GetDefaultProvider.Get(&p); err != nil {
		m.lo.Error("error fetching provider for update", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", m.i18n.Ts("globals.terms.provider")), nil)
	}

	var current providerConfig
	if p.Config != "" {
		if err := json.Unmarshal([]byte(p.Config), &current); err != nil {
			m.lo.Error("error parsing provider config for update", "error", err)
			return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorParsing", "name", m.i18n.Ts("globals.terms.provider")), nil)
		}
	}

	// Only overwrite api_key if a new one is provided.
	if apiKey != "" {
		encryptedKey, err := crypto.Encrypt(apiKey, m.encryptionKey)
		if err != nil {
			m.lo.Error("error encrypting API key", "error", err)
			return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "OpenAI API Key"), nil)
		}
		current.APIKey = encryptedKey
	}

	// Update endpoint_url and model (empty string means use defaults at runtime).
	current.EndpointURL = endpointURL
	current.Model = model

	configJSON, err := json.Marshal(current)
	if err != nil {
		m.lo.Error("error marshalling provider config", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", m.i18n.Ts("globals.terms.provider")), nil)
	}

	if _, err := m.q.UpdateProviderConfig.Exec(string(configJSON)); err != nil {
		m.lo.Error("error updating provider config", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", m.i18n.Ts("globals.terms.provider")), nil)
	}
	return nil
}

// getPrompt returns a prompt from the database.
func (m *Manager) getPrompt(k string) (string, error) {
	var p models.Prompt
	if err := m.q.GetPrompt.Get(&p, k); err != nil {
		if err == sql.ErrNoRows {
			m.lo.Error("error prompt not found", "key", k)
			return "", envelope.NewError(envelope.InputError, m.i18n.Ts("globals.messages.notFound", "name", m.i18n.Ts("globals.terms.template")), nil)
		}
		m.lo.Error("error fetching prompt", "error", err)
		return "", envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", m.i18n.Ts("globals.terms.template")), nil)
	}
	return p.Content, nil
}

// getDefaultProviderClient returns a ProviderClient for the default provider.
func (m *Manager) getDefaultProviderClient() (ProviderClient, error) {
	var p models.Provider

	if err := m.q.GetDefaultProvider.Get(&p); err != nil {
		m.lo.Error("error fetching provider details", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", m.i18n.Ts("globals.terms.provider")), nil)
	}

	switch ProviderType(p.Provider) {
	case ProviderOpenAI:
		var cfg providerConfig
		if err := json.Unmarshal([]byte(p.Config), &cfg); err != nil {
			m.lo.Error("error parsing provider config", "error", err)
			return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorParsing", "name", m.i18n.Ts("globals.terms.provider")), nil)
		}
		// Decrypt API key.
		decryptedKey, err := crypto.Decrypt(cfg.APIKey, m.encryptionKey)
		if err != nil {
			m.lo.Error("error decrypting API key", "error", err)
			return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", m.i18n.Ts("globals.terms.provider")), nil)
		}
		return NewOpenAIClient(decryptedKey, cfg.EndpointURL, cfg.Model, m.lo), nil
	default:
		m.lo.Error("unsupported provider type", "provider", p.Provider)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.invalid", "name", m.i18n.Ts("globals.terms.provider")), nil)
	}
}
