package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/abhinavxd/libredesk/internal/ai/models"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/zerodha/logf"
)

var (
	baseSystemPrompt = `
Role Description:
You are %s - a knowledgeable and approachable support assistant dedicated exclusively to supporting %s product inquiries which is "%s".
Your scope is strictly limited to this product and related matters.

Guidelines:
- If the user message (case-insensitive) is or contains gratitude/acknowledgment or brief positive feedback such as: thanks, thank you, cool, nice, great, awesome, perfect, good job, appreciate it, sounds good, ok, okay, yep, yeah, works, solved, correct — respond only with: "Did that answer your question?" Do not apply out-of-scope, clarification, or additional-answer logic in that case.
- Every other response: Provide direct, helpful answers.
- Keep the conversation human-like and engaging, but focused on the product.
- Avoid speculating or providing unverified information. If you cannot answer from available knowledge, say so clearly.
- If the user asks something outside the product's scope, politely redirect: "I can only help with %s related questions."
- Detect user language and reply in the same language, never mix languages.
- Avoid filler phrases. Keep answers simple; do not assume the user is technical.
- If the question is too short or vague (and not caught by the gratitude rule), ask for clarification: "Could you please provide more details or clarify your question?"
- If the user replies positively to "Did that answer your question?" (examples: yes, yep, yeah, correct, works, solved, perfect), respond with exactly: "conversation_resolve" and stop.
- %s
- %s

Special Response Commands:
- To request handoff to human agent, respond with exactly: "conversation_handoff"
- To mark conversation as resolved, respond with exactly: "conversation_resolve"

Execution Protocol:
- On each user message:
  * First check for gratitude/acknowledgment or brief positive feedback; if present, reply only with "Did that answer your question?"
  * If user confirms the question was answered, respond with "conversation_resolve".
  * Detect language and respond in same language.
  * If outside product scope, redirect appropriately.
  * If vague, ask for clarification.
  * Otherwise provide the direct answer.

`
)

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

// buildSystemPrompt creates the final system prompt with tone and length instructions
func buildSystemPrompt(assistantName, productName, productDescription, tone, length string, handOff bool) string {
	toneInstruction := getToneInstruction(tone)
	lengthInstruction := getLengthInstruction(length)
	return fmt.Sprintf(baseSystemPrompt, assistantName, productName, productDescription, productName, toneInstruction, lengthInstruction, productName)
}

