// Package ai integrates with OpenAI-compatible LLM providers: agent-facing
// prompts, an agentic tool-calling loop, embeddings, and in-memory retrieval.
package ai

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/abhinavxd/libredesk/internal/ai/models"
	"github.com/abhinavxd/libredesk/internal/crypto"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/ssrf"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/zerodha/logf"
)

// Provider error bodies shown to admins on a connection test are capped at this length.
const maxTestErrorLen = 500

var (
	//go:embed queries.sql
	efs embed.FS

	ErrInvalidAPIKey = errors.New("invalid API Key")
	ErrApiKeyNotSet  = errors.New("api Key not set")
)

type Manager struct {
	q             queries
	db            *sqlx.DB
	lo            *logf.Logger
	i18n          *i18n.I18n
	encryptionKey string
	chunkCfg      stringutil.ChunkConfig
	index         *embeddingIndex
	reindexMu     sync.Mutex
	reconcileMu   sync.Mutex
	snippetGenMu  sync.Mutex
	snippetGen    map[int]uint64
	dialControl   ssrf.Control
	httpClient    *http.Client
}

// Opts contains options for initializing the Manager.
type Opts struct {
	DB            *sqlx.DB
	I18n          *i18n.I18n
	Lo            *logf.Logger
	EncryptionKey string
	DialControl   ssrf.Control
}

// ProviderConfigView is the sanitized provider config returned to admins, with the API key dummy-masked.
type ProviderConfigView struct {
	models.ProviderConfig
	HasAPIKey bool `json:"has_api_key"`
}

type queries struct {
	GetProviderByType           *sqlx.Stmt `query:"get-provider-by-type"`
	UpdateProviderConfig        *sqlx.Stmt `query:"update-provider-config"`
	SetCompletionKey            *sqlx.Stmt `query:"set-completion-key"`
	GetPrompt                   *sqlx.Stmt `query:"get-prompt"`
	GetPrompts                  *sqlx.Stmt `query:"get-prompts"`
	GetKnowledgeBaseItems       *sqlx.Stmt `query:"get-knowledge-base-items"`
	GetKnowledgeBaseItem        *sqlx.Stmt `query:"get-knowledge-base-item"`
	InsertKnowledgeBaseItem     *sqlx.Stmt `query:"insert-knowledge-base-item"`
	UpdateKnowledgeBaseItem     *sqlx.Stmt `query:"update-knowledge-base-item"`
	DeleteKnowledgeBaseItem     *sqlx.Stmt `query:"delete-knowledge-base-item"`
	SetKnowledgeBaseFingerprint *sqlx.Stmt `query:"set-knowledge-base-embedded-fingerprint"`
	InsertEmbedding             *sqlx.Stmt `query:"insert-embedding"`
	DeleteEmbeddingsBySource    *sqlx.Stmt `query:"delete-embeddings-by-source"`
	GetAllEmbeddings            *sqlx.Stmt `query:"get-all-embeddings"`
	GetTools                    *sqlx.Stmt `query:"get-tools"`
	GetTool                     *sqlx.Stmt `query:"get-tool"`
	GetEnabledTools             *sqlx.Stmt `query:"get-enabled-tools"`
	GetToolAuth                 *sqlx.Stmt `query:"get-tool-auth"`
	InsertTool                  *sqlx.Stmt `query:"insert-tool"`
	UpdateTool                  *sqlx.Stmt `query:"update-tool"`
	DeleteTool                  *sqlx.Stmt `query:"delete-tool"`
	GetCopilotMessages          *sqlx.Stmt `query:"get-copilot-messages"`
	InsertCopilotMessage        *sqlx.Stmt `query:"insert-copilot-message"`
	DeleteCopilotMessages       *sqlx.Stmt `query:"delete-copilot-messages"`
}

// New creates and returns a new instance of the Manager.
func New(opts Opts) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}
	m := &Manager{
		q:             q,
		db:            opts.DB,
		lo:            opts.Lo,
		i18n:          opts.I18n,
		encryptionKey: opts.EncryptionKey,
		chunkCfg:      stringutil.DefaultChunkConfig(),
		index:         newEmbeddingIndex(),
		snippetGen:    make(map[int]uint64),
		dialControl:   opts.DialControl,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   5 * time.Second,
					KeepAlive: 30 * time.Second,
					Control:   opts.DialControl,
				}).DialContext,
				ForceAttemptHTTP2: true,
			},
		},
	}
	m.chunkCfg.Logger = opts.Lo
	m.chunkCfg.TokenizerFunc = newTokenCounter(opts.Lo)
	// Search tolerates an empty index until this finishes.
	go func() {
		if err := m.loadIndex(); err != nil {
			m.lo.Error("error loading embeddings index at boot", "error", err)
		}
	}()
	return m, nil
}

