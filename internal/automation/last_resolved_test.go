package automation

import (
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/automation/models"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/volatiletech/null/v9"
)

func TestHoursSinceLastResolved(t *testing.T) {
	engine := createTestEngine(new(mockConversationStore))
	now := time.Now()
	conversation := cmodels.Conversation{
		ResolvedAt:     null.TimeFrom(now.Add(-72 * time.Hour)),
		LastResolvedAt: null.TimeFrom(now.Add(-2 * time.Hour)),
	}
	previousValues := map[string]string{}
	tests := []struct {
		name     string
		field    string
		operator string
		value    string
		resolved bool
		want     bool
	}{
		{"latest resolution is recent", models.ConversationHoursSinceLastResolved, models.RuleOperatorLessThan, "24", true, true},
		{"first resolution is older", models.ConversationHoursSinceResolved, models.RuleOperatorGreaterThan, "24", true, true},
		{"latest resolution does not use first", models.ConversationHoursSinceLastResolved, models.RuleOperatorGreaterThan, "24", true, false},
		{"elapsed hours", models.ConversationHoursSinceLastResolved, models.RuleOperatorEquals, "2", true, true},
		{"greater than lower boundary", models.ConversationHoursSinceLastResolved, models.RuleOperatorGreaterThan, "1", true, true},
		{"greater than excludes equality", models.ConversationHoursSinceLastResolved, models.RuleOperatorGreaterThan, "2", true, false},
		{"less than excludes equality", models.ConversationHoursSinceLastResolved, models.RuleOperatorLessThan, "2", true, false},
		{"not equal excludes equality", models.ConversationHoursSinceLastResolved, models.RuleOperatorNotEqual, "2", true, false},
		{"not equal matches other hours", models.ConversationHoursSinceLastResolved, models.RuleOperatorNotEqual, "24", true, true},
		{"resolution is set", models.ConversationHoursSinceLastResolved, models.RuleOperatorSet, "", true, true},
		{"resolution is not missing", models.ConversationHoursSinceLastResolved, models.RuleOperatorNotSet, "", true, false},
		{"missing resolution is not set", models.ConversationHoursSinceLastResolved, models.RuleOperatorNotSet, "", false, true},
		{"missing resolution cannot be set", models.ConversationHoursSinceLastResolved, models.RuleOperatorSet, "", false, false},
		{"missing resolution is not recent", models.ConversationHoursSinceLastResolved, models.RuleOperatorLessThan, "24", false, false},
		{"missing resolution is not greater than negative hours", models.ConversationHoursSinceLastResolved, models.RuleOperatorGreaterThan, "-1", false, false},
		{"missing resolution is not zero hours", models.ConversationHoursSinceLastResolved, models.RuleOperatorEquals, "0", false, false},
		{"missing resolution cannot match inequality", models.ConversationHoursSinceLastResolved, models.RuleOperatorNotEqual, "24", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv := conversation
			if !tt.resolved {
				conv.LastResolvedAt = null.Time{}
			}
			rule := models.RuleDetail{
				Field: tt.field, FieldType: models.FieldTypeConversationField,
				Operator: tt.operator, Value: tt.value,
			}
			if got := engine.evaluateRule(rule, conv, previousValues); got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLastResolvedRuleActions(t *testing.T) {
	now := time.Now()
	action := models.RuleAction{Type: models.ActionSendCSAT, Value: []string{"0"}}
	rules := []models.Rule{
		{
			GroupOperator: models.OperatorAnd,
			Groups: []models.RuleGroup{
				{
					LogicalOp: models.OperatorAnd,
					Rules: []models.RuleDetail{
						{Field: models.ConversationStatus, Operator: models.RuleOperatorEquals, Value: "2"},
						{Field: models.ConversationHoursSinceLastResolved, Operator: models.RuleOperatorGreaterThan, Value: "24"},
					},
				},
			},
			Actions: []models.RuleAction{action},
		},
	}
	tests := []struct {
		name           string
		statusID       int
		lastResolvedAt null.Time
		wantAction     bool
	}{
		{"resolved long enough", 2, null.TimeFrom(now.Add(-48 * time.Hour)), true},
		{"just resolved again", 2, null.TimeFrom(now.Add(-2 * time.Hour)), false},
		{"exact threshold", 2, null.TimeFrom(now.Add(-24 * time.Hour)), false},
		{"reopened after resolution", 1, null.TimeFrom(now.Add(-48 * time.Hour)), false},
		{"missing last resolution", 2, null.Time{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := new(mockConversationStore)
			engine := createTestEngine(store)
			conversation := createTestConversation(func(c *cmodels.Conversation) {
				c.StatusID = null.IntFrom(tt.statusID)
				c.ResolvedAt = null.TimeFrom(now.Add(-72 * time.Hour))
				c.LastResolvedAt = tt.lastResolvedAt
			})
			wantCalls := 0
			if tt.wantAction {
				wantCalls = 1
				store.On("ApplyAction", action, conversation, umodels.User{}).Return(nil).Once()
			}
			previousValues := map[string]string{}
			engine.evalConversationRules(rules, conversation, previousValues)
			if store.callCount != wantCalls {
				t.Fatalf("got %d actions, want %d", store.callCount, wantCalls)
			}
			store.AssertExpectations(t)
		})
	}
}
