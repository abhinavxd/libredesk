// Package ai manages AI prompts and integrates with LLM providers.
package ai

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/abhinavxd/libredesk/internal/ai/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	hcmodels "github.com/abhinavxd/libredesk/internal/helpcenter/models"
	mmodels "github.com/abhinavxd/libredesk/internal/media/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/pgvector/pgvector-go"
	"github.com/zerodha/logf"
)

const (
	maxPendingRequestsPerConversation = 2
)

var (
	//go:embed queries.sql
	efs embed.FS

	ErrInvalidAPIKey             = errors.New("invalid API Key")
	ErrApiKeyNotSet              = errors.New("api Key not set")
	ErrKnowledgeBaseItemNotFound = errors.New("knowledge base item not found")
)

type ConversationStore interface {
	SendReply(media []mmodels.Media, inboxID, senderID, contactID int, conversationUUID, content string, to, cc, bcc []string, metaMap map[string]any) error
	RemoveConversationAssignee(uuid, typ string, actor umodels.User) error
	UpdateConversationTeamAssignee(uuid string, teamID int, actor umodels.User) error
	UpdateConversationStatus(uuid string, statusID int, status, snoozeDur string, actor umodels.User) error
}

type HelpCenterStore interface {
	SearchKnowledgeBase(helpCenterID int, query string, locale string, threshold float64, limit int) ([]hcmodels.KnowledgeBaseResult, error)
	GetHelpCenterByID(id int) (hcmodels.HelpCenter, error)
}

type Manager struct {
	q                              queries
	db                             *sqlx.DB
	lo                             *logf.Logger
	i18n                           *i18n.I18n
	embeddingCfg                   EmbeddingConfig
	chunkingCfg                    ChunkingConfig
	completionCfg                  CompletionConfig
	workerCfg                      WorkerConfig
	conversationCompletionsService *ConversationCompletionsService
	helpCenterStore                HelpCenterStore
	pendingRequests                sync.Map // conversationUUID -> *atomic.Int64
}

type EmbeddingConfig struct {
	Provider string        `json:"provider"`
	URL      string        `json:"url"`
	APIKey   string        `json:"api_key"`
	Model    string        `json:"model"`
	Timeout  time.Duration `json:"timeout"`
}

type ChunkingConfig struct {
	MaxTokens     int `json:"max_tokens"`
	MinTokens     int `json:"min_tokens"`
	OverlapTokens int `json:"overlap_tokens"`
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
	GetPrompt                *sqlx.Stmt `query:"get-prompt"`
	GetPrompts               *sqlx.Stmt `query:"get-prompts"`
	SetOpenAIKey             *sqlx.Stmt `query:"set-openai-key"`
	GetKnowledgeBaseItems    *sqlx.Stmt `query:"get-knowledge-base-items"`
	GetKnowledgeBaseItem     *sqlx.Stmt `query:"get-knowledge-base-item"`
	InsertKnowledgeBaseItem  *sqlx.Stmt `query:"insert-knowledge-base-item"`
	UpdateKnowledgeBaseItem  *sqlx.Stmt `query:"update-knowledge-base-item"`
	DeleteKnowledgeBaseItem  *sqlx.Stmt `query:"delete-knowledge-base-item"`
	InsertEmbedding          *sqlx.Stmt `query:"insert-embedding"`
	DeleteEmbeddingsBySource *sqlx.Stmt `query:"delete-embeddings-by-source"`
	SearchKnowledgeBase      *sqlx.Stmt `query:"search-knowledge-base"`
}

// New creates and returns a new instance of the Manager.
func New(embeddingCfg EmbeddingConfig, chunkingCfg ChunkingConfig, completionCfg CompletionConfig, workerCfg WorkerConfig, conversationStore ConversationStore, helpCenterStore HelpCenterStore, opts Opts) (*Manager, error) {
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
		chunkingCfg:     chunkingCfg,
		completionCfg:   completionCfg,
		workerCfg:       workerCfg,
		helpCenterStore: helpCenterStore,
	}

	// Initialize conversation completions service
	manager.conversationCompletionsService = NewConversationCompletionsService(
		manager,
		conversationStore,
		helpCenterStore,
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
	// Clean up conversations from rate limiting map
	m.startCleanupWorker()
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

	// Check rate limit per conversation
	if !m.tryAcquireConversationSlot(req.ConversationUUID) {
		m.lo.Warn("AI completion request rate limited", "conversation_uuid", req.ConversationUUID)
		return nil
	}

	return m.conversationCompletionsService.EnqueueRequest(req)
}