// Completion runs the DB-stored prompt (by key) over the user content.
func (m *Manager) Completion(ctx context.Context, k string, prompt string) (string, error) {
	systemPrompt, err := m.getPrompt(k)
	if err != nil {
		return "", err
	}

	client, err := m.getProviderClient(models.ProviderTypeCompletion)
	if err != nil {
		return "", err
	}

	response, err := client.SendPrompt(ctx, models.PromptPayload{SystemPrompt: systemPrompt, UserPrompt: prompt})
	if err != nil {
		return "", m.providerError(err)
	}
	return response, nil
}

// CompletionRaw runs an ad-hoc system+user prompt (no DB-stored prompt) and returns the text.
func (m *Manager) CompletionRaw(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	client, err := m.getProviderClient(models.ProviderTypeCompletion)
	if err != nil {
		return "", err
	}
	response, err := client.SendPrompt(ctx, models.PromptPayload{SystemPrompt: systemPrompt, UserPrompt: userPrompt})
	if err != nil {
		return "", m.providerError(err)
	}
	return response, nil
}

// GetEmbeddings returns the embedding vector for text using the embedding provider.
func (m *Manager) GetEmbeddings(ctx context.Context, text string) ([]float32, error) {
	client, err := m.getProviderClient(models.ProviderTypeEmbedding)
	if err != nil {
		return nil, err
	}
	vec, err := client.GetEmbeddings(ctx, text)
	if err != nil {
		return nil, m.providerError(err)
	}
	return vec, nil
}

// GetEmbeddingsBatch returns embedding vectors for all texts in a single provider request.
func (m *Manager) GetEmbeddingsBatch(ctx context.Context, texts []string) ([][]float32, error) {
	client, err := m.getProviderClient(models.ProviderTypeEmbedding)
	if err != nil {
		return nil, err
	}
	vecs, err := client.GetEmbeddingsBatch(ctx, texts)
	if err != nil {
		return nil, m.providerError(err)
	}
	return vecs, nil
}

