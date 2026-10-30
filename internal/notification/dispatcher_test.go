package notifier

import (
	"errors"
	"slices"
	"testing"

	"github.com/abhinavxd/libredesk/internal/notification/channels"
	"github.com/abhinavxd/libredesk/internal/notification/models"
	"github.com/volatiletech/null/v9"
)

type fakePreferences struct {
	channels map[int][]models.NotificationChannel
	err      error
}

func (p fakePreferences) EnabledChannels([]int, models.NotificationType) (map[int][]models.NotificationChannel, error) {
	return p.channels, p.err
}

type fakePushSender struct {
	userID  int
	payload models.PushPayload
}

type fakeChannelProvider struct {
	channel        models.NotificationChannel
	result         channels.Result
	notificationID null.Int
	deliveries     []channels.Delivery
}

func (p *fakePushSender) Send(userID int, payload models.PushPayload, _ models.NotificationReference) bool {
	p.userID = userID
	p.payload = payload
	return true
}

func (p *fakeChannelProvider) Channel() models.NotificationChannel {
	return p.channel
}

func (p *fakeChannelProvider) Send(delivery channels.Delivery) channels.Result {
	p.notificationID = delivery.NotificationID
	p.deliveries = append(p.deliveries, delivery)
	return p.result
}

func TestDispatcherPassesResultsThroughProviders(t *testing.T) {
	inApp := &fakeChannelProvider{
		channel: models.NotificationChannelInApp,
		result:  channels.Result{Sent: true, NotificationID: null.IntFrom(7)},
	}
	email := &fakeChannelProvider{channel: models.NotificationChannelEmail}
	d := NewDispatcher(DispatcherOpts{
		Pipeline: channels.NewPipeline(email, inApp),
		Prefs: fakePreferences{channels: map[int][]models.NotificationChannel{
			42: {models.NotificationChannelInApp, models.NotificationChannelEmail},
		}},
	})

	results, err := d.Send(models.Notification{Type: models.NotificationTypeMention, Recipients: []models.Recipient{{UserID: 42}}})
	if err != nil {
		t.Fatal(err)
	}

	if email.notificationID.Int != 7 {
		t.Fatalf("email notification ID = %d, want 7", email.notificationID.Int)
	}
	if len(results) != 1 || !slices.Equal(results[0].Channels, []models.NotificationChannel{models.NotificationChannelInApp}) {
		t.Fatalf("delivery results = %#v", results)
	}
}

func TestDispatcherSendsPushUsingNotificationRoute(t *testing.T) {
	push := &fakePushSender{}
	d := NewDispatcher(DispatcherOpts{
		Prefs: fakePreferences{channels: map[int][]models.NotificationChannel{
			42: {models.NotificationChannelPush},
		}},
		Pipeline: channels.NewPipeline(channels.NewPush(push)),
	})
	d.Send(models.Notification{
		Type:             models.NotificationTypeMention,
		Recipients:       []models.Recipient{{UserID: 42}},
		Title:            "You were mentioned",
		Body:             null.StringFrom("A teammate mentioned you"),
		ConversationUUID: "conversation-uuid",
		MessageUUID:      "message-uuid",
	})

	if push.userID != 42 {
		t.Fatalf("sent to user %d, want 42", push.userID)
	}
	if push.payload.URL != "/inboxes/mentioned/conversation/conversation-uuid?scrollTo=message-uuid" {
		t.Fatalf("push URL = %q", push.payload.URL)
	}
	if push.payload.Title != "You were mentioned" || push.payload.Body != "A teammate mentioned you" {
		t.Fatalf("unexpected push payload: %#v", push.payload)
	}
}

func TestDispatcherDoesNotSendPushWhenDisabled(t *testing.T) {
	push := &fakePushSender{}
	d := NewDispatcher(DispatcherOpts{
		Prefs: fakePreferences{channels: map[int][]models.NotificationChannel{
			42: nil,
		}},
		Pipeline: channels.NewPipeline(channels.NewPush(push)),
	})
	d.Send(models.Notification{
		Type:       models.NotificationTypeMention,
		Recipients: []models.Recipient{{UserID: 42}},
		Title:      "You were mentioned",
	})

	if push.userID != 0 {
		t.Fatalf("sent push to user %d when push is disabled", push.userID)
	}
}

