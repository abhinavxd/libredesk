package notifier

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/notification/models"
	"github.com/volatiletech/null/v9"
)

type fakePreferences struct {
	channels map[int][]models.NotificationChannel
}

func (p fakePreferences) EnabledChannels([]int, models.NotificationType) map[int][]models.NotificationChannel {
	return p.channels
}

type fakePushDispatcher struct {
	userID  int
	payload PushPayload
}

func (p *fakePushDispatcher) Send(userID int, payload PushPayload) {
	p.userID = userID
	p.payload = payload
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
