package automation

import (
	"github.com/abhinavxd/libredesk/internal/automation/models"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
)

func (e *Engine) MatchesContact(group models.RuleGroup, contact umodels.User) bool {
	if len(group.Rules) == 0 {
		return true
	}
	for _, rule := range group.Rules {
		if rule.FieldType != models.FieldTypeContactCustomAttribute {
			return false
		}
	}
	return e.evaluateGroup(group.Rules, group.LogicalOp, cmodels.Conversation{Contact: cmodels.ConversationContact{CustomAttributes: contact.CustomAttributes}}, nil)
}
