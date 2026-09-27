package channels

import (
	"time"

	"github.com/abhinavxd/libredesk/internal/notification/models"
)

type EmailQueue interface {
	Send(models.Email) bool
	SendAfter(models.Email, time.Duration) bool
}

type email struct {
	queue EmailQueue
}

func NewEmail(queue EmailQueue) Provider {
	return &email{queue: queue}
}

func (p *email) Channel() models.NotificationChannel {
	return models.NotificationChannelEmail
}

func (p *email) Send(delivery Delivery) Result {
	notification := delivery.Recipient.Email
	if notification == nil || notification.Recipient == "" {
		return Result{}
	}
	message := models.Email{
		UserID:         delivery.Recipient.UserID,
		NotificationID: delivery.NotificationID,
		Type:           delivery.Notification.Type,
		ConversationID: delivery.Notification.ConversationID,
		Recipient:      notification.Recipient,
		Subject:        notification.Subject,
		Content:        notification.Content,
	}
	if notification.Delay > 0 {
		return Result{Sent: p.queue.SendAfter(message, notification.Delay)}
	}
	return Result{Sent: p.queue.Send(message)}
}
