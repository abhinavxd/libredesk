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

func TestDelayedReplyEmailKeepsMessageReference(t *testing.T) {
	f := newReplyFixture(t, "reply_email_reference")
	queue, err := NewEmailQueue(EmailQueueOpts{DB: f.db, Lo: f.manager.lo, Checker: f.checker})
	if err != nil {
		t.Fatal(err)
	}
	email := models.Email{UserID: f.userID, Type: models.NotificationTypeNewReply,
		ConversationID: null.IntFrom(f.convID), MessageID: null.IntFrom(f.messageIDs[0]),
		Recipient: "agent@example.com", Subject: "Reply", Content: "First reply"}
	f.markSeen(t, 0)
	if !queue.SendAfter(email, time.Minute) {
		t.Fatal("enqueue failed")
	}
	var queued queuedEmail
	if err := f.db.Get(&queued, `SELECT user_id, notification_type, conversation_id, message_id FROM notification_email_queue WHERE user_id = $1`, f.userID); err != nil {
		t.Fatal(err)
	}
	ref := models.NotificationReference{Type: queued.Type, ConversationID: queued.ConversationID, MessageID: queued.MessageID}
	deliver, err := f.checker.ShouldDeliver(queued.UserID, ref, models.NotificationChannelEmail)
	if err != nil || deliver {
		t.Fatalf("seen before enqueue: deliver=%v err=%v", deliver, err)
	}
	email.MessageID = null.IntFrom(f.messageIDs[1])
	email.Content = "Second reply"
	if !queue.SendAfter(email, time.Minute) {
		t.Fatal("replacement enqueue failed")
	}
	if err := f.db.Get(&queued, `SELECT user_id, notification_type, conversation_id, message_id, content FROM notification_email_queue WHERE user_id = $1`, f.userID); err != nil {
		t.Fatal(err)
	}
	if queued.MessageID != email.MessageID || queued.Content != email.Content {
		t.Fatalf("replacement = %#v", queued)
	}
	ref.MessageID = queued.MessageID
	deliver, err = f.checker.ShouldDeliver(queued.UserID, ref, models.NotificationChannelEmail)
	if err != nil || !deliver {
		t.Fatalf("unseen replacement: deliver=%v err=%v", deliver, err)
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