func TestDispatcherReturnsPreferenceLookupFailure(t *testing.T) {
	lookupErr := errors.New("lookup failed")
	push := &fakePushSender{}
	d := NewDispatcher(DispatcherOpts{
		Prefs:    fakePreferences{err: lookupErr},
		Pipeline: channels.NewPipeline(channels.NewPush(push)),
	})
	_, err := d.Send(models.Notification{
		Type:       models.NotificationTypeMention,
		Recipients: []models.Recipient{{UserID: 42}},
		Title:      "You were mentioned",
	})

	if !errors.Is(err, lookupErr) {
		t.Fatalf("error = %v, want %v", err, lookupErr)
	}
	if push.userID != 0 {
		t.Fatalf("sent push to user %d after preference lookup failed", push.userID)
	}
}

func TestDispatcherKeepsRecipientPreferencesAndDeliveryResultsSeparate(t *testing.T) {
	inApp := &fakeChannelProvider{
		channel: models.NotificationChannelInApp,
		result:  channels.Result{Sent: true, NotificationID: null.IntFrom(7)},
	}
	email := &fakeChannelProvider{channel: models.NotificationChannelEmail}
	push := &fakeChannelProvider{channel: models.NotificationChannelPush, result: channels.Result{Sent: true}}
	d := NewDispatcher(DispatcherOpts{
		Pipeline: channels.NewPipeline(push, email, inApp),
		Prefs: fakePreferences{channels: map[int][]models.NotificationChannel{
			42: {models.NotificationChannelInApp, models.NotificationChannelEmail, models.NotificationChannelPush},
			43: {models.NotificationChannelEmail},
			44: {models.NotificationChannelPush},
		}},
	})
	results, err := d.Send(models.Notification{
		Type: models.NotificationTypeNewReply,
		Recipients: []models.Recipient{
			{UserID: 42, Email: &models.EmailNotification{Recipient: "agent-a@example.com", Content: "Hi Agent A"}},
			{UserID: 43, Email: &models.EmailNotification{Recipient: "agent-b@example.com", Content: "Hi Agent B"}},
			{UserID: 44}, {UserID: 45},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []models.DeliveryResult{
		{RecipientID: 42, Channels: []models.NotificationChannel{models.NotificationChannelInApp, models.NotificationChannelPush}},
		{RecipientID: 43},
		{RecipientID: 44, Channels: []models.NotificationChannel{models.NotificationChannelPush}},
		{RecipientID: 45},
	}
	if !slices.EqualFunc(results, want, func(got, want models.DeliveryResult) bool {
		return got.RecipientID == want.RecipientID && slices.Equal(got.Channels, want.Channels)
	}) {
		t.Fatalf("delivery results = %#v, want %#v", results, want)
	}
	if len(inApp.deliveries) != 1 || len(email.deliveries) != 2 || len(push.deliveries) != 2 {
		t.Fatalf("deliveries: in-app=%d email=%d push=%d", len(inApp.deliveries), len(email.deliveries), len(push.deliveries))
	}
	for index, userID := range []int{42, 43} {
		if email.deliveries[index].Recipient.UserID != userID {
			t.Fatalf("email went to user %d, want %d", email.deliveries[index].Recipient.UserID, userID)
		}
		if email.deliveries[index].NotificationID.Valid != (index == 0) {
			t.Fatal("email inherited another recipient's notification ID")
		}
	}
	if email.deliveries[0].Recipient.Email.Recipient != "agent-a@example.com" || email.deliveries[0].Recipient.Email.Content != "Hi Agent A" ||
		email.deliveries[1].Recipient.Email.Recipient != "agent-b@example.com" || email.deliveries[1].Recipient.Email.Content != "Hi Agent B" {
		t.Fatal("email delivery mixed recipients or personalized content")
	}
	if push.deliveries[0].Recipient.UserID != 42 || push.deliveries[1].Recipient.UserID != 44 || push.deliveries[1].NotificationID.Valid {
		t.Fatal("push delivery inherited another recipient or notification ID")
	}
}