// tryAcquireConversationSlot attempts to acquire a slot for AI completion for the given conversation.
// Returns true if slot was acquired, false if rate limit is reached, this prevents excessive enqueueing.
func (m *Manager) tryAcquireConversationSlot(conversationUUID string) bool {
	value, _ := m.pendingRequests.LoadOrStore(conversationUUID, &atomic.Int64{})
	counter := value.(*atomic.Int64)

	// Try to increment the counter
	newCount := counter.Add(1)
	if newCount > maxPendingRequestsPerConversation {
		// Rate limit exceeded, decrement back and return false
		counter.Add(-1)
		return false
	}

	return true
}

// releaseConversationSlot releases a slot for the given conversation when AI completion is done.
func (m *Manager) releaseConversationSlot(conversationUUID string) {
	if value, ok := m.pendingRequests.Load(conversationUUID); ok {
		counter := value.(*atomic.Int64)
		counter.Add(-1)
	}
}

// startCleanupWorker starts a background goroutine that cleans up inactive conversation entries every hour
func (m *Manager) startCleanupWorker() {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			var keysToDelete []any
			m.pendingRequests.Range(func(key, value any) bool {
				counter := value.(*atomic.Int64)
				if counter.Load() <= 0 {
					keysToDelete = append(keysToDelete, key)
				}
				return true
			})

			for _, key := range keysToDelete {
				m.pendingRequests.Delete(key)
			}

			if len(keysToDelete) > 0 {
				m.lo.Debug("AI rate limiter cleanup completed", "cleaned_conversations", len(keysToDelete))
			}
		}
	}()
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

// Knowledge Base CRUD

// GetKnowledgeBaseItems returns all knowledge base items
func (m *Manager) GetKnowledgeBaseItems() ([]models.KnowledgeBase, error) {
	var items = make([]models.KnowledgeBase, 0)
	if err := m.q.GetKnowledgeBaseItems.Select(&items); err != nil {
		m.lo.Error("error fetching knowledge base items", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "knowledge base items"), nil)
	}
	return items, nil
}

// GetKnowledgeBaseItem returns a specific knowledge base item by ID
func (m *Manager) GetKnowledgeBaseItem(id int) (models.KnowledgeBase, error) {
	var item models.KnowledgeBase
	if err := m.q.GetKnowledgeBaseItem.Get(&item, id); err != nil {
		if err == sql.ErrNoRows {
			return item, envelope.NewError(envelope.NotFoundError, m.i18n.Ts("globals.messages.notFound", "name", "knowledge base item"), nil)
		}
		m.lo.Error("error fetching knowledge base item", "error", err, "id", id)
		return item, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "knowledge base item"), nil)
	}
	return item, nil
}

// CreateKnowledgeBaseItem creates a new knowledge base item and generates embeddings using chunking
func (m *Manager) CreateKnowledgeBaseItem(itemType, content string, enabled bool) (models.KnowledgeBase, error) {
	// First, insert the knowledge base item for immediate availability
	var item models.KnowledgeBase
	if err := m.q.InsertKnowledgeBaseItem.Get(&item, itemType, content, enabled); err != nil {
		m.lo.Error("error creating knowledge base item", "error", err, "type", itemType)
		return item, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorCreating", "name", "knowledge base item"), nil)
	}

	m.lo.Info("knowledge base item created successfully", "id", item.ID, "type", itemType)

	// Generate embeddings asynchronously using chunking
	go m.processKnowledgeBaseContent(item.ID, content)

	return item, nil
}

