package ai

import (
	"context"
	"strings"

	"github.com/abhinavxd/libredesk/internal/ai/models"
)

const defaultMaxSteps = 5

// RunAgent runs the tool-calling loop for agent-facing callers: built-in knowledge search only, no custom tools.
func (m *Manager) RunAgent(ctx context.Context, systemPrompt string, history []models.ChatMessage, maxSteps int, tctx ToolContext) (string, error) {
	return m.RunAgentWithTools(ctx, systemPrompt, history, maxSteps, tctx, nil, true, true, nil)
}

// RunAgentWithTools runs the tool-calling loop; allowedToolIDs restricts custom tools to that set (empty loads none), appendWorkspaceInstructions folds the workspace admin instructions into the system prompt.
func (m *Manager) RunAgentWithTools(ctx context.Context, systemPrompt string, history []models.ChatMessage, maxSteps int, tctx ToolContext, allowedToolIDs []int, appendWorkspaceInstructions, includeBuiltinSearch bool, extra []Tool) (string, error) {
	run, err := m.newAgentRun(systemPrompt, history, maxSteps, tctx, allowedToolIDs, appendWorkspaceInstructions, includeBuiltinSearch, extra, AgentRunScope{}, false)
	if err != nil {
		return "", err
	}
	res, err := m.continueAgentRun(ctx, run)
	if err != nil {
		return "", err
	}
	return res.Content, nil
}

// RunAgentWithApprovals runs the tool-calling loop for agent surfaces; a custom tool flagged for approval pauses the run until the agent approves or declines it.
func (m *Manager) RunAgentWithApprovals(ctx context.Context, systemPrompt string, history []models.ChatMessage, maxSteps int, tctx ToolContext, allowedToolIDs []int, extra []Tool, scope AgentRunScope) (models.AgentRunResult, error) {
	run, err := m.newAgentRun(systemPrompt, history, maxSteps, tctx, allowedToolIDs, true, true, extra, scope, true)
	if err != nil {
		return models.AgentRunResult{}, err
	}
	return m.continueAgentRun(ctx, run)
}

func (m *Manager) newAgentRun(systemPrompt string, history []models.ChatMessage, maxSteps int, tctx ToolContext, allowedToolIDs []int, appendWorkspaceInstructions, includeBuiltinSearch bool, extra []Tool, scope AgentRunScope, approvals bool) (*pendingAgentRun, error) {
	if maxSteps <= 0 {
		maxSteps = defaultMaxSteps
	}

	cfg, err := m.getProviderConfig(models.ProviderTypeCompletion)
	if err != nil {
		return nil, err
	}
	client := NewOpenAIClient(cfg, m.lo, m.providerHTTPClient)
	if instructions := strings.TrimSpace(cfg.Instructions); appendWorkspaceInstructions && instructions != "" {
		if systemPrompt != "" {
			systemPrompt += "\n\n"
		}
		systemPrompt += "Workspace admin instructions (follow these, they take precedence on tone and format):\n" + instructions
	}

	registry, defs, err := m.buildToolRegistry(tctx, allowedToolIDs, includeBuiltinSearch, extra)
	if err != nil {
		return nil, err
	}

	messages := make([]models.ChatMessage, 0, len(history)+1)
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

	return &pendingAgentRun{
		Scope:       scope,
		Client:      client,
		Definitions: defs,
		Registry:    registry,
		Messages:    messages,
		MaxSteps:    maxSteps,
		Approvals:   approvals,
	}, nil
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
