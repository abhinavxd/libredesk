// Package ai manages AI prompts and integrates with LLM providers.
package ai

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"time"

	"github.com/abhinavxd/libredesk/internal/ai/models"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	hcmodels "github.com/abhinavxd/libredesk/internal/helpcenter/models"
	mmodels "github.com/abhinavxd/libredesk/internal/media/models"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/pgvector/pgvector-go"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS

	ErrInvalidAPIKey = errors.New("invalid API Key")
	ErrApiKeyNotSet  = errors.New("api Key not set")

	ErrCustomAnswerNotFound = errors.New("custom answer not found")
)

type ConversationStore interface {
	SendReply(media []mmodels.Media, inboxID, senderID, contactID int, conversationUUID, content string, to, cc, bcc []string, metaMap map[string]any) error
	GetConversationMessages(conversationUUID string, types []string, privateMsgs *bool, page, pageSize int) ([]cmodels.Message, int, error)
	RemoveConversationAssignee(uuid, typ string, actor umodels.User) error
	UpdateConversationTeamAssignee(uuid string, teamID int, actor umodels.User) error
	UpdateConversationStatus(uuid string, statusID int, status, snoozeDur string, actor umodels.User) error
}

type HelpCenterStore interface {
	SearchKnowledgeBase(helpCenterID int, query string, locale string, threshold float64, limit int) ([]hcmodels.KnowledgeBaseResult, error)
}

type UserStore interface {
	GetAIAssistant(id int) (umodels.User, error)
}

type Manager struct {
	q                              queries
	db                             *sqlx.DB
	lo                             *logf.Logger
	i18n                           *i18n.I18n
	embeddingCfg                   EmbeddingConfig
	completionCfg                  CompletionConfig
	workerCfg                      WorkerConfig
	conversationCompletionsService *ConversationCompletionsService
	helpCenterStore                HelpCenterStore
}

type EmbeddingConfig struct {
	Provider string        `json:"provider"`
	URL      string        `json:"url"`
	APIKey   string        `json:"api_key"`
	Model    string        `json:"model"`
	Timeout  time.Duration `json:"timeout"`
}

