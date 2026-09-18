package ai

import (
	"context"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/ai/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/google/uuid"
)

const pendingAgentRunTTL = 10 * time.Minute

type approvalTool interface {
	Tool
	approvalRequired() bool
	toolID() int
	toolUpdatedAt() time.Time
}

type AgentRunScope struct {
	AgentID          int
	ConversationID   int
	ConversationUUID string
	Surface          string
	UserMessage      string
}

type pendingAgentRun struct {
	ID            string
	Scope         AgentRunScope
	Client        ProviderClient
	Definitions   []models.ToolDef
	Registry      map[string]Tool
	Messages      []models.ChatMessage
	ToolCalls     []models.ToolCall
	NextToolCall  int
	Step          int
	MaxSteps      int
	ToolID        int
	ToolUpdatedAt time.Time
	CreatedAt     time.Time
	ExpiresAt     time.Time
	// Approvals is off for customer-facing runs: no agent is present to approve a tool call.
	Approvals bool
}

func (m *Manager) ResumeAgentRun(ctx context.Context, runID string, agentID int, approved bool) (models.AgentRunResult, AgentRunScope, error) {
	run, err := m.claimAgentRun(runID, agentID)
	if err != nil {
		return models.AgentRunResult{}, AgentRunScope{}, err
	}
	scope := run.Scope
	toolCall := run.ToolCalls[run.NextToolCall]
	result := "The agent declined this tool call."
	m.lo.Info("agent reviewed custom tool call", "run_id", runID, "agent_id", agentID, "conversation_uuid", run.Scope.ConversationUUID, "surface", run.Scope.Surface, "tool", toolCall.Function.Name, "approved", approved)
	if approved {
		if err := m.validateApprovedTool(run); err != nil {
			return models.AgentRunResult{}, scope, err
		}
		result = m.executeToolCall(ctx, run.Registry, toolCall)
	}
	m.appendToolResult(run, toolCall, result)
	run.NextToolCall++

	response, err := m.continueAgentRun(ctx, run)
	if err != nil {
		m.lo.Error("error completing agent run after tool decision", "run_id", runID, "agent_id", agentID, "tool", toolCall.Function.Name, "approved", approved, "error", err)
		messageKey := "ai.toolDeclinedResponseFailed"
		if approved {
			messageKey = "ai.toolRanResponseFailed"
		}
		return completedAgentRun(m.i18n.T(messageKey)), scope, nil
	}
	return response, scope, err
}

func (m *Manager) PendingAgentRunScope(runID string, agentID int) (AgentRunScope, error) {
	m.pendingRunMu.Lock()
	defer m.pendingRunMu.Unlock()
	m.deleteExpiredAgentRunsLocked(time.Now())
	run, ok := m.pendingRuns[runID]
	if !ok {
		return AgentRunScope{}, envelope.NewError(envelope.NotFoundError, m.i18n.T("ai.toolApprovalExpired"), nil)
	}
	if run.Scope.AgentID != agentID {
		return AgentRunScope{}, envelope.NewError(envelope.PermissionError, m.i18n.T("ai.toolApprovalUnavailable"), nil)
	}
	return run.Scope, nil
}

func (m *Manager) PendingAgentApproval(scope AgentRunScope) (string, *models.ToolApproval) {
	m.pendingRunMu.Lock()
	defer m.pendingRunMu.Unlock()
	m.deleteExpiredAgentRunsLocked(time.Now())
	var newest *pendingAgentRun
	for _, run := range m.pendingRuns {
		if !sameAgentRunScope(run.Scope, scope) || (newest != nil && !run.CreatedAt.After(newest.CreatedAt)) {
			continue
		}
		newest = run
	}
	if newest == nil {
		return "", nil
	}
	return newest.Scope.UserMessage, agentRunApproval(newest)
}

func (m *Manager) ClearPendingAgentRuns(scope AgentRunScope) {
	m.pendingRunMu.Lock()
	defer m.pendingRunMu.Unlock()
	for id, run := range m.pendingRuns {
		if sameAgentRunScope(run.Scope, scope) {
			delete(m.pendingRuns, id)
		}
	}
}

func (m *Manager) continueAgentRun(ctx context.Context, run *pendingAgentRun) (models.AgentRunResult, error) {
	if result := m.processAgentToolCalls(ctx, run); result != nil {
		return *result, nil
	}
	for run.Step < run.MaxSteps {
		m.lo.Debug("ai run step", "step", run.Step, "messages", len(run.Messages))
		res, err := m.chatCompletion(ctx, run.Client, models.ChatCompletionPayload{Messages: run.Messages, Tools: run.Definitions})
		if err != nil {
			return models.AgentRunResult{}, err
		}
		m.lo.Debug("ai run model response", "step", run.Step, "content_len", len(res.Content), "tool_calls", len(res.ToolCalls), "content", res.Content, "prompt_tokens", res.Usage.PromptTokens, "completion_tokens", res.Usage.CompletionTokens)
		if len(res.ToolCalls) == 0 {
			m.lo.Debug("ai run final answer", "answer", res.Content)
			return completedAgentRun(res.Content), nil
		}
		run.Messages = append(run.Messages, models.ChatMessage{Role: models.RoleAssistant, Content: res.Content, ToolCalls: res.ToolCalls})
		run.ToolCalls = res.ToolCalls
		run.NextToolCall = 0
		run.Step++
		if result := m.processAgentToolCalls(ctx, run); result != nil {
			return *result, nil
		}
	}

	res, err := m.chatCompletion(ctx, run.Client, models.ChatCompletionPayload{Messages: run.Messages})
	if err != nil {
		return models.AgentRunResult{}, err
	}
	if strings.TrimSpace(res.Content) == "" {
		m.lo.Warn("agent produced no answer within the step budget", "max_steps", run.MaxSteps)
		return models.AgentRunResult{}, envelope.NewError(envelope.GeneralError, m.i18n.T("ai.noAnswerWithinSteps"), nil)
	}
	m.lo.Debug("ai run final answer", "answer", res.Content, "forced", true)
	return completedAgentRun(res.Content), nil
}

