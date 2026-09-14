package notifier

import (
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/notification/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

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
	queue, err := NewEmailQueue(EmailQueueOpts{DB: db, Lo: &lo})
	if err != nil {
		t.Fatal(err)
	}
	d := NewDispatcher(DispatcherOpts{
		EmailQueue:   queue,
		EmailEnabled: true,
		Prefs:        fakePreferences{channels: map[int][]models.NotificationChannel{userID: {models.NotificationChannelEmail}}},
	})
	messageTime := time.Now().UTC().Add(-time.Minute).Truncate(time.Microsecond)
	n := Notification{
		Type:             models.NotificationTypeNewReply,
		RecipientIDs:     []int{userID},
		ConversationID:   null.IntFrom(convID),
		MessageCreatedAt: null.TimeFrom(messageTime),
		Email:            &EmailNotification{Recipients: []string{"read@example.com"}, Subject: "Reply", Content: "First reply"},
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
			d.SendAfter(n, time.Minute)
			db.MustExec(`UPDATE notification_email_queue SET send_at = now() - interval '1 second'`)
			due := queue.due()
			if len(due) != 1 {
				t.Fatalf("queued emails = %d, want 1", len(due))
			}
			if got := queue.seen(due[0]); got != tt.want {
				t.Fatalf("seen = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("coalesced reply uses latest message", func(t *testing.T) {
		db.MustExec(`UPDATE conversation_last_seen SET last_seen_at = $1`, messageTime.Add(time.Second))
		d.SendAfter(n, time.Minute)
		n.MessageCreatedAt = null.TimeFrom(messageTime.Add(2 * time.Second))
		n.Email.Content = "Second reply"
		d.SendAfter(n, time.Minute)
		db.MustExec(`UPDATE notification_email_queue SET send_at = now() - interval '1 second'`)
		due := queue.due()
		if len(due) != 1 || due[0].Content != "Second reply" {
			t.Fatalf("unexpected coalesced emails: %#v", due)
		}
		if queue.seen(due[0]) {
			t.Fatal("unread second reply suppressed")
		}
		db.MustExec(`UPDATE conversation_last_seen SET last_seen_at = $1`, messageTime.Add(3*time.Second))
		if !queue.seen(due[0]) {
			t.Fatal("read second reply was not suppressed")
		}
	})

	t.Run("legacy email without message time", func(t *testing.T) {
		n.MessageCreatedAt = null.Time{}
		d.SendAfter(n, time.Minute)
		db.MustExec(`UPDATE notification_email_queue SET send_at = now() - interval '1 second'`)
		due := queue.due()
		if len(due) != 1 {
			t.Fatalf("queued emails = %d, want 1", len(due))
		}
		if queue.seen(due[0]) {
			t.Fatal("legacy unread email suppressed")
		}
		db.MustExec(`UPDATE conversation_last_seen SET last_seen_at = now()`)
		if !queue.seen(due[0]) {
			t.Fatal("legacy read email was not suppressed")
		}
	})
}

func TestDispatcherReportsQueuedEmailRecipients(t *testing.T) {
	db := testutil.NewDB(t, "reply_email_recipients")
	var userID int
	if err := db.Get(&userID, `INSERT INTO users (type, email, first_name, last_name) VALUES ('agent', 'queued@example.com', 'Agent', '') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	lo := logf.New(logf.Opts{})
	queue, err := NewEmailQueue(EmailQueueOpts{DB: db, Lo: &lo})
	if err != nil {
		t.Fatal(err)
	}
	d := NewDispatcher(DispatcherOpts{EmailQueue: queue, EmailEnabled: true, Prefs: fakePreferences{channels: map[int][]models.NotificationChannel{userID: {models.NotificationChannelEmail}}}})
	n := Notification{Type: models.NotificationTypeNewReply, RecipientIDs: []int{userID}}
	email := []EmailNotification{{Recipients: []string{"queued@example.com"}, Subject: "Reply", Content: "Reply"}}
	if got := d.SendWithEmailsAfter(n, nil, time.Minute, d.EnabledChannels(n.RecipientIDs, n.Type)); len(got) != 0 {
		t.Fatalf("missing email reported as notified: %v", got)
	}
	if got := d.SendWithEmailsAfter(n, email, time.Minute, d.EnabledChannels(n.RecipientIDs, n.Type)); len(got) != 1 || got[0] != userID {
		t.Fatalf("queued email recipients = %v", got)
	}
	if err := queue.q.Enqueue.Close(); err != nil {
		t.Fatal(err)
	}
	if got := d.SendWithEmailsAfter(n, email, time.Minute, d.EnabledChannels(n.RecipientIDs, n.Type)); len(got) != 0 {
		t.Fatalf("failed enqueue reported as notified: %v", got)
	}
}
