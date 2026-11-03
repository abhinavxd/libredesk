package channels

import (
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/notification/models"
	"github.com/volatiletech/null/v9"
)

type emailQueueStub struct {
	sent    []models.Email
	delayed []models.Email
	delay   time.Duration
}

func (q *emailQueueStub) Send(email models.Email) bool {
	q.sent = append(q.sent, email)
	return true
}

func (q *emailQueueStub) SendAfter(email models.Email, delay time.Duration) bool {
	q.delayed = append(q.delayed, email)
	q.delay = delay
	return true
}

func TestEmailProviderSkipsMissingRecipient(t *testing.T) {
	for _, email := range []*models.EmailNotification{nil, {}} {
		queue := &emailQueueStub{}
		result := NewEmail(queue).Send(Delivery{Recipient: models.Recipient{UserID: 42, Email: email}})
		if result.Sent || len(queue.sent) != 0 || len(queue.delayed) != 0 {
			t.Fatalf("unexpected delivery: result=%#v queue=%#v", result, queue)
		}
	}
}

func TestEmailProviderSendsImmediately(t *testing.T) {
	queue := &emailQueueStub{}
	result := NewEmail(queue).Send(Delivery{
		Recipient: models.Recipient{UserID: 42, Email: &models.EmailNotification{
			Recipient: "agent@example.com",
			Subject:   "Assigned",
			Content:   "Conversation assigned",
		}},
		Notification: models.Notification{Type: models.NotificationTypeAssignment},
	})
	if !result.Sent || len(queue.sent) != 1 || queue.sent[0].Recipient != "agent@example.com" {
		t.Fatalf("unexpected delivery: result=%#v queue=%#v", result, queue)
	}
}

func TestEmailProviderSchedulesDelay(t *testing.T) {
	queue := &emailQueueStub{}
	result := NewEmail(queue).Send(Delivery{
		Recipient: models.Recipient{UserID: 42, Email: &models.EmailNotification{
			Recipient: "agent@example.com",
			Delay:     time.Minute,
		}},
	})
	if !result.Sent || len(queue.delayed) != 1 || queue.delay != time.Minute {
		t.Fatalf("unexpected delivery: result=%#v queue=%#v", result, queue)
	}
}

func TestEmailProviderKeepsNotificationAndMessageReferences(t *testing.T) {
	for _, delay := range []time.Duration{0, 5 * time.Minute} {
		t.Run(delay.String(), func(t *testing.T) {
			queue := &emailQueueStub{}
			result := NewEmail(queue).Send(Delivery{
				Recipient: models.Recipient{UserID: 42, Email: &models.EmailNotification{
					Recipient: "agent@example.com", Subject: "New reply", Content: "Latest reply", Delay: delay,
				}},
				NotificationID: null.IntFrom(7),
				Notification: models.Notification{
					Type: models.NotificationTypeNewReply, ConversationID: null.IntFrom(8), MessageID: null.IntFrom(9),
				},
			})
			if !result.Sent {
				t.Fatal("email was not accepted by the queue")
			}
			messages := queue.sent
			if delay > 0 {
				messages = queue.delayed
			}
			want := models.Email{
				UserID: 42, NotificationID: null.IntFrom(7), Type: models.NotificationTypeNewReply,
				ConversationID: null.IntFrom(8), MessageID: null.IntFrom(9),
				Recipient: "agent@example.com", Subject: "New reply", Content: "Latest reply",
			}
			if len(queue.sent)+len(queue.delayed) != 1 || len(messages) != 1 || messages[0] != want || queue.delay != delay {
				t.Fatalf("queued messages = %#v, delay = %v, want %#v after %v", messages, queue.delay, want, delay)
			}
		})
	}
}
