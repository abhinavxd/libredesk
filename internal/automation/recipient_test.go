package automation

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/automation/models"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/stretchr/testify/assert"
)

func TestEvaluateRuleIncomingTo(t *testing.T) {
	engine := createTestEngine(new(mockConversationStore))
	tests := []struct {
		name       string
		recipients []string
		rule       models.RuleDetail
		want       bool
	}{
		{
			name:       "equals recipient case insensitively",
			recipients: []string{"foundation@zerya.dev"},
			rule:       models.RuleDetail{Operator: models.RuleOperatorEquals, Value: "FOUNDATION@ZERYA.DEV"},
			want:       true,
		},
		{
			name:       "not equals rejects matching recipient",
			recipients: []string{"foundation@zerya.dev"},
			rule:       models.RuleDetail{Operator: models.RuleOperatorNotEqual, Value: "foundation@zerya.dev"},
			want:       false,
		},
		{
			name:       "contains matches recipient fragment",
			recipients: []string{"support@example.com", "foundation@zerya.dev"},
			rule:       models.RuleDetail{Operator: models.RuleOperatorContains, Value: "example.org, @zerya.dev"},
			want:       true,
		},
		{
			name:       "case sensitive match is respected",
			recipients: []string{"foundation@zerya.dev"},
			rule: models.RuleDetail{
				Operator:           models.RuleOperatorEquals,
				Value:              "Foundation@zerya.dev",
				CaseSensitiveMatch: true,
			},
			want: false,
		},
		{
			name:       "set requires a recipient",
			recipients: []string{"foundation@zerya.dev"},
			rule:       models.RuleDetail{Operator: models.RuleOperatorSet},
			want:       true,
		},
		{
			name: "not set matches missing recipients",
			rule: models.RuleDetail{Operator: models.RuleOperatorNotSet},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.rule.Field = models.ConversationIncomingTo
			tt.rule.FieldType = models.FieldTypeConversationField
			conversation := cmodels.Conversation{IncomingTo: tt.recipients}
			assert.Equal(t, tt.want, engine.evaluateRule(tt.rule, conversation, nil))
		})
	}
}
