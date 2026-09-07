package automation

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/automation/models"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/zerodha/logf"
)

const testSystemUserID = 99

func createQueueTestEngine(store *mockConversationStore) *Engine {
	logger := logf.New(logf.Opts{Level: logf.DebugLevel})
	return &Engine{
		conversationStore: store,
		lo:                &logger,
		systemUserID:      testSystemUserID,
		taskQueue:         make(chan ConversationTask, 100),
	}
}

func systemActor() umodels.User { return umodels.User{ID: testSystemUserID} }
func agentActor() umodels.User  { return umodels.User{ID: 7} }

func TestSuppressed_SystemActorEventDropped(t *testing.T) {
	engine := createQueueTestEngine(new(mockConversationStore))
	conv := createTestConversation()

	engine.suppress(conv.UUID)
	defer engine.unsuppress(conv.UUID)
	engine.EvaluateConversationUpdateRules(conv, models.EventConversationStatusChange, nil, systemActor())

	assert.Equal(t, 0, len(engine.taskQueue), "automation's own event must be dropped while suppressed")
}

func TestSuppressed_AgentActorEventEnqueued(t *testing.T) {
	engine := createQueueTestEngine(new(mockConversationStore))
	conv := createTestConversation()

	engine.suppress(conv.UUID)
	defer engine.unsuppress(conv.UUID)
	engine.EvaluateConversationUpdateRules(conv, models.EventConversationStatusChange, nil, agentActor())

	assert.Equal(t, 1, len(engine.taskQueue), "a real agent event during suppression must still evaluate")
}

func TestUnsuppressed_SystemActorEventEnqueued(t *testing.T) {
	engine := createQueueTestEngine(new(mockConversationStore))
	conv := createTestConversation()

	engine.EvaluateConversationUpdateRules(conv, models.EventConversationStatusChange, nil, systemActor())

	assert.Equal(t, 1, len(engine.taskQueue), "system-actor events outside suppression (e.g. autoassigner) must evaluate")
}

func TestSuppression_ScopedToConversation(t *testing.T) {
	engine := createQueueTestEngine(new(mockConversationStore))
	suppressed := createTestConversation()
	other := createTestConversation(func(c *cmodels.Conversation) { c.UUID = "other-uuid" })

	engine.suppress(suppressed.UUID)
	defer engine.unsuppress(suppressed.UUID)
	engine.EvaluateConversationUpdateRules(other, models.EventConversationStatusChange, nil, systemActor())

	assert.Equal(t, 1, len(engine.taskQueue), "suppression of one conversation must not affect another")
}

func TestSuppression_ReleasedAfterUnsuppress(t *testing.T) {
	engine := createQueueTestEngine(new(mockConversationStore))
	conv := createTestConversation()

	engine.suppress(conv.UUID)
	engine.unsuppress(conv.UUID)
	engine.EvaluateConversationUpdateRules(conv, models.EventConversationStatusChange, nil, systemActor())

	assert.Equal(t, 1, len(engine.taskQueue), "events after the suppress window must evaluate again")
}

func TestSuppression_NestedCounting(t *testing.T) {
	engine := createQueueTestEngine(new(mockConversationStore))
	conv := createTestConversation()

	engine.suppress(conv.UUID)
	engine.suppress(conv.UUID)
	engine.unsuppress(conv.UUID)
	engine.EvaluateConversationUpdateRules(conv, models.EventConversationStatusChange, nil, systemActor())
	assert.Equal(t, 0, len(engine.taskQueue), "still one suppress claim in flight, event must be dropped")

	engine.unsuppress(conv.UUID)
	engine.EvaluateConversationUpdateRules(conv, models.EventConversationStatusChange, nil, systemActor())
	assert.Equal(t, 1, len(engine.taskQueue), "all claims released, event must evaluate")
}

// Core no-loop invariant: an action's synchronous echo must not re-enter the queue.
func TestActionEcho_DoesNotRequeue(t *testing.T) {
	mockStore := new(mockConversationStore)
	engine := createQueueTestEngine(mockStore)
	conv := createTestConversation()

	rules := []models.Rule{
		createTestRule(
			[]models.RuleGroup{
				{
					LogicalOp: models.OperatorAnd,
					Rules: []models.RuleDetail{
						{Field: models.ConversationStatus, Operator: models.RuleOperatorEquals, Value: "1", FieldType: models.FieldTypeConversationField},
					},
				},
			},
			[]models.RuleAction{{Type: models.ActionSetStatus, Value: []string{"2"}}},
			models.OperatorOR,
		),
	}

	mockStore.On("ApplyAction", mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		engine.EvaluateConversationUpdateRules(conv, models.EventConversationStatusChange, nil, systemActor())
	}).Return(nil)

	engine.evalConversationRules(rules, conv, nil)

	assert.Equal(t, 1, mockStore.callCount, "action must run exactly once")
	assert.Equal(t, 0, len(engine.taskQueue), "the action's own echo must not re-enter the queue")
}

// Rule A's echo would match rule B and vice versa; both must die in the suppress window.
func TestActionEcho_PingPongRulesDoNotLoop(t *testing.T) {
	mockStore := new(mockConversationStore)
	engine := createQueueTestEngine(mockStore)
	conv := createTestConversation()

	ruleSetPriority := createTestRule(
		[]models.RuleGroup{
			{
				LogicalOp: models.OperatorAnd,
				Rules: []models.RuleDetail{
					{Field: models.ConversationStatus, Operator: models.RuleOperatorSet, Value: "", FieldType: models.FieldTypeConversationField},
				},
			},
		},
		[]models.RuleAction{{Type: models.ActionSetPriority, Value: []string{"3"}}},
		models.OperatorOR,
	)
	ruleSetStatus := createTestRule(
		[]models.RuleGroup{
			{
				LogicalOp: models.OperatorAnd,
				Rules: []models.RuleDetail{
					{Field: models.ConversationStatus, Operator: models.RuleOperatorSet, Value: "", FieldType: models.FieldTypeConversationField},
				},
			},
		},
		[]models.RuleAction{{Type: models.ActionSetStatus, Value: []string{"2"}}},
		models.OperatorOR,
	)

	mockStore.On("ApplyAction", mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		action := args.Get(0).(models.RuleAction)
		if action.Type == models.ActionSetPriority {
			engine.EvaluateConversationUpdateRules(conv, models.EventConversationPriorityChange, nil, systemActor())
		} else {
			engine.EvaluateConversationUpdateRules(conv, models.EventConversationStatusChange, nil, systemActor())
		}
	}).Return(nil)

	engine.evalConversationRules([]models.Rule{ruleSetPriority, ruleSetStatus}, conv, nil)

	assert.Equal(t, 2, mockStore.callCount, "each rule's action runs once")
	assert.Equal(t, 0, len(engine.taskQueue), "no echo may re-enter the queue, so no ping-pong is possible")
}