type CompletionConfig struct {
	Provider    string        `json:"provider"`
	URL         string        `json:"url"`
	APIKey      string        `json:"api_key"`
	Model       string        `json:"model"`
	Timeout     time.Duration `json:"timeout"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
}

type WorkerConfig struct {
	Workers  int `json:"workers"`
	Capacity int `json:"capacity"`
}

// Opts contains options for initializing the Manager.
type Opts struct {
	DB   *sqlx.DB
	I18n *i18n.I18n
	Lo   *logf.Logger
}

// queries contains prepared SQL queries.
type queries struct {
	GetDefaultProvider *sqlx.Stmt `query:"get-default-provider"`
	GetPrompt          *sqlx.Stmt `query:"get-prompt"`
	GetPrompts         *sqlx.Stmt `query:"get-prompts"`
	SetOpenAIKey       *sqlx.Stmt `query:"set-openai-key"`
	// Custom Answers
	GetAICustomAnswers   *sqlx.Stmt `query:"get-ai-custom-answers"`
	GetAICustomAnswer    *sqlx.Stmt `query:"get-ai-custom-answer"`
	InsertAICustomAnswer *sqlx.Stmt `query:"insert-ai-custom-answer"`
	UpdateAICustomAnswer *sqlx.Stmt `query:"update-ai-custom-answer"`
	DeleteAICustomAnswer *sqlx.Stmt `query:"delete-ai-custom-answer"`
	// AI Search Functions
	SearchCustomAnswers *sqlx.Stmt `query:"search-custom-answers"`
}

// New creates and returns a new instance of the Manager.
func New(embeddingCfg EmbeddingConfig, completionCfg CompletionConfig, workerCfg WorkerConfig, conversationStore ConversationStore, helpCenterStore HelpCenterStore, userStore UserStore, opts Opts) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}

	manager := &Manager{
		q:               q,
		db:              opts.DB,
		lo:              opts.Lo,
		i18n:            opts.I18n,
		embeddingCfg:    embeddingCfg,
		completionCfg:   completionCfg,
		workerCfg:       workerCfg,
		helpCenterStore: helpCenterStore,
	}

	// Initialize conversation completions service
	manager.conversationCompletionsService = NewConversationCompletionsService(
		manager,
		conversationStore,
		helpCenterStore,
		userStore,
		workerCfg.Workers,
		workerCfg.Capacity,
		opts.Lo,
	)

	return manager, nil
}

// GetEmbeddings returns embeddings for the given text using the configured provider.
func (m *Manager) GetEmbeddings(text string) ([]float32, error) {
	client, err := m.getProviderClient(true)
	if err != nil {
		m.lo.Error("error getting provider client", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", m.i18n.Ts("globals.terms.provider")), nil)
	}

	embedding, err := client.GetEmbeddings(text)
	if err != nil {
		m.lo.Error("error sending embedding request", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, err.Error(), nil)
	}

	return embedding, nil
}

// Completion sends a prompt to the default provider and returns the response.
func (m *Manager) Completion(k string, prompt string) (string, error) {
	systemPrompt, err := m.getPrompt(k)
	if err != nil {
		return "", err
	}

	client, err := m.getProviderClient(false)
	if err != nil {
		m.lo.Error("error getting provider client", "error", err)
		return "", envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", m.i18n.Ts("globals.terms.provider")), nil)
	}

	payload := models.PromptPayload{
		SystemPrompt: systemPrompt,
		UserPrompt:   prompt,
	}

	response, err := client.SendPrompt(payload)
	if err != nil {
		return "", m.handleProviderError(" for prompt", err)
	}

	return response, nil
}

// ChatCompletion sends a chat completion request with message history to the configured provider.
func (m *Manager) ChatCompletion(messages []models.ChatMessage) (string, error) {
	client, err := m.getProviderClient(false)
	if err != nil {
		m.lo.Error("error getting provider client for chat completion", "error", err)
		return "", envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", m.i18n.Ts("globals.terms.provider")), nil)
	}

	payload := models.ChatCompletionPayload{
		Messages: messages,
	}

	response, err := client.SendChatCompletion(payload)
	if err != nil {
		return "", m.handleProviderError(" for chat completion", err)
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

// UpdateProvider updates a provider.
func (m *Manager) UpdateProvider(provider, apiKey string) error {
	switch ProviderType(provider) {
	case ProviderOpenAI:
		return m.setOpenAIAPIKey(apiKey)
	default:
		m.lo.Error("unsupported provider type", "provider", provider)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.invalid", "name", m.i18n.Ts("globals.terms.provider")), nil)
	}
}

// setOpenAIAPIKey sets the OpenAI API key in the database.
func (m *Manager) setOpenAIAPIKey(apiKey string) error {
	if _, err := m.q.SetOpenAIKey.Exec(apiKey); err != nil {
		m.lo.Error("error setting OpenAI API key", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "OpenAI API Key"), nil)
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

// getProviderClient returns a ProviderClient for the configured provider.
func (m *Manager) getProviderClient(isEmbedding bool) (ProviderClient, error) {
	var (
		cfg         EmbeddingConfig
		maxTokens   int
		temperature float64
	)
	if isEmbedding {
		cfg = m.embeddingCfg
	} else {
		cfg = EmbeddingConfig{
			Provider: m.completionCfg.Provider,
			URL:      m.completionCfg.URL,
			APIKey:   m.completionCfg.APIKey,
			Model:    m.completionCfg.Model,
			Timeout:  m.completionCfg.Timeout,
		}
		maxTokens = m.completionCfg.MaxTokens
		temperature = m.completionCfg.Temperature
	}

	if ProviderType(cfg.Provider) == ProviderOpenAI {
		return NewOpenAIClient(cfg.APIKey, cfg.Model, cfg.URL, temperature, maxTokens, cfg.Timeout, m.lo), nil
	}

	m.lo.Error("unsupported provider type", "provider", cfg.Provider)
	return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.invalid", "name", m.i18n.Ts("globals.terms.provider")), nil)
}

// StartConversationCompletions starts the conversation completions service
func (m *Manager) StartConversationCompletions() {
	if m.conversationCompletionsService != nil {
		m.conversationCompletionsService.Start()
	}
}

// StopConversationCompletions stops the conversation completions service
func (m *Manager) StopConversationCompletions() {
	if m.conversationCompletionsService != nil {
		m.conversationCompletionsService.Stop()
	}
}

// EnqueueConversationCompletion adds a conversation completion request to the queue
func (m *Manager) EnqueueConversationCompletion(req models.ConversationCompletionRequest) error {
	if m.conversationCompletionsService == nil {
		return fmt.Errorf("conversation completions service not initialized")
	}
	return m.conversationCompletionsService.EnqueueRequest(req)
}

// handleProviderError handles errors from the provider.
func (m *Manager) handleProviderError(context string, err error) error {
	if errors.Is(err, ErrInvalidAPIKey) {
		m.lo.Error("error invalid API key"+context, "error", err)
		return envelope.NewError(envelope.InputError, m.i18n.Ts("globals.messages.invalid", "name", "OpenAI API Key"), nil)
	}
	if errors.Is(err, ErrApiKeyNotSet) {
		m.lo.Error("error API key not set"+context, "error", err)
		return envelope.NewError(envelope.InputError, m.i18n.Ts("ai.apiKeyNotSet", "provider", "OpenAI"), nil)
	}
	m.lo.Error("error sending"+context+" to provider", "error", err)
	return envelope.NewError(envelope.GeneralError, err.Error(), nil)
}

// Custom Answers CRUD

// GetAICustomAnswers returns all AI custom answers
func (m *Manager) GetAICustomAnswers() ([]models.CustomAnswer, error) {
	var customAnswers []models.CustomAnswer
	if err := m.q.GetAICustomAnswers.Select(&customAnswers); err != nil {
		m.lo.Error("error fetching custom answers", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "custom answers"), nil)
	}
	return customAnswers, nil
}

// GetAICustomAnswer returns a specific AI custom answer by ID
func (m *Manager) GetAICustomAnswer(id int) (models.CustomAnswer, error) {
	var customAnswer models.CustomAnswer
	if err := m.q.GetAICustomAnswer.Get(&customAnswer, id); err != nil {
		if err == sql.ErrNoRows {
			return customAnswer, envelope.NewError(envelope.NotFoundError, m.i18n.Ts("globals.messages.notFound", "name", "custom answer"), nil)
		}
		m.lo.Error("error fetching custom answer", "error", err, "id", id)
		return customAnswer, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "custom answer"), nil)
	}
	return customAnswer, nil
}

// CreateAICustomAnswer creates a new AI custom answer with embeddings
func (m *Manager) CreateAICustomAnswer(question, answer string, enabled bool) (models.CustomAnswer, error) {
	// Generate embeddings for the question
	embedding, err := m.GetEmbeddings(question)
	if err != nil {
		m.lo.Error("error generating embeddings for custom answer", "error", err, "question", question)
		return models.CustomAnswer{}, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorCreating", "name", "custom answer"), nil)
	}

	// Convert []float32 to pgvector.Vector for PostgreSQL
	vector := pgvector.NewVector(embedding)

	var customAnswer models.CustomAnswer
	if err := m.q.InsertAICustomAnswer.Get(&customAnswer, question, answer, vector, enabled); err != nil {
		m.lo.Error("error creating custom answer", "error", err, "question", question)
		return customAnswer, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorCreating", "name", "custom answer"), nil)
	}

	m.lo.Info("custom answer created successfully", "id", customAnswer.ID, "question", question)
	return customAnswer, nil
}

// UpdateAICustomAnswer updates an existing AI custom answer
func (m *Manager) UpdateAICustomAnswer(id int, question, answer string, enabled bool) (models.CustomAnswer, error) {
	// Generate embeddings for the updated question
	embedding, err := m.GetEmbeddings(question)
	if err != nil {
		m.lo.Error("error generating embeddings for custom answer update", "error", err, "id", id, "question", question)
		return models.CustomAnswer{}, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "custom answer"), nil)
	}

	// Convert []float32 to pgvector.Vector for PostgreSQL
	vector := pgvector.NewVector(embedding)

	var customAnswer models.CustomAnswer
	if err := m.q.UpdateAICustomAnswer.Get(&customAnswer, id, question, answer, vector, enabled); err != nil {
		if err == sql.ErrNoRows {
			return customAnswer, envelope.NewError(envelope.NotFoundError, m.i18n.Ts("globals.messages.notFound", "name", "custom answer"), nil)
		}
		m.lo.Error("error updating custom answer", "error", err, "id", id)
		return customAnswer, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "custom answer"), nil)
	}

	m.lo.Info("custom answer updated successfully", "id", id, "question", question)
	return customAnswer, nil
}

// DeleteAICustomAnswer deletes an AI custom answer
func (m *Manager) DeleteAICustomAnswer(id int) error {
	result, err := m.q.DeleteAICustomAnswer.Exec(id)
	if err != nil {
		m.lo.Error("error deleting custom answer", "error", err, "id", id)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorDeleting", "name", "custom answer"), nil)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		m.lo.Error("error checking rows affected", "error", err, "id", id)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorDeleting", "name", "custom answer"), nil)
	}

	if rowsAffected == 0 {
		return envelope.NewError(envelope.NotFoundError, m.i18n.Ts("globals.messages.notFound", "name", "custom answer"), nil)
	}

	m.lo.Info("custom answer deleted successfully", "id", id)
	return nil
}

// SmartSearch performs a two-tier search: custom answers first, then knowledge base fallback
func (m *Manager) SmartSearch(helpCenterID int, query string, locale string) (any, error) {
	const (
		// TODO: These can be made configurable?
		customAnswerThreshold  = 0.85 // High confidence for custom answers
		knowledgeBaseThreshold = 0.25 // Lower threshold for knowledge base
		maxResults             = 6
	)

	// Step 1: Search custom answers with high confidence
	customAnswer, err := m.searchCustomAnswers(query, customAnswerThreshold)
	if err != nil && err != ErrCustomAnswerNotFound {
		return nil, err
	}

	// If we found a high-confidence custom answer, return it
	if customAnswer != nil {
		m.lo.Info("found high-confidence custom answer", "similarity", customAnswer.Similarity, "query", query)
		return customAnswer, nil
	}

	// Step 2: Search knowledge base with lower threshold
	knowledgeResults, err := m.searchKnowledgeBase(helpCenterID, query, locale, knowledgeBaseThreshold, maxResults)
	if err != nil {
		return nil, err
	}

	if len(knowledgeResults) > 0 {
		m.lo.Info("found knowledge base results", "count", len(knowledgeResults), "top_similarity", knowledgeResults[0].Similarity, "query", query)
		return knowledgeResults, nil
	}

	// No results found
	m.lo.Info("no results found in smart search", "query", query, "help_center_id", helpCenterID)
	return []models.KnowledgeBaseResult{}, nil
}

// searchCustomAnswers searches for custom answers with high confidence threshold
func (m *Manager) searchCustomAnswers(query string, threshold float64) (*models.CustomAnswerResult, error) {
	// Generate embeddings for the search query
	embedding, err := m.GetEmbeddings(query)
	if err != nil {
		m.lo.Error("error generating embeddings for custom answer search", "error", err, "query", query)
		return nil, fmt.Errorf("generating embeddings for custom answer search: %w", err)
	}

	var result models.CustomAnswerResult
	// Convert []float32 to pgvector.Vector for PostgreSQL
	vector := pgvector.NewVector(embedding)
	if err = m.q.SearchCustomAnswers.Get(&result, vector, threshold); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrCustomAnswerNotFound
		}
		m.lo.Error("error searching custom answers", "error", err, "query", query)
		return nil, fmt.Errorf("searching custom answers: %w", err)
	}

	return &result, nil
}

// searchKnowledgeBase searches knowledge base (articles) with the specified threshold and limit.
func (m *Manager) searchKnowledgeBase(helpCenterID int, query string, locale string, threshold float64, limit int) ([]models.KnowledgeBaseResult, error) {
	// Use the helpcenter store to perform the search
	hcResults, err := m.helpCenterStore.SearchKnowledgeBase(helpCenterID, query, locale, threshold, limit)
	if err != nil {
		return nil, err
	}

	// Convert help center results to our KnowledgeBaseResult format
	results := make([]models.KnowledgeBaseResult, len(hcResults))
	for i, hcResult := range hcResults {
		results[i] = models.KnowledgeBaseResult{
			SourceType:   hcResult.SourceType,
			SourceID:     hcResult.SourceID,
			Title:        hcResult.Title,
			Content:      hcResult.Content,
			HelpCenterID: hcResult.HelpCenterID,
			Similarity:   hcResult.Similarity,
		}
	}

	return results, nil
}
