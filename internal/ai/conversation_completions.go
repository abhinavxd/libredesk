package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/abhinavxd/libredesk/internal/ai/models"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/zerodha/logf"
)

// systemPromptData holds the data for generating the AI system prompt
type systemPromptData struct {
	AssistantName      string
	ProductName        string
	ProductDescription string
	ToneInstruction    string
	LengthInstruction  string
	HandoffEnabled     bool
}

type queryRefinementResponse struct {
	OriginalLanguage string  `json:"original_language"`
	TranslatedQuery  string  `json:"translated_query"`
	RefinedQuery     string  `json:"refined_query"`
	ConfidenceScore  float64 `json:"confidence_score"`
}

type aiConversationResponse struct {
	Reasoning   string `json:"reasoning"`
	Response    string `json:"response"`
	UserMessage string `json:"user_message"`
}

// getToneInstruction returns the tone instruction based on the tone setting
func getToneInstruction(tone string) string {
	switch tone {
	case "neutral":
		return "Keep your tone neutral and straightforward."
	case "friendly":
		return "Keep your tone friendly and approachable, you can use emojis to enhance friendliness."
	case "professional":
		return "Keep your tone professional and formal."
	case "humorous":
		return "Keep your tone humorous and light-hearted, but still helpful. You can use emojis to enhance friendliness."
	default:
		return "Keep your tone friendly and approachable."
	}
}

// getLengthInstruction returns the length instruction based on the length setting
func getLengthInstruction(length string) string {
	switch length {
	case "concise":
		return "Keep responses very brief and to the point (1-2 sentences max)."
	case "medium":
		return "Keep messages under 5-6 sentences unless detailed steps are needed."
	case "long":
		return "Provide detailed, comprehensive responses with step-by-step instructions when helpful."
	default:
		return "Keep messages under 5-6 sentences unless detailed steps are needed."
	}
}

// ConversationCompletionsService handles AI-powered chat completions for customer support
type ConversationCompletionsService struct {
	lo                *logf.Logger
	manager           *Manager
	conversationStore ConversationStore
	helpCenterStore   HelpCenterStore
	requestQueue      chan models.ConversationCompletionRequest
	workers           int
	capacity          int
	wg                sync.WaitGroup
	ctx               context.Context
	cancel            context.CancelFunc
	closed            bool
	closedMu          sync.RWMutex
}

// NewConversationCompletionsService creates a new conversation completions service
func NewConversationCompletionsService(manager *Manager, conversationStore ConversationStore, helpCenterStore HelpCenterStore, workers, capacity int, lo *logf.Logger) *ConversationCompletionsService {
	ctx, cancel := context.WithCancel(context.Background())

	return &ConversationCompletionsService{
		lo:                lo,
		manager:           manager,
		conversationStore: conversationStore,
		helpCenterStore:   helpCenterStore,
		requestQueue:      make(chan models.ConversationCompletionRequest, capacity),
		workers:           workers,
		capacity:          capacity,
		ctx:               ctx,
		cancel:            cancel,
	}
}

// Start initializes and starts the worker pool
func (s *ConversationCompletionsService) Start() {
	for range s.workers {
		s.wg.Add(1)
		go s.worker()
	}
}

// Stop gracefully shuts down the service
func (s *ConversationCompletionsService) Stop() {
	s.closedMu.Lock()
	defer s.closedMu.Unlock()

	if s.closed {
		return
	}

	s.closed = true
	s.cancel()
	close(s.requestQueue)
	s.wg.Wait()
}

// EnqueueRequest adds a completion request to the queue
func (s *ConversationCompletionsService) EnqueueRequest(req models.ConversationCompletionRequest) error {
	s.closedMu.RLock()
	defer s.closedMu.RUnlock()

	if s.closed {
		// Release the slot since request can't be processed
		s.manager.releaseConversationSlot(req.ConversationUUID)
		return fmt.Errorf("conversation completions service is closed")
	}

	select {
	case s.requestQueue <- req:
		return nil
	default:
		// Release the slot since request is being dropped
		s.manager.releaseConversationSlot(req.ConversationUUID)
		s.lo.Warn("AI completion request queue is full, dropping request", "conversation_uuid", req.ConversationUUID)
		return fmt.Errorf("request queue is full")
	}
}

