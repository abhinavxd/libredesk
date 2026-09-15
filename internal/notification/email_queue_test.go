package notifier

import (
	"errors"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/notification/channels"
	"github.com/abhinavxd/libredesk/internal/notification/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

type emailDeliveryProvider struct {
	err error
}

func (p *emailDeliveryProvider) Send(Message) error {
	return p.err
}

func (p *emailDeliveryProvider) Name() string {
	return ProviderEmail
}

func TestDelayedReplyEmailUsesMessageTime(t *testing.T) {
	db := testutil.NewDB(t, "reply_email_time")
	var userID, inboxID, convID int
	if err := db.Get(&userID, `INSERT INTO users (type, email, first_name, last_name) VALUES ('agent', 'read@example.com', 'Agent', '') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&inboxID, `INSERT INTO inboxes (name, channel) VALUES ('Test', 'email') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&convID, `INSERT INTO conversations (contact_id, inbox_id, status_id) VALUES ($1, $2, (SELECT id FROM conversation_statuses LIMIT 1)) RETURNING id`, userID, inboxID); err != nil {
		t.Fatal(err)
	}
	lo := logf.New(logf.Opts{})
	provider := &emailDeliveryProvider{}
	outbound := NewService(map[string]Notifier{ProviderEmail: provider}, 1, 1, &lo)
	queue, err := NewEmailQueue(EmailQueueOpts{DB: db, Outbound: outbound, Lo: &lo})
	if err != nil {
		t.Fatal(err)
	}
	d := NewDispatcher(DispatcherOpts{
		Pipeline: channels.NewPipeline(channels.NewEmail(queue)),
		Prefs:    fakePreferences{channels: map[int][]models.NotificationChannel{userID: {models.NotificationChannelEmail}}},
	})
	messageTime := time.Now().UTC().Add(-time.Minute).Truncate(time.Microsecond)
	n := models.Notification{
		Type: models.NotificationTypeNewReply,
		Recipients: []models.Recipient{{
			UserID: userID,
			Email: &models.EmailNotification{
				Recipient: "read@example.com",
				Subject:   "Reply",
				Content:   "First reply",
				Delay:     time.Minute,
			},
		}},
		ConversationID:   null.IntFrom(convID),
		MessageCreatedAt: null.TimeFrom(messageTime),
	}
	db.MustExec(`INSERT INTO conversation_last_seen (user_id, conversation_id, last_seen_at) VALUES ($1, $2, $3)`, userID, convID, messageTime.Add(-time.Second))
	for _, tt := range []struct {
		name     string
		readTime time.Time
		want     bool
	}{
		{"read before reply", messageTime.Add(-time.Second), false},
		{"read at reply", messageTime, true},
		{"read before enqueue", messageTime.Add(time.Second), true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			db.MustExec(`UPDATE conversation_last_seen SET last_seen_at = $1`, tt.readTime)
			d.Send(n)
			db.MustExec(`UPDATE notification_email_queue SET send_at = now() - interval '1 second'`)
			due := queue.due()
			if len(due) != 1 {
				t.Fatalf("queued emails = %d, want 1", len(due))
			}
			got, err := queue.seen(due[0])
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("seen = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("coalesced reply uses latest message", func(t *testing.T) {
		db.MustExec(`UPDATE conversation_last_seen SET last_seen_at = $1`, messageTime.Add(time.Second))
		d.Send(n)
		var firstSendAt time.Time
		if err := db.Get(&firstSendAt, `SELECT send_at FROM notification_email_queue`); err != nil {
			t.Fatal(err)
		}
		n.MessageCreatedAt = null.TimeFrom(messageTime.Add(2 * time.Second))
		n.Recipients[0].Email.Content = "Second reply"
		n.Recipients[0].Email.Delay = 2 * time.Minute
		d.Send(n)
		var secondSendAt time.Time
		if err := db.Get(&secondSendAt, `SELECT send_at FROM notification_email_queue`); err != nil {
			t.Fatal(err)
		}
		if !secondSendAt.After(firstSendAt) {
			t.Fatalf("coalesced email send time = %v, want after %v", secondSendAt, firstSendAt)
		}
		db.MustExec(`UPDATE notification_email_queue SET send_at = now() - interval '1 second'`)
		due := queue.due()
		if len(due) != 1 || due[0].Content != "Second reply" {
			t.Fatalf("unexpected coalesced emails: %#v", due)
		}
		seen, err := queue.seen(due[0])
		if err != nil {
			t.Fatal(err)
		}
		if seen {
			t.Fatal("unread second reply suppressed")
		}
		db.MustExec(`UPDATE conversation_last_seen SET last_seen_at = $1`, messageTime.Add(3*time.Second))
		seen, err = queue.seen(due[0])
		if err != nil {
			t.Fatal(err)
		}
		if !seen {
			t.Fatal("read second reply was not suppressed")
		}
	})

	t.Run("legacy email without message time", func(t *testing.T) {
		n.MessageCreatedAt = null.Time{}
		d.Send(n)
		db.MustExec(`UPDATE notification_email_queue SET send_at = now() - interval '1 second'`)
		due := queue.due()
		if len(due) != 1 {
			t.Fatalf("queued emails = %d, want 1", len(due))
		}
		seen, err := queue.seen(due[0])
		if err != nil {
			t.Fatal(err)
		}
		if seen {
			t.Fatal("legacy unread email suppressed")
		}
		db.MustExec(`UPDATE conversation_last_seen SET last_seen_at = now()`)
		seen, err = queue.seen(due[0])
		if err != nil {
			t.Fatal(err)
		}
		if !seen {
			t.Fatal("legacy read email was not suppressed")
		}
	})
}

func TestDelayedEmailSeenCheckReturnsDatabaseError(t *testing.T) {
	db := testutil.NewDB(t, "notification_email_seen_error")
	lo := logf.New(logf.Opts{})
	queue, err := NewEmailQueue(EmailQueueOpts{DB: db, Lo: &lo})
	if err != nil {
		t.Fatal(err)
	}
	db.MustExec(`DROP TABLE conversation_last_seen`)

	if _, err := queue.seen(queuedEmail{}); err == nil {
		t.Fatal("seen-state database error was ignored")
	}
}

func TestDelayedEmailRetriesThenGivesUp(t *testing.T) {
	db := testutil.NewDB(t, "reply_email_recipients")
	var userID, inboxID, convID int
	if err := db.Get(&userID, `INSERT INTO users (type, email, first_name, last_name) VALUES ('agent', 'queued@example.com', 'Agent', '') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&inboxID, `INSERT INTO inboxes (name, channel) VALUES ('Test', 'email') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&convID, `INSERT INTO conversations (contact_id, inbox_id, status_id) VALUES ($1, $2, (SELECT id FROM conversation_statuses LIMIT 1)) RETURNING id`, userID, inboxID); err != nil {
		t.Fatal(err)
	}
	lo := logf.New(logf.Opts{})
	provider := &emailDeliveryProvider{}
	outbound := NewService(map[string]Notifier{ProviderEmail: provider}, 1, 1, &lo)
	queue, err := NewEmailQueue(EmailQueueOpts{DB: db, Outbound: outbound, Lo: &lo})
	if err != nil {
		t.Fatal(err)
	}
	d := NewDispatcher(DispatcherOpts{
		Pipeline: channels.NewPipeline(channels.NewEmail(queue)),
		Prefs:    fakePreferences{channels: map[int][]models.NotificationChannel{userID: {models.NotificationChannelEmail}}},
	})
	n := models.Notification{Type: models.NotificationTypeNewReply, Recipients: []models.Recipient{{UserID: userID}}, ConversationID: null.IntFrom(convID)}
	email := &models.EmailNotification{Recipient: "queued@example.com", Subject: "Reply", Content: "Reply", Delay: time.Minute}
	send := func(email *models.EmailNotification) {
		n.Recipients[0].Email = email
		d.Send(n)
	}
	queued := func() int {
		var count int
		if err := db.Get(&count, `SELECT COUNT(*) FROM notification_email_queue`); err != nil {
			t.Fatal(err)
		}
		return count
	}
	claimOne := func() queuedEmail {
		t.Helper()
		db.MustExec(`UPDATE notification_email_queue SET send_at = now() - interval '1 second'`)
		due := queue.due()
		if len(due) != 1 {
			t.Fatalf("queued emails = %d, want 1", len(due))
		}
		return due[0]
	}

	send(nil)
	if queued() != 0 {
		t.Fatal("a notification without an email body was queued")
	}

	send(email)
	if queued() != 1 {
		t.Fatal("email was not queued")
	}
	if !queue.deliver(claimOne()) {
		t.Fatal("email was not delivered")
	}
	if queued() != 0 {
		t.Fatalf("queued emails after delivery = %d, want 0", queued())
	}

	provider.err = errors.New("SMTP unavailable")
	send(email)
	if queue.deliver(claimOne()) {
		t.Fatal("failed send reported as delivered")
	}
	if queued() != 1 {
		t.Fatalf("queued emails after failure = %d, want 1", queued())
	}
	for attempt := 2; attempt <= emailQueueMaxAttempts; attempt++ {
		queue.deliver(claimOne())
	}
	if queued() != 0 {
		t.Fatalf("queued emails after final failure = %d, want 0", queued())
	}

	send(email)
	db.MustExec(`UPDATE notification_email_queue SET attempts = $1`, emailQueueMaxAttempts-1)
	send(email)
	var attempts int
	if err := db.Get(&attempts, `SELECT attempts FROM notification_email_queue`); err != nil {
		t.Fatal(err)
	}
	if attempts != 0 {
		t.Fatalf("attempts after replacement = %d, want 0", attempts)
	}
}

func TestClaimedEmailRemainsRecoverable(t *testing.T) {
	db := testutil.NewDB(t, "notification_email_recovery")
	var userID int
	if err := db.Get(&userID, `INSERT INTO users (type, email, first_name, last_name) VALUES ('agent', 'recover@example.com', 'Agent', '') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	lo := logf.New(logf.Opts{})
	provider := &emailDeliveryProvider{}
	queue, err := NewEmailQueue(EmailQueueOpts{
		DB:       db,
		Outbound: NewService(map[string]Notifier{ProviderEmail: provider}, 1, 1, &lo),
		Lo:       &lo,
	})
	if err != nil {
		t.Fatal(err)
	}
	email := models.Email{
		UserID:    userID,
		Type:      models.NotificationTypeNewReply,
		Recipient: "recover@example.com",
		Subject:   "Reply",
		Content:   "Pending reply",
	}
	if !queue.SendAfter(email, -time.Second) {
		t.Fatal("queueing email failed")
	}
	claimed := queue.due()
	if len(claimed) != 1 {
		t.Fatalf("claimed emails = %d, want 1", len(claimed))
	}
	var queued int
	if err := db.Get(&queued, `SELECT COUNT(*) FROM notification_email_queue`); err != nil {
		t.Fatal(err)
	}
	if queued != 1 {
		t.Fatalf("queued emails after claim = %d, want 1", queued)
	}
	if due := queue.due(); len(due) != 0 {
		t.Fatalf("emails reclaimed before lease expiry = %d, want 0", len(due))
	}
	db.MustExec(`UPDATE notification_email_queue SET send_at = now() - interval '1 second'`)
	if due := queue.due(); len(due) != 1 {
		t.Fatalf("emails after lease expiry = %d, want 1", len(due))
	}
}
