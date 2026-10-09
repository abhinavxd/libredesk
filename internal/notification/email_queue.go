package notifier

import (
	"context"
	"time"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/notification/models"
	"github.com/jmoiron/sqlx"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

const (
	emailQueueTick        = 15 * time.Second
	emailQueueBatch       = 500
	emailQueueClaimLease  = 5 * time.Minute
	emailQueueMaxAttempts = 3
	emailQueueRetryDelay  = time.Minute
)

type EmailSender interface {
	Send(Message) error
	SendSync(Message) error
}

type emailQueueQueries struct {
	Enqueue *sqlx.Stmt `query:"enqueue-notification-email"`
	Claim   *sqlx.Stmt `query:"claim-due-notification-emails"`
	Delete  *sqlx.Stmt `query:"delete-claimed-notification-email"`
	Retry   *sqlx.Stmt `query:"retry-claimed-notification-email"`
}

type queuedEmail struct {
	ID             int64                   `db:"id"`
	ClaimedAt      time.Time               `db:"updated_at"`
	UserID         int                     `db:"user_id"`
	NotificationID null.Int                `db:"notification_id"`
	Type           models.NotificationType `db:"notification_type"`
	ConversationID null.Int                `db:"conversation_id"`
	MessageID      null.Int                `db:"message_id"`
	Attempts       int                     `db:"attempts"`
	Recipient      string                  `db:"recipient_email"`
	Subject        string                  `db:"subject"`
	Content        string                  `db:"content"`
}

type EmailQueue struct {
	q        emailQueueQueries
	checker  DeliveryChecker
	outbound EmailSender
	lo       *logf.Logger
}

type EmailQueueOpts struct {
	DB       *sqlx.DB
	Checker  DeliveryChecker
	Outbound EmailSender
	Lo       *logf.Logger
}

func NewEmailQueue(opts EmailQueueOpts) (*EmailQueue, error) {
	var q emailQueueQueries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, queriesFS); err != nil {
		return nil, err
	}
	return &EmailQueue{
		q:        q,
		checker:  opts.Checker,
		outbound: opts.Outbound,
		lo:       opts.Lo,
	}, nil
}

func (q *EmailQueue) Send(e models.Email) bool {
	if err := q.outbound.Send(Message{
		RecipientEmails: []string{e.Recipient},
		Subject:         e.Subject,
		Content:         e.Content,
		Provider:        ProviderEmail,
	}); err != nil {
		q.lo.Error("error sending notification email", "user_id", e.UserID, "type", e.Type, "error", err)
		return false
	}
	return true
}

func (q *EmailQueue) SendAfter(e models.Email, delay time.Duration) bool {
	if _, err := q.q.Enqueue.Exec(e.UserID, e.NotificationID, e.Type, e.ConversationID, e.Recipient,
		e.Subject, e.Content, time.Now().Add(delay), e.MessageID); err != nil {
		q.lo.Error("error queueing notification email", "user_id", e.UserID, "type", e.Type, "error", err)
		return false
	}
	return true
}

func (q *EmailQueue) Run(ctx context.Context) {
	ticker := time.NewTicker(emailQueueTick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, e := range q.due() {
				deliver, err := q.checker.ShouldDeliver(e.UserID, models.NotificationReference{
					Type: e.Type, ConversationID: e.ConversationID, MessageID: e.MessageID, NotificationID: e.NotificationID,
				}, models.NotificationChannelEmail)
				if err != nil {
					q.lo.Error("error checking notification email eligibility", "user_id", e.UserID, "error", err)
					continue
				}
				if !deliver {
					q.delete(e)
					continue
				}
				q.deliver(e)
			}
		}
	}
}

func (q *EmailQueue) due() []queuedEmail {
	var due []queuedEmail
	if err := q.q.Claim.Select(&due, emailQueueBatch, time.Now().Add(emailQueueClaimLease)); err != nil {
		q.lo.Error("error claiming due notification emails", "error", err)
		return nil
	}
	return due
}

func (q *EmailQueue) deliver(e queuedEmail) bool {
	if err := q.outbound.SendSync(Message{
		RecipientEmails: []string{e.Recipient},
		Subject:         e.Subject,
		Content:         e.Content,
		Provider:        ProviderEmail,
	}); err != nil {
		q.lo.Error("error delivering notification email", "user_id", e.UserID, "type", e.Type, "error", err)
		if e.Attempts+1 >= emailQueueMaxAttempts {
			q.delete(e)
		} else {
			q.retry(e)
		}
		return false
	}
	q.delete(e)
	return true
}

func (q *EmailQueue) delete(e queuedEmail) bool {
	result, err := q.q.Delete.Exec(e.ID, e.ClaimedAt)
	if err != nil {
		q.lo.Error("error deleting delivered notification email", "user_id", e.UserID, "type", e.Type, "error", err)
		return false
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		q.lo.Error("error counting deleted notification emails", "user_id", e.UserID, "type", e.Type, "error", err)
		return false
	}
	return deleted > 0
}

func (q *EmailQueue) retry(e queuedEmail) {
	if _, err := q.q.Retry.Exec(e.ID, time.Now().Add(emailQueueRetryDelay), e.ClaimedAt); err != nil {
		q.lo.Error("error scheduling notification email retry", "user_id", e.UserID, "type", e.Type, "error", err)
	}
}