// buildSystemPrompt renders the final system prompt with tone and length instructions
func buildSystemPrompt(assistantName, productName, productDescription, tone, length string, handoffEnabled bool) (string, error) {
	data := systemPromptData{
		AssistantName:      assistantName,
		ProductName:        productName,
		ProductDescription: productDescription,
		ToneInstruction:    getToneInstruction(tone),
		LengthInstruction:  getLengthInstruction(length),
		HandoffEnabled:     handoffEnabled,
	}

	tmpl, err := template.New("systemPrompt").Parse(ConversationSystemPrompt)
	if err != nil {
		return "", fmt.Errorf("failed to parse system prompt template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute system prompt template: %w", err)
	}

	return buf.String(), nil
}

// worker processes completion requests from the queue
func (s *ConversationCompletionsService) worker() {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return
		case req, ok := <-s.requestQueue:
			if !ok {
				return
			}
			s.processCompletionRequest(req)
		}
	}
}

// processCompletionRequest handles a single completion request
func (s *ConversationCompletionsService) processCompletionRequest(req models.ConversationCompletionRequest) {
	var (
		messages        = req.Messages
		start           = time.Now()
		aiAssistantMeta umodels.AIAssistantMeta
	)
	s.lo.Info("processing AI completion request", "conversation_uuid", req.ConversationUUID)

	// Ensure we always release the conversation slot when done
	defer s.manager.releaseConversationSlot(req.ConversationUUID)

	if req.AIAssistant.ID == 0 {
		s.lo.Warn("AI assistant not found, skipping AI completion", "conversation_uuid", req.ConversationUUID)
		return
	}

	if err := json.Unmarshal(req.AIAssistant.Meta, &aiAssistantMeta); err != nil {
		s.lo.Error("error parsing AI assistant meta", "error", err, "assistant_id", req.AIAssistant.ID, "conversation_uuid", req.ConversationUUID, "meta", req.AIAssistant.Meta)
		return
	}

	if !req.AIAssistant.Enabled {
		s.lo.Warn("AI assistant is disabled, skipping AI completion", "assistant_id", req.AIAssistant.ID, "conversation_uuid", req.ConversationUUID)
		return
	}

	// Get the latest message from contact
	latestContactMessage := s.getLatestContactMessage(messages)

	// Build context from latest contact message
	context, err := s.buildSearchContext(req, latestContactMessage)
	if err != nil {
		s.lo.Error("error building help center context", "error", err, "conversation_uuid", req.ConversationUUID)
		// Continue without help center context
	}

	// Build chat messages array with proper roles
	chatMessages, err := s.buildChatMessages(context, messages, req.AIAssistant, aiAssistantMeta)
	if err != nil {
		s.lo.Error("failed to build chat messages", "error", err, "conversation_uuid", req.ConversationUUID)
		return
	}

	// Send AI completion request to the provider
	upstreamStartAt := time.Now()
	aiResponse, err := s.manager.ChatCompletion(chatMessages)
	if err != nil {
		s.lo.Error("error getting AI chat completion", "error", err, "conversation_uuid", req.ConversationUUID)
		return
	}

	// Log upstream processing time
	s.lo.Debug("AI chat completion upstream processing time", "conversation_uuid", req.ConversationUUID, "duration_ms", time.Since(upstreamStartAt).Milliseconds())

	// Process AI response
	var (
		handoffRequested bool
		resolved         bool
		finalResponse    string
		reasoning        string
	)

	// Try to parse as JSON first
	cleanedResponse := stringutil.CleanJSONResponse(aiResponse)
	var structuredResponse aiConversationResponse
	if err := json.Unmarshal([]byte(cleanedResponse), &structuredResponse); err != nil {
		// Fallback: treat as plain text response
		s.lo.Debug("AI response not in JSON format, using as plain text",
			"conversation_uuid", req.ConversationUUID,
			"response", aiResponse)
		finalResponse = strings.TrimSpace(aiResponse)
		reasoning = ""
	} else {
		// Successfully parsed JSON
		finalResponse = strings.TrimSpace(structuredResponse.Response)
		reasoning = strings.TrimSpace(structuredResponse.Reasoning)
		s.lo.Info("AI reasoning captured",
			"conversation_uuid", req.ConversationUUID,
			"reasoning", reasoning)
	}

	// Check for conversation handoff and resolution
	switch finalResponse {
	case "conversation_handoff":
		s.lo.Info("AI requested conversation handoff", "conversation_uuid", req.ConversationUUID)
		if structuredResponse.UserMessage != "" {
			finalResponse = structuredResponse.UserMessage
		} else {
			finalResponse = "Connecting you with one of our support agents who can better assist you."
		}
		handoffRequested = true
	case "conversation_resolve":
		s.lo.Info("AI requested conversation resolution", "conversation_uuid", req.ConversationUUID)
		finalResponse = ""
		resolved = true
	default:
		// Convert markdown to HTML for consistent formatting with TipTap editor output, since LLMs often use markdown for formatting in their responses.
		// Requesting HTML directly was not consistent.
		finalResponse = s.convertMarkdownToHTML(finalResponse)
	}

	// Send AI response
	if finalResponse != "" {
		// Prepare metadata with reasoning if available
		metaMap := map[string]any{
			"ai_generated":       true,
			"processing_time_ms": time.Since(start).Milliseconds(),
			"ai_model":           s.manager.completionCfg.Model,
			"ai_provider":        s.manager.completionCfg.Provider,
		}

		// Add reasoning if available
		if reasoning != "" {
			metaMap["ai_reasoning"] = reasoning
		}

		if err = s.conversationStore.SendReply(
			nil, // No media attachments for AI responses
			req.InboxID,
			req.AIAssistant.ID,
			req.ContactID,
			req.ConversationUUID,
			finalResponse,
			[]string{}, // to
			[]string{}, // cc
			[]string{}, // bcc
			metaMap,
		); err != nil {
			s.lo.Error("error sending AI response", "conversation_uuid", req.ConversationUUID, "error", err)
			return
		}
	}

	// If handoff is requested and enabled for this AI assistant, remove conversation assignee and optionally update team assignee if team ID is set
	if handoffRequested && aiAssistantMeta.HandOff {
		// First unassign the conversation from the AI assistant
		if err := s.conversationStore.RemoveConversationAssignee(req.ConversationUUID, "user", req.AIAssistant); err != nil {
			s.lo.Error("error removing conversation assignee", "conversation_uuid", req.ConversationUUID, "error", err)
		} else {
			s.lo.Info("conversation assignee removed for handoff", "conversation_uuid", req.ConversationUUID)
		}

		// Set the handoff team if specified
		if aiAssistantMeta.HandOffTeam > 0 {
			if err := s.conversationStore.UpdateConversationTeamAssignee(req.ConversationUUID, aiAssistantMeta.HandOffTeam, req.AIAssistant); err != nil {
				s.lo.Error("error updating conversation team assignee", "conversation_uuid", req.ConversationUUID, "team_id", aiAssistantMeta.HandOffTeam, "error", err)
			} else {
				s.lo.Info("conversation handoff to team", "conversation_uuid", req.ConversationUUID, "team_id", aiAssistantMeta.HandOffTeam)
			}
		}
	}

	// Resolve the conversation if requested
	if resolved {
		if err := s.conversationStore.UpdateConversationStatus(req.ConversationUUID, 0, cmodels.StatusResolved, "", req.AIAssistant); err != nil {
			s.lo.Error("error updating conversation status to resolved", "conversation_uuid", req.ConversationUUID, "error", err)
		} else {
			s.lo.Info("conversation marked as resolved", "conversation_uuid", req.ConversationUUID)
		}
	}

	// Log the reasoning if available
	if reasoning != "" {
		s.lo.Info("AI completion request processed successfully with reasoning",
			"conversation_uuid", req.ConversationUUID,
			"processing_time", time.Since(start),
			"response_length", len(finalResponse),
			"reasoning", reasoning,
			"has_reasoning", true)
	} else {
		s.lo.Info("AI completion request processed successfully",
			"conversation_uuid", req.ConversationUUID,
			"processing_time", time.Since(start),
			"response_length", len(finalResponse),
			"response_type", "plain_text",
			"has_reasoning", false)
	}
}

