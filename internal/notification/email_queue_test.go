package notifier

import (
	"errors"
	"testing"
	"time"

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

func TestDispatcherReportsDelayedEmailDelivery(t *testing.T) {
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
	d := NewDispatcher(DispatcherOpts{EmailQueue: queue, EmailEnabled: true, Prefs: fakePreferences{channels: map[int][]models.NotificationChannel{userID: {models.NotificationChannelEmail}}}})
	n := Notification{Type: models.NotificationTypeNewReply, RecipientIDs: []int{userID}, ConversationID: null.IntFrom(convID)}
	email := []EmailNotification{{Recipients: []string{"queued@example.com"}, Subject: "Reply", Content: "Reply"}}
	var deliveries []bool
	onDelivery := func(_ int, delivered bool) {
		deliveries = append(deliveries, delivered)
	}
	d.SendWithEmailsAfter(n, nil, time.Minute, d.EnabledChannels(n.RecipientIDs, n.Type), onDelivery)
	if len(deliveries) != 1 || deliveries[0] {
		t.Fatalf("missing email deliveries = %v", deliveries)
	}
	deliveries = nil
	d.SendWithEmailsAfter(n, email, time.Minute, d.EnabledChannels(n.RecipientIDs, n.Type), onDelivery)
	if len(deliveries) != 0 {
		t.Fatalf("queued email completed before delivery: %v", deliveries)
	}
	db.MustExec(`UPDATE notification_email_queue SET send_at = now() - interval '1 second'`)
	due := queue.due()
	if len(due) != 1 {
		t.Fatalf("queued emails = %d, want 1", len(due))
	}
	queue.deliver(due[0])
	if len(deliveries) != 1 || !deliveries[0] {
		t.Fatalf("successful email deliveries = %v", deliveries)
	}
	var queued int
	if err := db.Get(&queued, `SELECT COUNT(*) FROM notification_email_queue`); err != nil {
		t.Fatal(err)
	}
	if queued != 0 {
		t.Fatalf("queued emails after delivery = %d, want 0", queued)
	}
	deliveries = nil
	provider.err = errors.New("SMTP unavailable")
	d.SendWithEmailsAfter(n, email, time.Minute, d.EnabledChannels(n.RecipientIDs, n.Type), onDelivery)
	db.MustExec(`UPDATE notification_email_queue SET send_at = now() - interval '1 second'`)
	due = queue.due()
	if len(due) != 1 {
		t.Fatalf("queued emails = %d, want 1", len(due))
	}
	queue.deliver(due[0])
	if len(deliveries) != 1 || deliveries[0] {
		t.Fatalf("failed email deliveries = %v", deliveries)
	}
	if err := db.Get(&queued, `SELECT COUNT(*) FROM notification_email_queue`); err != nil {
		t.Fatal(err)
	}
	if queued != 1 {
		t.Fatalf("queued emails after failure = %d, want 1", queued)
	}
	for attempt := 2; attempt <= emailQueueMaxAttempts; attempt++ {
		db.MustExec(`UPDATE notification_email_queue SET send_at = now() - interval '1 second'`)
		due = queue.due()
		if len(due) != 1 {
			t.Fatalf("queued emails for attempt %d = %d, want 1", attempt, len(due))
		}
		queue.deliver(due[0])
	}
	if err := db.Get(&queued, `SELECT COUNT(*) FROM notification_email_queue`); err != nil {
		t.Fatal(err)
	}
	if queued != 0 {
		t.Fatalf("queued emails after final failure = %d, want 0", queued)
	}
	deliveries = nil
	d.SendWithEmailsAfter(n, email, time.Minute, d.EnabledChannels(n.RecipientIDs, n.Type), onDelivery)
	db.MustExec(`UPDATE notification_email_queue SET attempts = $1`, emailQueueMaxAttempts-1)
	d.SendWithEmailsAfter(n, email, time.Minute, d.EnabledChannels(n.RecipientIDs, n.Type), onDelivery)
	var attempts int
	if err := db.Get(&attempts, `SELECT attempts FROM notification_email_queue`); err != nil {
		t.Fatal(err)
	}
	if attempts != 0 {
		t.Fatalf("attempts after replacement = %d, want 0", attempts)
	}
	if err := queue.q.Enqueue.Close(); err != nil {
		t.Fatal(err)
	}
	deliveries = nil
	d.SendWithEmailsAfter(n, email, time.Minute, d.EnabledChannels(n.RecipientIDs, n.Type), onDelivery)
	if len(deliveries) != 1 || deliveries[0] {
		t.Fatalf("failed enqueue deliveries = %v", deliveries)
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
	email := queuedEmail{
		UserID:    userID,
		Type:      models.NotificationTypeNewReply,
		Recipient: "recover@example.com",
		Subject:   "Reply",
		Content:   "Pending reply",
	}
	if !queue.SendAfter(email, -time.Second, nil) {
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
