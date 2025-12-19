// Package ai manages AI prompts and integrates with LLM providers.
package ai

import (
	"database/sql"
	"embed"
	"encoding/json"
	"errors"

	"github.com/abhinavxd/libredesk/internal/ai/models"
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
	q    queries
	lo   *logf.Logger
	i18n *i18n.I18n
}

// Opts contains options for initializing the Manager.
type Opts struct {
	DB   *sqlx.DB
	I18n *i18n.I18n
	Lo   *logf.Logger
}

// queries contains prepared SQL queries.
type queries struct {
	GetDefaultProvider   *sqlx.Stmt `query:"get-default-provider"`
	GetPrompt            *sqlx.Stmt `query:"get-prompt"`
	GetPrompts           *sqlx.Stmt `query:"get-prompts"`
	SetOpenAIKey         *sqlx.Stmt `query:"set-openai-key"`
	SetOpenRouterConfig  *sqlx.Stmt `query:"set-openrouter-config"`
	GetProviders         *sqlx.Stmt `query:"get-providers"`
	SetDefaultProvider   *sqlx.Stmt `query:"set-default-provider"`
	UpsertOpenRouter     *sqlx.Stmt `query:"upsert-openrouter"`
	GetProvider          *sqlx.Stmt `query:"get-provider"`
}

