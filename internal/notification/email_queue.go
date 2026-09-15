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

type replyNotificationStore interface {
	ReleaseReplyNotification(conversationID, userID int, messageCreatedAt time.Time)
}

type EmailSender interface {
	Send(Message) error
	SendSync(Message) error
}

type emailQueueQueries struct {
	IsSeen  *sqlx.Stmt `query:"is-notification-seen"`
	Enqueue *sqlx.Stmt `query:"enqueue-notification-email"`
	Claim   *sqlx.Stmt `query:"claim-due-notification-emails"`
	Delete  *sqlx.Stmt `query:"delete-claimed-notification-email"`
	Retry   *sqlx.Stmt `query:"retry-claimed-notification-email"`
}

type queuedEmail struct {
	ID               int64                   `db:"id"`
	ClaimedAt        time.Time               `db:"updated_at"`
	UserID           int                     `db:"user_id"`
	NotificationID   null.Int                `db:"notification_id"`
	Type             models.NotificationType `db:"notification_type"`
	ConversationID   null.Int                `db:"conversation_id"`
	Attempts         int                     `db:"attempts"`
	Recipient        string                  `db:"recipient_email"`
	Subject          string                  `db:"subject"`
	Content          string                  `db:"content"`
	MessageCreatedAt null.Time               `db:"message_created_at"`
}

type EmailQueue struct {
	q                 emailQueueQueries
	outbound          EmailSender
	conversationStore replyNotificationStore
	lo                *logf.Logger
}

type EmailQueueOpts struct {
	DB       *sqlx.DB
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
		outbound: opts.Outbound,
		lo:       opts.Lo,
	}, nil
}

func (q *EmailQueue) SetConversationStore(store replyNotificationStore) {
	q.conversationStore = store
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
		e.Subject, e.Content, time.Now().Add(delay), e.MessageCreatedAt); err != nil {
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
				seen, err := q.seen(e)
				if err != nil {
					continue
				}
				if seen {
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

func (q *EmailQueue) seen(e queuedEmail) (bool, error) {
	var seen bool
	if err := q.q.IsSeen.Get(&seen, e.UserID, e.NotificationID, e.ConversationID, e.MessageCreatedAt); err != nil {
		q.lo.Error("error checking notification seen state", "user_id", e.UserID, "type", e.Type, "error", err)
		return false, err
	}
	return seen, nil
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
			q.abandon(e)
		} else {
			q.retry(e)
		}
		return false
	}
	q.delete(e)
	return true
}

func (q *EmailQueue) abandon(e queuedEmail) {
	if !q.delete(e) || e.NotificationID.Valid || !e.ConversationID.Valid || q.conversationStore == nil {
		return
	}
	switch e.Type {
	case models.NotificationTypeNewReply, models.NotificationTypeNewReplyParticipating, models.NotificationTypeConversationReopened:
		q.conversationStore.ReleaseReplyNotification(e.ConversationID.Int, e.UserID, e.MessageCreatedAt.Time)
	}
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