// getLatestContactMessage returns the text content of the latest contact message
func (s *ConversationCompletionsService) getLatestContactMessage(messages []cmodels.Message) string {
	for _, msg := range messages {
		if msg.SenderType == cmodels.SenderTypeContact {
			return msg.TextContent
		}
	}
	return ""
}

// buildSearchContext performs context-aware search across knowledge sources and builds context
func (s *ConversationCompletionsService) buildSearchContext(req models.ConversationCompletionRequest, latestContactMessage string) (string, error) {
	if s.helpCenterStore == nil {
		return "", nil
	}

	// Use the provided latest contact message as query
	if latestContactMessage == "" {
		return "", nil
	}

	// Get target language from help center's default locale
	locale := ""
	if req.HelpCenterID.Valid {
		helpCenter, err := s.helpCenterStore.GetHelpCenterByID(req.HelpCenterID.Int)
		if err != nil {
			s.lo.Error("error fetching help center for default locale", "error", err, "help_center_id", req.HelpCenterID.Int)
		} else {
			locale = helpCenter.DefaultLocale
		}
	}

	// Default fallback
	if locale == "" {
		s.lo.Warn("no help center locale found for completions, defaulting to English", "conversation_uuid", req.ConversationUUID)
		locale = "en"
	}

	// Attempt context-aware query refinement
	searchQuery := latestContactMessage
	var confidence float64 = 0.0

	refinementResponse, err := s.refineSearchQuery(latestContactMessage, locale, req.Messages)
	if err != nil {
		s.lo.Error("query refinement failed", "error", err, "original_query", latestContactMessage)
		return "", err
	} else {
		confidence = refinementResponse.ConfidenceScore
		// Use refined query if confidence is above threshold (0.7)
		if confidence >= 0.7 && refinementResponse.RefinedQuery != "" {
			searchQuery = refinementResponse.RefinedQuery
			s.lo.Info("using refined query for search",
				"original", latestContactMessage,
				"refined", refinementResponse.RefinedQuery,
				"confidence", confidence,
				"locale", locale)
		} else {
			// Low confidence refinement - use translated query if available
			if refinementResponse.TranslatedQuery != "" {
				searchQuery = refinementResponse.TranslatedQuery
				s.lo.Info("low confidence refinement, using translated query",
					"original", latestContactMessage,
					"translated", refinementResponse.TranslatedQuery,
					"refined", refinementResponse.RefinedQuery,
					"confidence", confidence)
			} else {
				// Both refinement and translation failed
				searchQuery = latestContactMessage
				s.lo.Warn("low confidence refinement and no translation, using original query",
					"original", latestContactMessage,
					"refined", refinementResponse.RefinedQuery,
					"confidence", confidence)
			}
		}
	}

	result, err := s.manager.SmartSearch(req.HelpCenterID.Int, searchQuery, locale)
	if err != nil {
		return "", err
	}

	if len(result) == 0 {
		s.lo.Warn("no relevant help center content found",
			"conversation_uuid", req.ConversationUUID,
			"original_query", latestContactMessage,
			"search_query", searchQuery,
			"confidence", confidence)
		return "", nil
	}

	// Build context based on unified search results
	var contextBuilder strings.Builder

	if len(result) > 0 {
		contextBuilder.WriteString("Relevant knowledge base content:\n\n")

		for i, item := range result {
			// Different handling for snippets vs articles
			if item.SourceType == "snippet" {
				contextBuilder.WriteString(fmt.Sprintf("%d. [SNIPPET] %s\n", i+1, item.Title))
			} else {
				contextBuilder.WriteString(fmt.Sprintf("%d. [ARTICLE] %s\n", i+1, item.Title))
			}
			if item.Content != "" {
				contextBuilder.WriteString(fmt.Sprintf("   %s\n\n", item.Content))
			}
		}

		s.lo.Info("found relevant help center content",
			"conversation_uuid", req.ConversationUUID,
			"results_count", len(result),
			"search_query", searchQuery,
			"refinement_confidence", confidence)

		return contextBuilder.String(), nil
	}

	return "", nil
}