// New creates and returns a new instance of the Manager.
func New(opts Opts) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}
	return &Manager{
		q:    q,
		lo:   opts.Lo,
		i18n: opts.I18n,
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
			return "", envelope.NewError(envelope.InputError, m.i18n.Ts("globals.messages.invalid", "name", "API Key"), nil)
		}
		if errors.Is(err, ErrApiKeyNotSet) {
			m.lo.Error("error API key not set", "error", err)
			return "", envelope.NewError(envelope.InputError, m.i18n.Ts("ai.apiKeyNotSet", "provider", "AI"), nil)
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

// GetProviders returns information about all configured providers.
func (m *Manager) GetProviders() ([]ProviderInfo, error) {
	var providers = make([]models.Provider, 0)
	if err := m.q.GetProviders.Select(&providers); err != nil {
		m.lo.Error("error fetching providers", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", m.i18n.Ts("globals.terms.provider")), nil)
	}

	result := make([]ProviderInfo, 0, len(providers))
	for _, p := range providers {
		info := ProviderInfo{
			Provider:  p.Provider,
			Name:      p.Name,
			IsDefault: p.IsDefault,
		}

		// Parse config to check if API key is set and get model
		var config map[string]interface{}
		if err := json.Unmarshal([]byte(p.Config), &config); err == nil {
			if apiKey, ok := config["api_key"].(string); ok && apiKey != "" {
				info.HasAPIKey = true
			}
			if model, ok := config["model"].(string); ok {
				info.Model = model
			}
		}
		result = append(result, info)
	}
	return result, nil
}

// GetAvailableModels returns available models for OpenRouter.
func (m *Manager) GetAvailableModels() []string {
	models, _ := FetchOpenRouterModels()
	return models
}

// UpdateProvider updates a provider.
func (m *Manager) UpdateProvider(provider, apiKey, model string) error {
	switch ProviderType(provider) {
	case ProviderOpenAI:
		return m.setOpenAIAPIKey(apiKey)
	case ProviderOpenRouter:
		return m.setOpenRouterConfig(apiKey, model)
	default:
		m.lo.Error("unsupported provider type", "provider", provider)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.invalid", "name", m.i18n.Ts("globals.terms.provider")), nil)
	}
}

// SetDefaultProvider sets a provider as the default.
func (m *Manager) SetDefaultProvider(provider string) error {
	if _, err := m.q.SetDefaultProvider.Exec(provider); err != nil {
		m.lo.Error("error setting default provider", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", m.i18n.Ts("globals.terms.provider")), nil)
	}
	return nil
}

// TestProvider tests the connection to a provider.
// getSavedAPIKey retrieves the saved API key for a provider from the database.
func (m *Manager) getSavedAPIKey(provider string) (string, string) {
	rows, err := m.q.GetProviders.Queryx()
	if err != nil {
		return "", ""
	}
	defer rows.Close()

	for rows.Next() {
		var p models.Provider
		if err := rows.StructScan(&p); err != nil {
			continue
		}
		if p.Provider != provider {
			continue
		}
		var config map[string]interface{}
		if err := json.Unmarshal([]byte(p.Config), &config); err != nil {
			return "", ""
		}
		apiKey, _ := config["api_key"].(string)
		model, _ := config["model"].(string)
		return apiKey, model
	}
	return "", ""
}

// TestProvider tests the connection to a provider.
// If apiKey is empty, it uses the saved key from the database.
func (m *Manager) TestProvider(provider, apiKey, model string) error {
	// If no API key provided, use saved one from database
	if apiKey == "" {
		savedKey, savedModel := m.getSavedAPIKey(provider)
		apiKey = savedKey
		if model == "" {
			model = savedModel
		}
	}

	var client ProviderClient
	switch ProviderType(provider) {
	case ProviderOpenAI:
		client = NewOpenAIClient(apiKey, m.lo)
	case ProviderOpenRouter:
		client = NewOpenRouterClient(apiKey, model, m.lo)
	default:
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.invalid", "name", m.i18n.Ts("globals.terms.provider")), nil)
	}

	// Test with a simple prompt
	_, err := client.SendPrompt(PromptPayload{
		SystemPrompt: "You are a helpful assistant.",
		UserPrompt:   "Say OK to confirm the connection works.",
	})
	if err != nil {
		if errors.Is(err, ErrInvalidAPIKey) {
			return envelope.NewError(envelope.InputError, "Invalid API Key", nil)
		}
		if errors.Is(err, ErrApiKeyNotSet) {
			return envelope.NewError(envelope.InputError, "API Key not set", nil)
		}
		return envelope.NewError(envelope.GeneralError, err.Error(), nil)
	}
	return nil
}

// setOpenAIAPIKey sets the OpenAI API key in the database.
func (m *Manager) setOpenAIAPIKey(apiKey string) error {
	if _, err := m.q.SetOpenAIKey.Exec(apiKey); err != nil {
		m.lo.Error("error setting OpenAI API key", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "OpenAI API Key"), nil)
	}
	return nil
}

// setOpenRouterConfig sets the OpenRouter config in the database.
func (m *Manager) setOpenRouterConfig(apiKey, model string) error {
	if model == "" {
		model = "anthropic/claude-3-haiku"
	}

	// First, ensure OpenRouter provider exists
	if _, err := m.q.UpsertOpenRouter.Exec(); err != nil {
		m.lo.Error("error upserting OpenRouter provider", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "OpenRouter"), nil)
	}

	// Then update its config
	if _, err := m.q.SetOpenRouterConfig.Exec(apiKey, model); err != nil {
		m.lo.Error("error setting OpenRouter config", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "OpenRouter"), nil)
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

	var config struct {
		APIKey string `json:"api_key"`
		Model  string `json:"model"`
	}
	if err := json.Unmarshal([]byte(p.Config), &config); err != nil {
		m.lo.Error("error parsing provider config", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorParsing", "name", m.i18n.Ts("globals.terms.provider")), nil)
	}

	switch ProviderType(p.Provider) {
	case ProviderOpenAI:
		return NewOpenAIClient(config.APIKey, m.lo), nil
	case ProviderOpenRouter:
		return NewOpenRouterClient(config.APIKey, config.Model, m.lo), nil
	default:
		m.lo.Error("unsupported provider type", "provider", p.Provider)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.invalid", "name", m.i18n.Ts("globals.terms.provider")), nil)
	}
}

// GenerateEmbedding generates an embedding for the given text.
// This always uses OpenAI since OpenRouter doesn't support embeddings.
func (m *Manager) GenerateEmbedding(text string) ([]float32, error) {
	// Get OpenAI API key - embeddings require OpenAI regardless of default provider
	var p models.Provider
	if err := m.q.GetProvider.Get(&p, string(ProviderOpenAI)); err != nil {
		if err == sql.ErrNoRows {
			return nil, envelope.NewError(envelope.InputError, "OpenAI API key required for embeddings", nil)
		}
		m.lo.Error("error fetching OpenAI provider", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "OpenAI provider"), nil)
	}

	var config struct {
		APIKey string `json:"api_key"`
	}
	if err := json.Unmarshal([]byte(p.Config), &config); err != nil {
		m.lo.Error("error parsing OpenAI config", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, "Error parsing OpenAI config", nil)
	}

	if config.APIKey == "" {
		return nil, envelope.NewError(envelope.InputError, "OpenAI API key required for embeddings", nil)
	}

	client := NewOpenAIClient(config.APIKey, m.lo)
	return client.GenerateEmbedding(text)
}

// CompletionWithSystemPrompt sends a prompt with a custom system prompt to the default provider.
func (m *Manager) CompletionWithSystemPrompt(systemPrompt, userPrompt string) (string, error) {
	client, err := m.getDefaultProviderClient()
	if err != nil {
		m.lo.Error("error getting provider client", "error", err)
		return "", envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", m.i18n.Ts("globals.terms.provider")), nil)
	}

	payload := PromptPayload{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
	}

	response, err := client.SendPrompt(payload)
	if err != nil {
		if errors.Is(err, ErrInvalidAPIKey) {
			m.lo.Error("error invalid API key", "error", err)
			return "", envelope.NewError(envelope.InputError, m.i18n.Ts("globals.messages.invalid", "name", "API Key"), nil)
		}
		if errors.Is(err, ErrApiKeyNotSet) {
			m.lo.Error("error API key not set", "error", err)
			return "", envelope.NewError(envelope.InputError, m.i18n.Ts("ai.apiKeyNotSet", "provider", "AI Provider"), nil)
		}
		m.lo.Error("error sending prompt to provider", "error", err)
		return "", envelope.NewError(envelope.GeneralError, err.Error(), nil)
	}

	return response, nil
}
