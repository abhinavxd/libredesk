package notifier

import (
	"bytes"
	"errors"
	"strings"
	"testing"

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
	payload PushPayload
}

func (p *fakePushDispatcher) Send(userID int, payload PushPayload) bool {
	p.userID = userID
	p.payload = payload
	return true
}

func TestDispatcherSendsPushUsingNotificationRoute(t *testing.T) {
	push := &fakePushDispatcher{}
	d := &Dispatcher{
		prefs: fakePreferences{channels: map[int][]models.NotificationChannel{
			42: {models.NotificationChannelPush},
		}},
		push: push,
	}
	d.Send(Notification{
		Type:             models.NotificationTypeMention,
		RecipientIDs:     []int{42},
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
	d := &Dispatcher{
		prefs: fakePreferences{channels: map[int][]models.NotificationChannel{
			42: nil,
		}},
		push: push,
	}
	d.Send(Notification{
		Type:         models.NotificationTypeMention,
		RecipientIDs: []int{42},
		Title:        "You were mentioned",
	})

	if push.userID != 0 {
		t.Fatalf("sent push to user %d when push is disabled", push.userID)
	}
}

func TestDispatcherHandlesPreferenceLookupFailure(t *testing.T) {
	var logs bytes.Buffer
	lo := logf.New(logf.Opts{Writer: &logs})
	push := &fakePushDispatcher{}
	d := &Dispatcher{
		prefs: fakePreferences{err: errors.New("lookup failed")},
		push:  push,
		lo:    &lo,
	}
	d.Send(Notification{
		Type:         models.NotificationTypeMention,
		RecipientIDs: []int{42},
		Title:        "You were mentioned",
	})

	if push.userID != 0 {
		t.Fatalf("sent push to user %d after preference lookup failed", push.userID)
	}
	if !strings.Contains(logs.String(), "error fetching notification preferences") {
		t.Fatalf("missing preference lookup error log: %s", logs.String())
	}
}
