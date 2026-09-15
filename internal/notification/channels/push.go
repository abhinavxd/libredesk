package channels

import (
	"fmt"

	"github.com/abhinavxd/libredesk/internal/notification/models"
)

type PushSender interface {
	Send(userID int, payload models.PushPayload) bool
}

type push struct {
	sender PushSender
}

func NewPush(sender PushSender) Provider {
	return &push{sender: sender}
}

func (p *push) Channel() models.NotificationChannel {
	return models.NotificationChannelPush
}

func (p *push) Send(delivery Delivery) Result {
	n := delivery.Notification
	sent := p.sender.Send(delivery.Recipient.UserID, models.PushPayload{
		Title: n.Title,
		Body:  n.Body.String,
		Tag:   fmt.Sprintf("%s_%s", n.Type, n.ConversationUUID),
		URL:   pushRoute(string(n.Type), n.ConversationUUID, n.MessageUUID),
	})
	return Result{Sent: sent}
}

func pushRoute(notificationType, conversationUUID, messageUUID string) string {
	list := "assigned"
	if notificationType == "mention" {
		list = "mentioned"
	}
	route := fmt.Sprintf("/inboxes/%s/conversation/%s", list, conversationUUID)
	if messageUUID != "" {
		route += "?scrollTo=" + messageUUID
	}
	return route
}
