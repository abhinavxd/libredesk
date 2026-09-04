package automation

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/automation/models"
	"github.com/stretchr/testify/assert"
)

func TestEvaluateStringValuesEmailRecipients(t *testing.T) {
	tests := []struct {
		name   string
		values []string
		rule   models.RuleDetail
		want   bool
	}{
		{
			name:   "equals any recipient case insensitively",
			values: []string{"support@example.com", "foundation@zerya.dev"},
			rule:   models.RuleDetail{Operator: models.RuleOperatorEquals, Value: "FOUNDATION@ZERYA.DEV"},
			want:   true,
		},
		{
			name:   "not equals requires all recipients to differ",
			values: []string{"support@example.com", "foundation@zerya.dev"},
			rule:   models.RuleDetail{Operator: models.RuleOperatorNotEqual, Value: "foundation@zerya.dev"},
			want:   false,
		},
		{
			name:   "contains matches any configured fragment",
			values: []string{"foundation@zerya.dev"},
			rule:   models.RuleDetail{Operator: models.RuleOperatorContains, Value: "example.org, @zerya.dev"},
			want:   true,
		},
		{
			name:   "case sensitive match is respected",
			values: []string{"foundation@zerya.dev"},
			rule: models.RuleDetail{
				Operator:           models.RuleOperatorEquals,
				Value:              "Foundation@zerya.dev",
				CaseSensitiveMatch: true,
			},
			want: false,
		},
		{
			name:   "set requires at least one recipient",
			values: []string{"foundation@zerya.dev"},
			rule:   models.RuleDetail{Operator: models.RuleOperatorSet},
			want:   true,
		},
		{
			name: "not set matches missing recipients",
			rule: models.RuleDetail{Operator: models.RuleOperatorNotSet},
			want: true,
		},
		{
			name:   "empty recipients are not set",
			values: []string{"", "  "},
			rule:   models.RuleDetail{Operator: models.RuleOperatorNotSet},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, evaluateStringValues(tt.values, tt.rule))
		})
	}
}