func (m *Manager) processAgentToolCalls(ctx context.Context, run *pendingAgentRun) *models.AgentRunResult {
	for run.NextToolCall < len(run.ToolCalls) {
		toolCall := run.ToolCalls[run.NextToolCall]
		if run.Approvals {
			if customTool, ok := run.Registry[toolCall.Function.Name].(approvalTool); ok && customTool.approvalRequired() {
				run.ID = uuid.NewString()
				run.ToolID = customTool.toolID()
				run.ToolUpdatedAt = customTool.toolUpdatedAt()
				run.CreatedAt = time.Now()
				run.ExpiresAt = run.CreatedAt.Add(pendingAgentRunTTL)
				m.storeAgentRun(run)
				return &models.AgentRunResult{Status: models.AgentRunApprovalRequired, Approval: agentRunApproval(run)}
			}
		}
		m.appendToolResult(run, toolCall, m.executeToolCall(ctx, run.Registry, toolCall))
		run.NextToolCall++
	}
	run.ToolCalls = nil
	return nil
}

func (m *Manager) appendToolResult(run *pendingAgentRun, toolCall models.ToolCall, result string) {
	run.Messages = append(run.Messages, models.ChatMessage{
		Role:       models.RoleTool,
		ToolCallID: toolCall.ID,
		Name:       toolCall.Function.Name,
		Content:    result,
	})
}

func (m *Manager) storeAgentRun(run *pendingAgentRun) {
	m.pendingRunMu.Lock()
	defer m.pendingRunMu.Unlock()
	m.deleteExpiredAgentRunsLocked(time.Now())
	m.pendingRuns[run.ID] = run
}

func (m *Manager) claimAgentRun(runID string, agentID int) (*pendingAgentRun, error) {
	m.pendingRunMu.Lock()
	defer m.pendingRunMu.Unlock()
	now := time.Now()
	m.deleteExpiredAgentRunsLocked(now)
	run, ok := m.pendingRuns[runID]
	if !ok {
		return nil, envelope.NewError(envelope.NotFoundError, m.i18n.T("ai.toolApprovalExpired"), nil)
	}
	if run.Scope.AgentID != agentID {
		return nil, envelope.NewError(envelope.PermissionError, m.i18n.T("ai.toolApprovalUnavailable"), nil)
	}
	delete(m.pendingRuns, runID)
	return run, nil
}

func (m *Manager) validateApprovedTool(run *pendingAgentRun) error {
	tool, err := m.GetTool(run.ToolID)
	if err != nil || !tool.Enabled || tool.Name != run.ToolCalls[run.NextToolCall].Function.Name || !tool.UpdatedAt.Equal(run.ToolUpdatedAt) {
		return envelope.NewError(envelope.ConflictError, m.i18n.T("ai.toolChangedBeforeApproval"), nil)
	}
	if run.Scope.Surface == models.ToolInvocationCopilot && !tool.CopilotEnabled {
		return envelope.NewError(envelope.ConflictError, m.i18n.T("ai.toolChangedBeforeApproval"), nil)
	}
	if run.Scope.Surface == models.ToolInvocationReply && !tool.GenerateReplyEnabled {
		return envelope.NewError(envelope.ConflictError, m.i18n.T("ai.toolChangedBeforeApproval"), nil)
	}
	return nil
}

func (m *Manager) deleteExpiredAgentRunsLocked(now time.Time) {
	for id, run := range m.pendingRuns {
		if !run.ExpiresAt.After(now) {
			delete(m.pendingRuns, id)
		}
	}
}

func completedAgentRun(content string) models.AgentRunResult {
	return models.AgentRunResult{Status: models.AgentRunCompleted, Content: stripCodeFence(content)}
}

func agentRunApproval(run *pendingAgentRun) *models.ToolApproval {
	toolCall := run.ToolCalls[run.NextToolCall]
	return &models.ToolApproval{RunID: run.ID, ToolName: toolCall.Function.Name, Arguments: toolCall.Function.Arguments}
}

func sameAgentRunScope(a, b AgentRunScope) bool {
	return a.AgentID == b.AgentID && a.ConversationID == b.ConversationID && a.ConversationUUID == b.ConversationUUID && a.Surface == b.Surface
}
