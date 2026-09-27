package channels

import (
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/notification/models"
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
