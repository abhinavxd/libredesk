package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/abhinavxd/libredesk/internal/ai/models"
)

const (
	replyDraftSystemPrompt = `You are a support-agent assistant for libredesk, a customer support tool. Draft a reply to the customer's latest message in the conversation below.

Ground your answer in the knowledge base: call the search_articles tool before answering when the question is about the product. Be concise, accurate and professional. Do not invent information; if the knowledge base does not cover it, say what you can and suggest next steps. Return only the reply text the agent can send, with no preamble or sign-off placeholders.`

	copilotSystemPrompt = `You are Copilot, an assistant for support agents inside libredesk.

Always call the search_articles tool before answering any question about the product, company, policies, pricing, or how something works - do not answer these from your own knowledge without searching first. Only skip the search for pure chit-chat or when the answer is already present in the provided conversation context. If the search returns nothing relevant, say you could not find it in the knowledge base. Answer clearly and concisely, ground answers in what the search and conversation context return, and if you are unsure, say so.`
)

// GenerateReply drafts a reply to a conversation using the agentic loop (tools included).
func (m *Manager) GenerateReply(ctx context.Context, transcript, instruction string, tctx ToolContext) (string, error) {
	var user strings.Builder
	if strings.TrimSpace(transcript) != "" {
		fmt.Fprintf(&user, "Conversation so far:\n%s\n\n", transcript)
	}
	if strings.TrimSpace(instruction) != "" {
		fmt.Fprintf(&user, "Additional instruction from the agent: %s\n\n", instruction)
	}
	user.WriteString("Draft the reply now.")

	history := []models.ChatMessage{{Role: "user", Content: user.String()}}
	return m.RunAgent(ctx, replyDraftSystemPrompt, history, defaultMaxSteps, tctx)
}

// Copilot answers an agent's chat message, optionally grounded in a conversation.
func (m *Manager) Copilot(ctx context.Context, conversationContext string, history []models.ChatMessage, tctx ToolContext) (string, error) {
	msgs := make([]models.ChatMessage, 0, len(history)+1)
	if strings.TrimSpace(conversationContext) != "" {
		msgs = append(msgs, models.ChatMessage{
			Role:    "user",
			Content: "Context - the conversation the agent is viewing:\n" + conversationContext,
		})
	}
	msgs = append(msgs, history...)
	return m.RunAgent(ctx, copilotSystemPrompt, msgs, defaultMaxSteps, tctx)
}

// GetCopilotMessages returns an agent's persisted copilot chat for a conversation.
func (m *Manager) GetCopilotMessages(conversationID, userID int) ([]models.CopilotMessage, error) {
	msgs := []models.CopilotMessage{}
	if err := m.q.GetCopilotMessages.Select(&msgs, conversationID, userID); err != nil {
		return nil, err
	}
	return msgs, nil
}

// SaveCopilotMessage persists one turn of an agent's copilot chat on a conversation.
func (m *Manager) SaveCopilotMessage(conversationID, userID int, role, content string) error {
	_, err := m.q.InsertCopilotMessage.Exec(conversationID, userID, role, content)
	return err
}

// ClearCopilotMessages deletes an agent's copilot chat for a conversation.
func (m *Manager) ClearCopilotMessages(conversationID, userID int) error {
	_, err := m.q.DeleteCopilotMessages.Exec(conversationID, userID)
	return err
}