// buildChatMessages creates a properly structured chat messages array for AI completion
func (s *ConversationCompletionsService) buildChatMessages(helpCenterContext string, messages []cmodels.Message, senderUser umodels.User, aiAssistantMeta umodels.AIAssistantMeta) ([]models.ChatMessage, error) {
	var chatMessages []models.ChatMessage

	// 1. Add system prompt with dynamic assistant name and product
	assistantName := "AI Assistant"
	productName := "our product"
	productDescription := ""
	answerTone := "friendly"
	answerLength := "medium"

	// Fallback to default values if not set
	if aiAssistantMeta.ProductName != "" {
		productName = aiAssistantMeta.ProductName
	}
	if aiAssistantMeta.AnswerTone != "" {
		answerTone = aiAssistantMeta.AnswerTone
	}
	if aiAssistantMeta.AnswerLength != "" {
		answerLength = aiAssistantMeta.AnswerLength
	}
	if aiAssistantMeta.ProductDescription != "" {
		productDescription = aiAssistantMeta.ProductDescription
	}
	if senderUser.FirstName != "" {
		assistantName = senderUser.FirstName
	}

	// Inject help center context into the system prompt if present
	systemPrompt, err := buildSystemPrompt(assistantName, productName, productDescription, answerTone, answerLength, aiAssistantMeta.HandOff)
	if err != nil {
		return nil, fmt.Errorf("failed to build system prompt: %w", err)
	}
	if helpCenterContext != "" {
		systemPrompt += "\n\nKnowledge base context (for reference):\n" + helpCenterContext
		systemPrompt += "\n\nNote: If the knowledge base content is in a different language than the customer's question, you may use it as reference but always respond in the customer's language."
	}

	chatMessages = append(chatMessages, models.ChatMessage{
		Role:    "system",
		Content: systemPrompt,
	})

	// 2. Add conversation history with proper roles
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]

		// Skip private messages
		if msg.Private {
			continue
		}

		role := "assistant"
		if msg.SenderType == cmodels.SenderTypeContact {
			role = "user"
		}

		chatMessages = append(chatMessages, models.ChatMessage{
			Role:    role,
			Content: msg.TextContent,
		})
	}

	return chatMessages, nil
}

