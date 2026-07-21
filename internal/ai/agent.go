package ai

import (
	"context"
	"strings"

	"github.com/abhinavxd/libredesk/internal/ai/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
)

const defaultMaxSteps = 5

// RunAgent runs the tool-calling loop for agent-facing callers: built-in knowledge search only, no custom tools.
func (m *Manager) RunAgent(ctx context.Context, systemPrompt string, history []models.ChatMessage, maxSteps int, tctx ToolContext) (string, error) {
	return m.RunAgentWithTools(ctx, systemPrompt, history, maxSteps, tctx, nil, true, true, nil)
}

// RunAgentWithTools runs the tool-calling loop; allowedToolIDs restricts custom tools to that set (empty loads none), appendWorkspaceInstructions folds the workspace admin instructions into the system prompt.
func (m *Manager) RunAgentWithTools(ctx context.Context, systemPrompt string, history []models.ChatMessage, maxSteps int, tctx ToolContext, allowedToolIDs []int, appendWorkspaceInstructions, includeBuiltinSearch bool, extra []Tool) (string, error) {
	if maxSteps <= 0 {
		maxSteps = defaultMaxSteps
	}

	cfg, err := m.getProviderConfig(models.ProviderTypeCompletion)
	if err != nil {
		return "", err
	}
	client := NewOpenAIClient(cfg, m.lo, m.providerHTTPClient)
	if instructions := strings.TrimSpace(cfg.Instructions); appendWorkspaceInstructions && instructions != "" {
		if systemPrompt != "" {
			systemPrompt += "\n\n"
		}
		systemPrompt += "Workspace admin instructions (follow these, they take precedence on tone and format):\n" + instructions
	}

	registry, defs, err := m.buildToolRegistry(tctx, allowedToolIDs, includeBuiltinSearch)
	if err != nil {
		return "", err
	}
	for _, t := range extra {
		registry[t.Name()] = t
		defs = append(defs, toolDef(t))
	}

	messages := make([]models.ChatMessage, 0, len(history)+2)
	if systemPrompt != "" {
		messages = append(messages, models.ChatMessage{Role: models.RoleSystem, Content: systemPrompt})
	}
	messages = append(messages, history...)

	imageCount := 0
	for _, msg := range messages {
		imageCount += len(msg.Images)
	}
	toolNames := make([]string, len(defs))
	for i, d := range defs {
		toolNames[i] = d.Function.Name
	}
	m.lo.Debug("ai run starting", "model", cfg.Model, "vision", cfg.Vision, "max_steps", maxSteps, "history_messages", len(history), "images", imageCount, "tools", len(defs), "tool_names", strings.Join(toolNames, ","))
	m.lo.Debug("ai run system prompt", "prompt", systemPrompt)

	for step := 0; step < maxSteps; step++ {
		m.lo.Debug("ai run step", "step", step, "messages", len(messages))
		res, err := m.chatCompletion(ctx, client, models.ChatCompletionPayload{Messages: messages, Tools: defs})
		if err != nil {
			return "", err
		}
		m.lo.Debug("ai run model response", "step", step, "content_len", len(res.Content), "tool_calls", len(res.ToolCalls), "content", res.Content, "prompt_tokens", res.Usage.PromptTokens, "completion_tokens", res.Usage.CompletionTokens)

		if len(res.ToolCalls) == 0 {
			m.lo.Debug("ai run final answer", "answer", res.Content)
			return res.Content, nil
		}

		messages = append(messages, models.ChatMessage{
			Role:      models.RoleAssistant,
			Content:   res.Content,
			ToolCalls: res.ToolCalls,
		})

		for _, tc := range res.ToolCalls {
			result := m.executeToolCall(ctx, registry, tc)
			messages = append(messages, models.ChatMessage{
				Role:       models.RoleTool,
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
				Content:    result,
			})
		}
	}

	// Step budget exhausted, force a final answer by omitting tools.
	res, err := m.chatCompletion(ctx, client, models.ChatCompletionPayload{Messages: messages})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(res.Content) == "" {
		m.lo.Warn("agent produced no answer within the step budget", "max_steps", maxSteps)
		return "", envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	m.lo.Debug("ai run final answer", "answer", res.Content, "forced", true)
	return res.Content, nil
}

func (m *Manager) executeToolCall(ctx context.Context, registry map[string]Tool, tc models.ToolCall) string {
	m.lo.Debug("ai run tool call", "tool", tc.Function.Name, "args", tc.Function.Arguments)
	tool, ok := registry[tc.Function.Name]
	if !ok {
		m.lo.Warn("model called unknown tool", "tool", tc.Function.Name)
		return "error: unknown tool " + tc.Function.Name
	}
	out, err := tool.Execute(ctx, tc.Function.Arguments)
	if err != nil {
		m.lo.Error("error executing tool", "tool", tc.Function.Name, "error", err)
		return "the tool call failed"
	}
	m.lo.Debug("ai run tool result", "tool", tc.Function.Name, "result_len", len(out), "result", out)
	return out
}

func (m *Manager) chatCompletion(ctx context.Context, client ProviderClient, payload models.ChatCompletionPayload) (models.ChatCompletionResult, error) {
	res, err := client.SendChatCompletion(ctx, payload)
	if err != nil {
		return models.ChatCompletionResult{}, m.providerError(err)
	}
	return res, nil
}
