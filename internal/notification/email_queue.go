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
	emailQueueTick  = 15 * time.Second
	emailQueueBatch = 500
)

type emailQueueQueries struct {
	IsSeen  *sqlx.Stmt `query:"is-notification-seen"`
	Enqueue *sqlx.Stmt `query:"enqueue-notification-email"`
	Dequeue *sqlx.Stmt `query:"dequeue-due-notification-emails"`
}

type queuedEmail struct {
	UserID         int                     `db:"user_id"`
	NotificationID null.Int                `db:"notification_id"`
	Type           models.NotificationType `db:"notification_type"`
	ConversationID null.Int                `db:"conversation_id"`
	Recipient      string                  `db:"recipient_email"`
	Subject        string                  `db:"subject"`
	Content        string                  `db:"content"`
	QueuedAt       time.Time               `db:"queued_at"`
}

type EmailQueue struct {
	q        emailQueueQueries
	outbound *Service
	lo       *logf.Logger
}

type EmailQueueOpts struct {
	DB       *sqlx.DB
	Outbound *Service
	Lo       *logf.Logger
}

func NewEmailQueue(opts EmailQueueOpts) (*EmailQueue, error) {
	var q emailQueueQueries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, queriesFS); err != nil {
		return nil, err
	}
	return &EmailQueue{
		q:        q,
		outbound: opts.Outbound,
		lo:       opts.Lo,
	}, nil
}

func (q *EmailQueue) Send(e queuedEmail) {
	if err := q.outbound.Send(Message{
		RecipientEmails: []string{e.Recipient},
		Subject:         e.Subject,
		Content:         e.Content,
		Provider:        ProviderEmail,
	}); err != nil {
		q.lo.Error("error sending notification email", "user_id", e.UserID, "type", e.Type, "error", err)
	}
}

func (q *EmailQueue) SendAfter(e queuedEmail, delay time.Duration) {
	if _, err := q.q.Enqueue.Exec(e.UserID, e.NotificationID, e.Type, e.ConversationID, e.Recipient,
		e.Subject, e.Content, time.Now().Add(delay)); err != nil {
		q.lo.Error("error queueing notification email", "user_id", e.UserID, "type", e.Type, "error", err)
	}
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
				if ctx.Err() != nil {
					return
				}
				if q.seen(e) {
					continue
				}
				q.Send(e)
			}
		}
	}
}

func (q *EmailQueue) due() []queuedEmail {
	var due []queuedEmail
	if err := q.q.Dequeue.Select(&due, emailQueueBatch); err != nil {
		q.lo.Error("error dequeueing due notification emails", "error", err)
		return nil
	}
	return due
}

func (q *EmailQueue) seen(e queuedEmail) bool {
	var seen bool
	if err := q.q.IsSeen.Get(&seen, e.UserID, e.NotificationID, e.ConversationID, e.QueuedAt); err != nil {
		q.lo.Error("error checking notification seen state", "user_id", e.UserID, "type", e.Type, "error", err)
		return false
	}
	return seen
}
