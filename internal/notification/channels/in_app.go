package channels

import (
	"encoding/json"

	"github.com/abhinavxd/libredesk/internal/notification/models"
	wsmodels "github.com/abhinavxd/libredesk/internal/ws/models"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

type InAppStore interface {
	Create(int, models.NotificationType, string, null.String, null.Int, null.Int, null.Int, json.RawMessage) (models.UserNotification, error)
}

type WSHub interface {
	BroadcastMessage(msg wsmodels.BroadcastMessage)
}

type inApp struct {
	store InAppStore
	wsHub WSHub
	lo    *logf.Logger
}

func NewInApp(store InAppStore, wsHub WSHub, lo *logf.Logger) Provider {
	return &inApp{store: store, wsHub: wsHub, lo: lo}
}

func (p *inApp) Channel() models.NotificationChannel {
	return models.NotificationChannelInApp
}

func (p *inApp) Send(delivery Delivery) Result {
	n := delivery.Notification
	notification, err := p.store.Create(delivery.Recipient.UserID, n.Type, n.Title, n.Body,
		n.ConversationID, n.MessageID, n.ActorID, n.Meta)
	if err != nil {
		p.lo.Error("error creating in-app notification", "recipient_id", delivery.Recipient.UserID,
			"type", n.Type, "error", err)
		return Result{}
	}
	notification.ConversationUUID = null.StringFrom(n.ConversationUUID)
	notification.MessageUUID = null.StringFrom(n.MessageUUID)
	notification.ActorFirstName = null.StringFrom(n.ActorFirstName)
	notification.ActorLastName = null.StringFrom(n.ActorLastName)
	p.broadcast(delivery.Recipient.UserID, notification)
	return Result{Sent: true, NotificationID: null.IntFrom(notification.ID)}
}

func (p *inApp) broadcast(userID int, notification any) {
	if p.wsHub == nil {
		return
	}
	message := wsmodels.Message{Type: wsmodels.MessageTypeNewNotification, Data: notification}
	data, err := json.Marshal(message)
	if err != nil {
		p.lo.Error("error marshalling notification for Websocket", "error", err)
		return
	}
	p.wsHub.BroadcastMessage(wsmodels.BroadcastMessage{Data: data, Users: []int{userID}})
}
