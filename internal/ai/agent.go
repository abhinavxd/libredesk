package ai

import (
	"context"
	"strings"

	"github.com/abhinavxd/libredesk/internal/ai/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
)

const defaultMaxSteps = 5

// RunAgent runs the tool-calling loop up to maxSteps and returns the model's text answer.
func (m *Manager) RunAgent(ctx context.Context, systemPrompt string, history []models.ChatMessage, maxSteps int, tctx ToolContext, extraTools ...Tool) (string, error) {
	if maxSteps <= 0 {
		maxSteps = defaultMaxSteps
	}

	cfg, err := m.getProviderConfig(models.ProviderTypeCompletion)
	if err != nil {
		return "", err
	}
	client := NewOpenAIClient(cfg, m.lo)
	if instructions := strings.TrimSpace(cfg.Instructions); instructions != "" && systemPrompt != "" {
		systemPrompt += "\n\nWorkspace admin instructions (follow these, they take precedence on tone and format):\n" + instructions
	}

	registry, defs, err := m.buildToolRegistry(tctx, extraTools...)
	if err != nil {
		return "", err
	}

	messages := make([]models.ChatMessage, 0, len(history)+2)
	if systemPrompt != "" {
		messages = append(messages, models.ChatMessage{Role: "system", Content: systemPrompt})
	}
	messages = append(messages, history...)

	for step := 0; step < maxSteps; step++ {
		res, err := m.chatCompletion(client, models.ChatCompletionPayload{Messages: messages, Tools: defs})
		if err != nil {
			return "", err
		}

		if len(res.ToolCalls) == 0 {
			return res.Content, nil
		}

		messages = append(messages, models.ChatMessage{
			Role:      "assistant",
			Content:   res.Content,
			ToolCalls: res.ToolCalls,
		})

		for _, tc := range res.ToolCalls {
			result := m.executeToolCall(ctx, registry, tc)
			messages = append(messages, models.ChatMessage{
				Role:       "tool",
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
				Content:    result,
			})
		}
	}

	// Step budget exhausted, force a final answer by omitting tools.
	res, err := m.chatCompletion(client, models.ChatCompletionPayload{Messages: messages})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(res.Content) == "" {
		m.lo.Warn("agent produced no answer within the step budget", "max_steps", maxSteps)
		return "", envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return res.Content, nil
}

func (m *Manager) executeToolCall(ctx context.Context, registry map[string]Tool, tc models.ToolCall) string {
	tool, ok := registry[tc.Function.Name]
	if !ok {
		m.lo.Warn("model called unknown tool", "tool", tc.Function.Name)
		return "error: unknown tool " + tc.Function.Name
	}
	out, err := tool.Execute(ctx, tc.Function.Arguments)
	if err != nil {
		m.lo.Error("error executing tool", "tool", tc.Function.Name, "error", err)
		return "error executing tool: " + err.Error()
	}
	return out
}

func (m *Manager) chatCompletion(client ProviderClient, payload models.ChatCompletionPayload) (models.ChatCompletionResult, error) {
	res, err := client.SendChatCompletion(payload)
	if err != nil {
		return models.ChatCompletionResult{}, m.providerError(err)
	}
	return res, nil
}
