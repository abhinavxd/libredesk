package conversation

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/volatiletech/null/v9"
)

func TestReplyCountsTowardSLA(t *testing.T) {
	agent := umodels.User{ID: 5, Type: umodels.UserTypeAgent}
	assistant := umodels.User{ID: 7, Type: umodels.UserTypeAIAssistant}
	assignedTo := func(id int) models.Conversation {
		return models.Conversation{AssignedUserID: null.IntFrom(id)}
	}

	tests := []struct {
		name         string
		sender       umodels.User
		conversation models.Conversation
		want         bool
	}{
		{"an agent reply counts", agent, assignedTo(9), true},
		{"an agent reply on an unassigned conversation counts", agent, models.Conversation{}, true},
		{"a reply from a sender that could not be loaded counts", umodels.User{}, models.Conversation{}, true},
		{"an assistant reply counts while the conversation is assigned to it", assistant, assignedTo(7), true},
		{"an assistant reply after a handoff to a team does not count", assistant, models.Conversation{}, false},
		{"an assistant reply after a handoff to an agent does not count", assistant, assignedTo(5), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := replyCountsTowardSLA(tt.sender, tt.conversation); got != tt.want {
				t.Fatalf("replyCountsTowardSLA = %v, want %v", got, tt.want)
			}
		})
	}
}