// convertMarkdownToHTML converts markdown content to HTML using stringutil and removes single paragraph wrapping
func (s *ConversationCompletionsService) convertMarkdownToHTML(markdown string) string {
	htmlContent, err := stringutil.MarkdownToHTML(markdown)
	if err != nil {
		s.lo.Error("error converting markdown to HTML", "error", err, "markdown", markdown)
		// Return original markdown as fallback
		return markdown
	}

	// Remove wrapping <p> tags if the content is a single paragraph
	// This prevents double paragraph wrapping in the chat UI
	htmlContent = strings.TrimSpace(htmlContent)
	if strings.HasPrefix(htmlContent, "<p>") && strings.HasSuffix(htmlContent, "</p>") && strings.Count(htmlContent, "<p>") == 1 {
		htmlContent = htmlContent[3 : len(htmlContent)-4]
	}

	return htmlContent
}

// prepareConversationContext formats the last N messages for the LLM prompt
func (s *ConversationCompletionsService) prepareConversationContext(messages []cmodels.Message, maxMessages int, maxContentLength int) string {
	var contextBuilder strings.Builder

	// Get the last N messages (excluding the very latest one which is the current query)
	messageCount := 0
	for i := 1; i < len(messages) && messageCount < maxMessages; i++ {
		msg := messages[i]

		// Skip private messages
		if msg.Private {
			continue
		}

		// Determine role
		role := "ASSISTANT"
		if msg.SenderType == cmodels.SenderTypeContact {
			role = "USER"
		}

		// Truncate content if too long
		content := msg.TextContent
		if len(content) > maxContentLength {
			content = content[:maxContentLength] + "... [truncated]"
		}

		contextBuilder.WriteString(fmt.Sprintf("%s: %s\n", role, content))
		messageCount++
	}

	if contextBuilder.Len() == 0 {
		return "No prior conversation context."
	}

	return strings.TrimSpace(contextBuilder.String())
}

// refineSearchQuery performs context-aware query refinement using LLM
func (s *ConversationCompletionsService) refineSearchQuery(query, targetLanguage string, messages []cmodels.Message) (queryRefinementResponse, error) {
	// Prepare conversation context
	conversationContext := s.prepareConversationContext(messages, 3, 200)

	// Build the refinement prompt
	prompt := fmt.Sprintf(QueryRefinementPrompt, targetLanguage, targetLanguage, conversationContext, query)

	// Create chat messages for LLM call
	chatMessages := []models.ChatMessage{
		{
			Role:    "system",
			Content: QueryRefinementSystemMessage,
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	// Call LLM for refinement
	response, err := s.manager.ChatCompletion(chatMessages)
	if err != nil {
		s.lo.Error("error calling LLM for query refinement", "error", err, "query", query)
		return queryRefinementResponse{}, fmt.Errorf("LLM call failed: %w", err)
	}

	// Parse JSON response with safety net for markdown-wrapped responses
	cleanedResponse := stringutil.CleanJSONResponse(response)
	var refinementResponse queryRefinementResponse
	if err := json.Unmarshal([]byte(cleanedResponse), &refinementResponse); err != nil {
		s.lo.Error("error parsing LLM refinement response", "error", err,
			"original_response", response,
			"cleaned_response", cleanedResponse)
		return queryRefinementResponse{}, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	s.lo.Debug("query refinement completed",
		"original_query", query,
		"refined_query", refinementResponse.RefinedQuery,
		"translated_query", refinementResponse.TranslatedQuery,
		"confidence", refinementResponse.ConfidenceScore,
		"target_language", targetLanguage)

	return refinementResponse, nil
}