// UpdateKnowledgeBaseItem updates an existing knowledge base item and regenerates embeddings
func (m *Manager) UpdateKnowledgeBaseItem(id int, itemType, content string, enabled bool) (models.KnowledgeBase, error) {
	// First, update the knowledge base item for immediate availability
	var item models.KnowledgeBase
	if err := m.q.UpdateKnowledgeBaseItem.Get(&item, id, itemType, content, enabled); err != nil {
		if err == sql.ErrNoRows {
			return item, envelope.NewError(envelope.NotFoundError, m.i18n.Ts("globals.messages.notFound", "name", "knowledge base item"), nil)
		}
		m.lo.Error("error updating knowledge base item", "error", err, "id", id)
		return item, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "knowledge base item"), nil)
	}

	m.lo.Info("knowledge base item updated successfully", "id", id, "type", itemType)

	// Delete old embeddings and regenerate new ones asynchronously
	go m.processKnowledgeBaseContent(id, content)

	return item, nil
}

// DeleteKnowledgeBaseItem deletes a knowledge base item and its embeddings
func (m *Manager) DeleteKnowledgeBaseItem(id int) error {
	// Delete embeddings first
	if _, err := m.q.DeleteEmbeddingsBySource.Exec("knowledge_base", id); err != nil {
		m.lo.Error("error deleting embeddings for knowledge base item", "error", err, "id", id)
		// Continue with deletion even if embedding deletion fails
	}

	// Delete the knowledge base item
	if _, err := m.q.DeleteKnowledgeBaseItem.Exec(id); err != nil {
		m.lo.Error("error deleting knowledge base item", "error", err, "id", id)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorDeleting", "name", "knowledge base item"), nil)
	}
	return nil
}

// SmartSearch performs unified search across knowledge base and help center articles
func (m *Manager) SmartSearch(helpCenterID int, query, locale string) ([]models.UnifiedKnowledgeResult, error) {
	const (
		// TODO: These can be made configurable?
		threshold  = 0.15
		maxResults = 8
	)

	// Search both knowledge base and help center concurrently with same threshold
	knowledgeBaseResults, err := m.searchKnowledgeBaseItems(query, threshold, maxResults)
	if err != nil && err != ErrKnowledgeBaseItemNotFound {
		return nil, err
	}

	helpCenterResults, err := m.searchHelpCenter(helpCenterID, query, locale, threshold, maxResults)
	if err != nil {
		return nil, err
	}

	// Combine results from both sources
	var allResults []models.UnifiedKnowledgeResult

	// Convert knowledge base results to UnifiedKnowledgeResult format
	for _, kb := range knowledgeBaseResults {
		allResults = append(allResults, models.UnifiedKnowledgeResult{
			SourceType:   "knowledge_base",
			SourceID:     kb.ID,
			Title:        "",
			Content:      kb.Content,
			HelpCenterID: nil, // Knowledge base items are not tied to help centers
			Similarity:   kb.Similarity,
		})
	}

	// Add help center results
	allResults = append(allResults, helpCenterResults...)

	if len(allResults) == 0 {
		m.lo.Info("no results found in smart search", "query", query)
		return []models.UnifiedKnowledgeResult{}, nil
	}

	// Sort all results by similarity score (highest first)
	sort.Slice(allResults, func(i, j int) bool {
		return allResults[i].Similarity > allResults[j].Similarity
	})

	// Limit to maxResults
	if len(allResults) > maxResults {
		allResults = allResults[:maxResults]
	}

	m.lo.Info("found unified search results", "count", len(allResults), "top_similarity", allResults[0].Similarity, "query", query)
	return allResults, nil
}