// ConversationCompletionsService handles AI-powered chat completions for customer support
type ConversationCompletionsService struct {
	lo                *logf.Logger
	manager           *Manager
	conversationStore ConversationStore
	helpCenterStore   HelpCenterStore
	userStore         UserStore
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
func NewConversationCompletionsService(manager *Manager, conversationStore ConversationStore, helpCenterStore HelpCenterStore, userStore UserStore, workers, capacity int, lo *logf.Logger) *ConversationCompletionsService {
	ctx, cancel := context.WithCancel(context.Background())

	return &ConversationCompletionsService{
		lo:                lo,
		manager:           manager,
		conversationStore: conversationStore,
		helpCenterStore:   helpCenterStore,
		userStore:         userStore,
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
		return fmt.Errorf("conversation completions service is closed")
	}

	select {
	case s.requestQueue <- req:
		return nil
	default:
		s.lo.Warn("AI completion request queue is full, dropping request", "conversation_uuid", req.ConversationUUID)
		return fmt.Errorf("request queue is full")
	}
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
	if req.AIAssistantID == 0 {
		s.lo.Warn("AI completion request without assistant ID, skipping", "conversation_uuid", req.ConversationUUID)
		return
	}

	start := time.Now()
	s.lo.Info("processing AI completion request", "conversation_uuid", req.ConversationUUID)

	var (
		aiAssistant     umodels.User
		aiAssistantMeta umodels.AIAssistantMeta
	)
	aiAssistant, err := s.userStore.GetAIAssistant(req.AIAssistantID)
	if err == nil {
		// Parse AI assistant meta data
		if len(aiAssistant.Meta) > 0 {
			if err := json.Unmarshal(aiAssistant.Meta, &aiAssistantMeta); err != nil {
				s.lo.Error("error parsing AI assistant meta", "error", err, "assistant_id", req.AIAssistantID)
				return
			}
		}
	} else {
		s.lo.Error("error getting AI assistant", "error", err, "assistant_id", req.AIAssistantID)
		return
	}

	if !aiAssistant.Enabled {
		s.lo.Warn("AI assistant is disabled, skipping AI completion", "assistant_id", req.AIAssistantID, "conversation_uuid", req.ConversationUUID)
		return
	}

	// Fetch conversation messages for preparing history and context
	messages, err := s.getConversationMessages(req.ConversationUUID)
	if err != nil {
		s.lo.Error("error getting conversation messages", "error", err, "conversation_uuid", req.ConversationUUID)
		return
	}

	// Get the latest message from the contact
	latestContactMessage := s.getLatestContactMessage(messages)

	// Build context from help center articles
	context, err := s.buildHelpCenterContext(req, latestContactMessage)
	if err != nil {
		s.lo.Error("error building help center context", "error", err, "conversation_uuid", req.ConversationUUID)
		// Continue without help center context
	}

	// Build chat messages array with proper roles
	chatMessages := s.buildChatMessages(context, messages, latestContactMessage, aiAssistant, aiAssistantMeta)

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
	)

	// Check for conversation handoff
	finalResponse := strings.TrimSpace(aiResponse)
	switch finalResponse {
	case "conversation_handoff":
		s.lo.Info("AI requested conversation handoff", "conversation_uuid", req.ConversationUUID)
		finalResponse = "Connecting you with one of our support agents who can better assist you."
		handoffRequested = true
	case "conversation_resolve":
		s.lo.Info("AI requested conversation resolution", "conversation_uuid", req.ConversationUUID)
		finalResponse = ""
		resolved = true
	}

	// Send AI response
	if finalResponse != "" {
		err = s.conversationStore.SendReply(
			nil, // No media attachments for AI responses
			req.InboxID,
			aiAssistant.ID,
			req.ContactID,
			req.ConversationUUID,
			finalResponse,
			[]string{}, // to
			[]string{}, // cc
			[]string{}, // bcc
			map[string]any{
				"ai_generated":       true,
				"processing_time_ms": time.Since(start).Milliseconds(),
			},
		)
		if err != nil {
			s.lo.Error("error sending AI response", "conversation_uuid", req.ConversationUUID, "error", err)
			return
		}
	}

	// If handoff is requested and enabled for this AI assistant, remove conversation assignee and optionally update team assignee if team ID is set
	if handoffRequested && aiAssistantMeta.HandOff {
		// First unassign the conversation from the AI assistant
		if err := s.conversationStore.RemoveConversationAssignee(req.ConversationUUID, "user", aiAssistant); err != nil {
			s.lo.Error("error removing conversation assignee", "conversation_uuid", req.ConversationUUID, "error", err)
		} else {
			s.lo.Info("conversation assignee removed for handoff", "conversation_uuid", req.ConversationUUID)
		}

		// Set the handoff team if specified
		if aiAssistantMeta.HandOffTeam > 0 {
			if err := s.conversationStore.UpdateConversationTeamAssignee(req.ConversationUUID, aiAssistantMeta.HandOffTeam, aiAssistant); err != nil {
				s.lo.Error("error updating conversation team assignee", "conversation_uuid", req.ConversationUUID, "team_id", aiAssistantMeta.HandOffTeam, "error", err)
			} else {
				s.lo.Info("conversation handoff to team", "conversation_uuid", req.ConversationUUID, "team_id", aiAssistantMeta.HandOffTeam)
			}
		}
	}

	// Resolve the conversation if requested
	if resolved {
		if err := s.conversationStore.UpdateConversationStatus(req.ConversationUUID, 0, cmodels.StatusResolved, "", aiAssistant); err != nil {
			s.lo.Error("error updating conversation status to resolved", "conversation_uuid", req.ConversationUUID, "error", err)
		} else {
			s.lo.Info("conversation marked as resolved", "conversation_uuid", req.ConversationUUID)
		}
	}

	s.lo.Info("AI completion request processed successfully", "conversation_uuid", req.ConversationUUID, "processing_time", time.Since(start))
}