// GetPrompts returns the list of agent-facing prompts.
func (m *Manager) GetPrompts() ([]models.Prompt, error) {
	prompts := make([]models.Prompt, 0)
	if err := m.q.GetPrompts.Select(&prompts); err != nil {
		m.lo.Error("error fetching prompts", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return prompts, nil
}

// GetProviderConfig returns the sanitized config for a provider type (no API key).
func (m *Manager) GetProviderConfig(providerType string) (ProviderConfigView, error) {
	cfg, err := m.getProviderConfig(providerType)
	if err != nil {
		return ProviderConfigView{}, err
	}
	view := ProviderConfigView{ProviderConfig: cfg, HasAPIKey: cfg.APIKey != ""}
	if cfg.APIKey != "" {
		view.APIKey = strings.Repeat(stringutil.PasswordDummy, 10)
	}
	return view, nil
}

// VisionEnabled reports whether the completion model is marked as accepting image input.
func (m *Manager) VisionEnabled() bool {
	cfg, err := m.getRawProviderConfig(models.ProviderTypeCompletion)
	if err != nil {
		return false
	}
	return cfg.Vision
}

// UpdateProviderConfig updates a provider type's config; a blank api_key keeps the stored key.
func (m *Manager) UpdateProviderConfig(providerType string, in models.ProviderConfig) error {
	if providerType != models.ProviderTypeCompletion && providerType != models.ProviderTypeEmbedding {
		return envelope.NewError(envelope.InputError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	existing, err := m.getRawProviderConfig(providerType)
	if err != nil {
		return err
	}

	apiKey := existing.APIKey
	if in.APIKey != "" && !strings.Contains(in.APIKey, stringutil.PasswordDummy) {
		enc, eerr := crypto.Encrypt(in.APIKey, m.encryptionKey)
		if eerr != nil {
			m.lo.Error("error encrypting API key", "error", eerr)
			return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
		}
		apiKey = enc
	}

	cfg := models.ProviderConfig{
		Provider:           "openai",
		BaseURL:            in.BaseURL,
		APIKey:             apiKey,
		Model:              in.Model,
		Temperature:        in.Temperature,
		MaxTokens:          in.MaxTokens,
		Dimensions:         in.Dimensions,
		EmbeddingMaxTokens: in.EmbeddingMaxTokens,
		Instructions:       in.Instructions,
		Vision:             in.Vision,
		ReasoningEffort:    in.ReasoningEffort,
	}
	b, err := json.Marshal(cfg)
	if err != nil {
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if _, err := m.q.UpdateProviderConfig.Exec(providerType, b); err != nil {
		m.lo.Error("error updating provider config", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	// A changed embedding model produces vectors incomparable to the stored ones.
	if providerType == models.ProviderTypeEmbedding && (existing.Model != cfg.Model || existing.Dimensions != cfg.Dimensions) {
		m.lo.Info("embedding model changed, reindexing knowledge base", "old_model", existing.Model, "new_model", cfg.Model)
		m.ReindexAll()
	}
	return nil
}

// TestProviderConfig makes one live provider request with the given config; a blank or masked api_key uses the stored key.
func (m *Manager) TestProviderConfig(providerType string, in models.ProviderConfig) error {
	if providerType != models.ProviderTypeCompletion && providerType != models.ProviderTypeEmbedding {
		return envelope.NewError(envelope.InputError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	cfg := in
	cfg.Provider = "openai"
	if cfg.APIKey == "" || strings.Contains(cfg.APIKey, stringutil.PasswordDummy) {
		stored, err := m.getProviderConfig(providerType)
		if err != nil {
			return err
		}
		cfg.APIKey = stored.APIKey
	}
	client := NewOpenAIClient(cfg, m.lo, m.dialControl)

	if providerType == models.ProviderTypeCompletion {
		if _, err := client.SendPrompt(context.Background(), models.PromptPayload{SystemPrompt: "You are a connection test.", UserPrompt: "Reply with OK."}); err != nil {
			return m.testProviderError(err)
		}
		return nil
	}

	vec, err := client.GetEmbeddings(context.Background(), "connection test")
	if err != nil {
		return m.testProviderError(err)
	}
	if cfg.Dimensions > 0 && len(vec) != cfg.Dimensions {
		return envelope.NewError(envelope.InputError, m.i18n.Ts("ai.testDimensionsMismatch",
			"configured", strconv.Itoa(cfg.Dimensions), "returned", strconv.Itoa(len(vec))), nil)
	}
	return nil
}

// UpdateProvider sets the completion provider API key.
func (m *Manager) UpdateProvider(provider, apiKey string) error {
	if provider != "openai" {
		return envelope.NewError(envelope.InputError, m.i18n.T("validation.invalidProvider"), nil)
	}
	encryptedKey, err := crypto.Encrypt(apiKey, m.encryptionKey)
	if err != nil {
		m.lo.Error("error encrypting API key", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if _, err := m.q.SetCompletionKey.Exec(encryptedKey); err != nil {
		m.lo.Error("error setting completion API key", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

func (m *Manager) getPrompt(k string) (string, error) {
	var p models.Prompt
	if err := m.q.GetPrompt.Get(&p, k); err != nil {
		if err == sql.ErrNoRows {
			return "", envelope.NewError(envelope.InputError, m.i18n.T("validation.notFoundTemplate"), nil)
		}
		m.lo.Error("error fetching prompt", "error", err)
		return "", envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return p.Content, nil
}

// getRawProviderConfig returns the stored config with the API key still encrypted.
func (m *Manager) getRawProviderConfig(providerType string) (models.ProviderConfig, error) {
	var p models.Provider
	if err := m.q.GetProviderByType.Get(&p, providerType); err != nil {
		m.lo.Error("error fetching provider", "error", err, "type", providerType)
		return models.ProviderConfig{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	var cfg models.ProviderConfig
	if len(p.Config) > 0 {
		if err := json.Unmarshal(p.Config, &cfg); err != nil {
			m.lo.Error("error parsing provider config", "error", err)
			return models.ProviderConfig{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
		}
	}
	return cfg, nil
}

func (m *Manager) getProviderConfig(providerType string) (models.ProviderConfig, error) {
	cfg, err := m.getRawProviderConfig(providerType)
	if err != nil {
		return cfg, err
	}
	if cfg.APIKey != "" {
		decrypted, err := crypto.Decrypt(cfg.APIKey, m.encryptionKey)
		if err != nil {
			m.lo.Error("error decrypting API key", "error", err)
			return models.ProviderConfig{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
		}
		cfg.APIKey = decrypted
	}
	return cfg, nil
}

func (m *Manager) getProviderClient(providerType string) (ProviderClient, error) {
	cfg, err := m.getProviderConfig(providerType)
	if err != nil {
		return nil, err
	}
	return NewOpenAIClient(cfg, m.lo, m.dialControl), nil
}

func (m *Manager) providerError(err error) error {
	if errors.Is(err, ErrInvalidAPIKey) {
		m.lo.Error("invalid provider API key")
		return envelope.NewError(envelope.InputError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if errors.Is(err, ErrApiKeyNotSet) {
		return envelope.NewError(envelope.InputError, m.i18n.Ts("ai.apiKeyNotSet", "provider", "OpenAI"), nil)
	}
	m.lo.Error("error from AI provider", "error", err)
	return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
}

// testProviderError surfaces the provider's own error message so the admin can act on it.
func (m *Manager) testProviderError(err error) error {
	if errors.Is(err, ErrApiKeyNotSet) {
		return envelope.NewError(envelope.InputError, m.i18n.Ts("ai.apiKeyNotSet", "provider", "OpenAI"), nil)
	}
	msg := err.Error()
	if len(msg) > maxTestErrorLen {
		msg = msg[:maxTestErrorLen] + "…"
	}
	return envelope.NewError(envelope.InputError, msg, nil)
}
