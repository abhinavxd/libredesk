package ai

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/ai/models"
	"github.com/jmoiron/sqlx/types"
)

type approvalTestProvider struct {
	responses []models.ChatCompletionResult
	calls     int
	failAt    int
}

type approvalTestTool struct {
	executions int
}

func (p *approvalTestProvider) SendPrompt(context.Context, models.PromptPayload) (string, error) {
	return "", nil
}

func (p *approvalTestProvider) SendChatCompletion(context.Context, models.ChatCompletionPayload) (models.ChatCompletionResult, error) {
	p.calls++
	if p.calls == p.failAt {
		return models.ChatCompletionResult{}, errors.New("provider unavailable")
	}
	response := p.responses[0]
	p.responses = p.responses[1:]
	return response, nil
}

func (p *approvalTestProvider) GetEmbeddings(context.Context, string) ([]float32, error) {
	return nil, nil
}

func (p *approvalTestProvider) GetEmbeddingsBatch(context.Context, []string) ([][]float32, error) {
	return nil, nil
}

func (t *approvalTestTool) Name() string { return "update_balance" }

func (t *approvalTestTool) Description() string { return "Updates a balance." }

func (t *approvalTestTool) Parameters() types.JSONText { return types.JSONText(`{"type":"object"}`) }

func (t *approvalTestTool) Execute(context.Context, string) (string, error) {
	t.executions++
	return "updated", nil
}

func TestAgentToolDeclineResumesWithoutExecution(t *testing.T) {
	m := newTestManager(t)
	tool := &approvalTestTool{}
	provider := &approvalTestProvider{responses: []models.ChatCompletionResult{
		{ToolCalls: []models.ToolCall{{ID: "call-1", Type: "function", Function: models.ToolCallFunction{Name: tool.Name(), Arguments: `{"amount":100}`}}}},
		{Content: "No changes were made."},
	}}
	run := &pendingAgentRun{
		Scope:     AgentRunScope{AgentID: 7, ConversationID: 9, ConversationUUID: "conversation-1", Surface: models.ToolInvocationCopilot},
		Client:    provider,
		Registry:  map[string]Tool{tool.Name(): tool},
		MaxSteps:  5,
		Approvals: true,
	}

	result, err := m.continueAgentRun(t.Context(), run)
	if err != nil {
		t.Fatalf("continue run: %v", err)
	}
	if result.Status != models.AgentRunApprovalRequired || result.Approval == nil {
		t.Fatalf("expected approval request, got %#v", result)
	}
	if tool.executions != 0 {
		t.Fatalf("tool executed before approval: %d", tool.executions)
	}

	result, _, err = m.ResumeAgentRun(t.Context(), result.Approval.RunID, 7, false)
	if err != nil {
		t.Fatalf("decline run: %v", err)
	}
	if result.Status != models.AgentRunCompleted || result.Content != "No changes were made." {
		t.Fatalf("unexpected completed result: %#v", result)
	}
	if tool.executions != 0 {
		t.Fatalf("declined tool executed: %d", tool.executions)
	}
	if _, _, err := m.ResumeAgentRun(t.Context(), run.ID, 7, false); err == nil {
		t.Fatal("expected a second decision to fail")
	}
}

func TestAgentToolDecisionIsFinalWhenFollowUpFails(t *testing.T) {
	m := newTestManager(t)
	tool := &approvalTestTool{}
	provider := &approvalTestProvider{
		responses: []models.ChatCompletionResult{
			{ToolCalls: []models.ToolCall{{ID: "call-1", Type: "function", Function: models.ToolCallFunction{Name: tool.Name(), Arguments: `{}`}}}},
		},
		failAt: 2,
	}
	run := &pendingAgentRun{
		Scope:     AgentRunScope{AgentID: 7, ConversationID: 9, ConversationUUID: "conversation-1", Surface: models.ToolInvocationCopilot},
		Client:    provider,
		Registry:  map[string]Tool{tool.Name(): tool},
		MaxSteps:  5,
		Approvals: true,
	}

	result, err := m.continueAgentRun(t.Context(), run)
	if err != nil {
		t.Fatalf("continue run: %v", err)
	}
	result, _, err = m.ResumeAgentRun(t.Context(), result.Approval.RunID, 7, false)
	if err != nil {
		t.Fatalf("decline run: %v", err)
	}
	if result.Status != models.AgentRunCompleted {
		t.Fatalf("expected a final result, got %#v", result)
	}
}

func TestSecondRunInSameScopeIsRejectedWhileOneAwaitsApproval(t *testing.T) {
	m := newTestManager(t)
	tool := &approvalTestTool{}
	scope := AgentRunScope{AgentID: 7, ConversationID: 9, ConversationUUID: "conversation-1", Surface: models.ToolInvocationCopilot}
	newRun := func() *pendingAgentRun {
		return &pendingAgentRun{
			Scope: scope,
			Client: &approvalTestProvider{responses: []models.ChatCompletionResult{
				{ToolCalls: []models.ToolCall{{ID: "call-1", Type: "function", Function: models.ToolCallFunction{Name: tool.Name(), Arguments: `{}`}}}},
			}},
			Registry:  map[string]Tool{tool.Name(): tool},
			MaxSteps:  5,
			Approvals: true,
		}
	}

	if _, err := m.continueAgentRun(t.Context(), newRun()); err != nil {
		t.Fatalf("first run: %v", err)
	}
	if _, err := m.continueAgentRun(t.Context(), newRun()); err == nil {
		t.Fatal("expected the second run in the same scope to be rejected")
	}
	if len(m.pendingRuns) != 1 {
		t.Fatalf("expected one pending run, got %d", len(m.pendingRuns))
	}
}

func (t *approvalTestTool) approvalRequired() bool { return true }

func (t *approvalTestTool) toolID() int { return 1 }

func (t *approvalTestTool) toolUpdatedAt() time.Time { return time.Unix(1, 0) }