// searchKnowledgeBaseItems searches for knowledge base items with the specified threshold and limit
func (m *Manager) searchKnowledgeBaseItems(query string, threshold float64, limit int) ([]models.KnowledgeBaseResult, error) {
	// Generate embeddings for the search query
	embedding, err := m.GetEmbeddings(query)
	if err != nil {
		m.lo.Error("error generating embeddings for knowledge base search", "error", err, "query", query)
		return nil, fmt.Errorf("generating embeddings for knowledge base search: %w", err)
	}

	var results []models.KnowledgeBaseResult
	// Convert []float32 to pgvector.Vector for PostgreSQL
	vector := pgvector.NewVector(embedding)
	if err = m.q.SearchKnowledgeBase.Select(&results, vector, threshold, limit); err != nil {
		if err == sql.ErrNoRows {
			return []models.KnowledgeBaseResult{}, ErrKnowledgeBaseItemNotFound
		}
		m.lo.Error("error searching knowledge base", "error", err, "query", query)
		return nil, fmt.Errorf("searching knowledge base: %w", err)
	}

	return results, nil
}

// searchHelpCenter searches help center articles with the specified threshold and limit.
func (m *Manager) searchHelpCenter(helpCenterID int, query, locale string, threshold float64, limit int) ([]models.UnifiedKnowledgeResult, error) {
	hcResults, err := m.helpCenterStore.SearchKnowledgeBase(helpCenterID, query, locale, threshold, limit)
	if err != nil {
		return nil, err
	}

	// Convert help center results to our UnifiedKnowledgeResult format
	results := make([]models.UnifiedKnowledgeResult, len(hcResults))
	for i, hcResult := range hcResults {
		results[i] = models.UnifiedKnowledgeResult{
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

// GetChunkConfig returns the configured chunking configuration
func (m *Manager) GetChunkConfig() stringutil.ChunkConfig {
	return stringutil.ChunkConfig{
		MaxTokens:      m.chunkingCfg.MaxTokens,
		MinTokens:      m.chunkingCfg.MinTokens,
		OverlapTokens:  m.chunkingCfg.OverlapTokens,
		TokenizerFunc:  nil, // Use default tokenizer
		PreserveBlocks: []string{"pre", "code", "table"},
		Logger:         m.lo,
	}
}

// processKnowledgeBaseContent processes knowledge base content by chunking it and generating embeddings
// This function is designed to be called asynchronously to avoid blocking the main operation
func (m *Manager) processKnowledgeBaseContent(itemID int, content string) {
	// First, delete any existing embeddings for this item
	if _, err := m.q.DeleteEmbeddingsBySource.Exec("knowledge_base", itemID); err != nil {
		m.lo.Error("error deleting existing embeddings in background", "error", err, "item_id", itemID)
		// Continue with processing even if deletion fails
	}

	// Chunk the HTML content with configured parameters
	chunks, err := stringutil.ChunkHTMLContent("", content, m.GetChunkConfig())
	if err != nil {
		m.lo.Error("error chunking HTML content", "error", err, "item_id", itemID)
		return
	}

	if len(chunks) == 0 {
		m.lo.Warn("no chunks generated for knowledge base item", "item_id", itemID)
		return
	}

	// Process each chunk
	for i, chunk := range chunks {
		// Generate embeddings for the chunk text
		embedding, err := m.GetEmbeddings(chunk.Text)
		if err != nil {
			m.lo.Error("error generating embeddings for chunk in background", "error", err, "item_id", itemID, "chunk", i)
			continue // Skip this chunk but continue with others
		}

		// Convert []float32 to pgvector.Vector for PostgreSQL
		vector := pgvector.NewVector(embedding)

		// Create metadata for the chunk
		meta := fmt.Sprintf(`{"chunk_index": %d, "total_chunks": %d, "has_heading": %t, "has_code": %t, "has_table": %t}`,
			chunk.ChunkIndex, chunk.TotalChunks, chunk.HasHeading, chunk.HasCode, chunk.HasTable)

		m.lo.Debug("ai knowledge base chunk metadata", "item_id", itemID, "chunk", i, "metadata", meta)

		// Store the embedding in the centralized embeddings table
		if _, err := m.q.InsertEmbedding.Exec("knowledge_base", itemID, chunk.Text, vector, meta); err != nil {
			m.lo.Error("error storing embedding for chunk in background", "error", err, "item_id", itemID, "chunk", i)
			continue // Skip this chunk but continue with others
		}
	}
	m.lo.Info("knowledge base item embeddings processed successfully in background", "item_id", itemID, "chunks_processed", len(chunks))
}
