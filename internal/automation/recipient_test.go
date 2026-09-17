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
			name:       "equals any recipient case insensitively",
			recipients: []string{"support@example.com", "foundation@zerya.dev"},
			rule:       models.RuleDetail{Operator: models.RuleOperatorEquals, Value: "FOUNDATION@ZERYA.DEV"},
			want:       true,
		},
		{
			name:       "not equals requires all recipients to differ",
			recipients: []string{"support@example.com", "foundation@zerya.dev"},
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
			name:       "contains ignores empty candidates",
			recipients: []string{"support@example.com"},
			rule:       models.RuleDetail{Operator: models.RuleOperatorContains, Value: ", missing.example,  ,"},
			want:       false,
		},
		{
			name:       "not contains ignores empty candidates",
			recipients: []string{"support@example.com"},
			rule:       models.RuleDetail{Operator: models.RuleOperatorNotContains, Value: ", missing.example,  ,"},
			want:       true,
		},
		{
			name:       "contains matches a repeated candidate around blanks",
			recipients: []string{"support@example.com"},
			rule:       models.RuleDetail{Operator: models.RuleOperatorContains, Value: ", @example.com,  , @example.com,"},
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
		{
			name:       "empty recipients are not set",
			recipients: []string{"", "  "},
			rule:       models.RuleDetail{Operator: models.RuleOperatorNotSet},
			want:       true,
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
