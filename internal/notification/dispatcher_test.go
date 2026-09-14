package notifier

import (
	"bytes"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/notification/channels"
	"github.com/abhinavxd/libredesk/internal/notification/models"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

type fakePreferences struct {
	channels map[int][]models.NotificationChannel
	err      error
}

func (p fakePreferences) EnabledChannels([]int, models.NotificationType) (map[int][]models.NotificationChannel, error) {
	return p.channels, p.err
}

type fakePushDispatcher struct {
	userID  int
	payload models.PushPayload
}

type fakeChannelProvider struct {
	channel        models.NotificationChannel
	result         channels.Result
	notificationID null.Int
}

func (p *fakePushDispatcher) Send(userID int, payload models.PushPayload) bool {
	p.userID = userID
	p.payload = payload
	return true
}

func (p *fakeChannelProvider) Channel() models.NotificationChannel {
	return p.channel
}

func (p *fakeChannelProvider) Send(delivery channels.Delivery) channels.Result {
	p.notificationID = delivery.NotificationID
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

	results := d.Send(models.Notification{Type: models.NotificationTypeMention, Recipients: []models.Recipient{{UserID: 42}}})

	if email.notificationID.Int != 7 {
		t.Fatalf("email notification ID = %d, want 7", email.notificationID.Int)
	}
	if len(results) != 1 || !slices.Equal(results[0].Channels, []models.NotificationChannel{models.NotificationChannelInApp}) {
		t.Fatalf("delivery results = %#v", results)
	}
}

func TestDispatcherSendsPushUsingNotificationRoute(t *testing.T) {
	push := &fakePushDispatcher{}
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
	push := &fakePushDispatcher{}
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

func TestDispatcherHandlesPreferenceLookupFailure(t *testing.T) {
	var logs bytes.Buffer
	lo := logf.New(logf.Opts{Writer: &logs})
	push := &fakePushDispatcher{}
	d := NewDispatcher(DispatcherOpts{
		Prefs:    fakePreferences{err: errors.New("lookup failed")},
		Pipeline: channels.NewPipeline(channels.NewPush(push)),
		Lo:       &lo,
	})
	d.Send(models.Notification{
		Type:       models.NotificationTypeMention,
		Recipients: []models.Recipient{{UserID: 42}},
		Title:      "You were mentioned",
	})

	if push.userID != 0 {
		t.Fatalf("sent push to user %d after preference lookup failed", push.userID)
	}
	if !strings.Contains(logs.String(), "error fetching notification preferences") {
		t.Fatalf("missing preference lookup error log: %s", logs.String())
	}
}