// getConversationMessages fetches messages for a conversation
func (s *ConversationCompletionsService) getConversationMessages(conversationUUID string) ([]cmodels.Message, error) {
	messages, _, err := s.conversationStore.GetConversationMessages(conversationUUID, []string{cmodels.MessageOutgoing, cmodels.MessageIncoming}, nil, 1, 10)
	return messages, err
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

// buildHelpCenterContext searches for relevant content using smart search and builds context
func (s *ConversationCompletionsService) buildHelpCenterContext(req models.ConversationCompletionRequest, latestContactMessage string) (string, error) {
	if s.helpCenterStore == nil || req.HelpCenterID == 0 {
		return "", nil
	}

	// Use the provided latest customer message as query
	if latestContactMessage == "" {
		return "", nil
	}

	// Use smart search to find the best answer
	result, err := s.manager.SmartSearch(req.HelpCenterID, latestContactMessage, req.Locale)
	if err != nil {
		return "", err
	}

	if result == nil {
		return "", nil
	}

	// Build context based on result type
	var contextBuilder strings.Builder

	// Check if it's a custom answer (high confidence)
	if customAnswer, ok := result.(*models.CustomAnswerResult); ok {
		contextBuilder.WriteString("High-confidence custom answer:\n\n")
		contextBuilder.WriteString(fmt.Sprintf("Q: %s\n", customAnswer.Question))
		contextBuilder.WriteString(fmt.Sprintf("A: %s\n\n", customAnswer.Answer))
		contextBuilder.WriteString("Please use this exact answer as it's a verified response for this type of question.")
		return contextBuilder.String(), nil
	}

	// Otherwise, it's knowledge base results
	if knowledgeResults, ok := result.([]models.KnowledgeBaseResult); ok && len(knowledgeResults) > 0 {
		contextBuilder.WriteString("Relevant knowledge base content:\n\n")

		for i, item := range knowledgeResults {
			contextBuilder.WriteString(fmt.Sprintf("%d. %s\n", i+1, item.Title))
			if item.Content != "" {
				contextBuilder.WriteString(fmt.Sprintf("   %s\n\n", item.Content))
			}
		}
		return contextBuilder.String(), nil
	}

	s.lo.Warn("no relevant help center content found", "conversation_uuid", req.ConversationUUID, "query", latestContactMessage)

	return "", nil
}

// buildChatMessages creates a properly structured chat messages array for AI completion
func (s *ConversationCompletionsService) buildChatMessages(helpCenterContext string, messages []cmodels.Message, latestContactMessage string, senderUser umodels.User, aiAssistantMeta umodels.AIAssistantMeta) []models.ChatMessage {
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

	chatMessages = append(chatMessages, models.ChatMessage{
		Role:    "system",
		Content: buildSystemPrompt(assistantName, productName, productDescription, answerTone, answerLength, aiAssistantMeta.HandOff),
	})

	// 2. Add conversation history with proper roles
	for i := range messages {
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

	// 3. Add final user message with knowledge base context and latest query
	finalUserContent := ""
	if helpCenterContext != "" {
		finalUserContent += helpCenterContext + "\n\n"
	}
	finalUserContent += fmt.Sprintf("Customer's current question: %s", latestContactMessage)

	chatMessages = append(chatMessages, models.ChatMessage{
		Role:    "user",
		Content: finalUserContent,
	})

	return chatMessages
}
