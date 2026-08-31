package webhook

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/webhook/models"
	"github.com/lib/pq"
)

func TestMatchesWebhookFilters(t *testing.T) {
	sales := models.Webhook{InboxIDs: pq.Int64Array{10}}
	supportTeam := models.Webhook{TeamIDs: pq.Int64Array{3}}
	agent := models.Webhook{UserIDs: pq.Int64Array{7}}
	salesAndAgent := models.Webhook{InboxIDs: pq.Int64Array{10}, UserIDs: pq.Int64Array{7}}
	open := models.Webhook{}

	salesConv := map[string]any{"inbox_id": 10, "assigned_team_id": 3, "assigned_user_id": 7}
	otherConv := map[string]any{"inbox_id": 11, "assigned_team_id": 4, "assigned_user_id": 8}
	nested := map[string]any{"conversation": salesConv, "assigned_to": 7}
	message := map[string]any{"inbox_id": 10, "text_content": "hi"}

	tests := []struct {
		name    string
		w       models.Webhook
		event   models.WebhookEvent
		payload any
		want    bool
	}{
		{"empty filters match all", open, models.EventConversationCreated, otherConv, true},
		{"inbox match", sales, models.EventConversationCreated, salesConv, true},
		{"inbox miss", sales, models.EventConversationCreated, otherConv, false},
		{"team match", supportTeam, models.EventConversationAssigned, salesConv, true},
		{"team miss", supportTeam, models.EventConversationAssigned, otherConv, false},
		{"user match", agent, models.EventConversationAssigned, salesConv, true},
		{"user from assigned_to", agent, models.EventConversationAssigned, nested, true},
		{"and across types", salesAndAgent, models.EventMessageCreated, salesConv, true},
		{"and fails when user missing", salesAndAgent, models.EventMessageCreated, message, false},
		{"inbox on message", sales, models.EventMessageCreated, message, true},
		{"test always matches", sales, models.EventWebhookTest, otherConv, true},
		{"missing id fails set filter", sales, models.EventMessageCreated, map[string]any{"text_content": "x"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesWebhookFilters(tt.w, DeliveryTask{Event: tt.event, Payload: tt.payload})
			if got != tt.want {
				t.Fatalf("got %v want %v", got, tt.want)
			}
		})
	}
}
