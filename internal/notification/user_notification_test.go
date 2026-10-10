package notifier

import (
	"fmt"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/notification/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/jmoiron/sqlx"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

type replyAgentStore struct{ agent umodels.User }

type replyFixture struct {
	db         *sqlx.DB
	manager    *UserNotificationManager
	checker    *ReplyDeliveryChecker
	agents     *replyAgentStore
	userID     int
	convID     int
	messageIDs []int
}

func newReplyFixture(t *testing.T, name string) replyFixture {
	t.Helper()
	f := replyFixture{db: testutil.NewDB(t, name)}
	if err := f.db.Get(&f.userID, `INSERT INTO users (type, email, first_name, last_name) VALUES ('agent', 'agent@example.com', 'Agent', '') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	var inboxID int
	if err := f.db.Get(&inboxID, `INSERT INTO inboxes (name, channel) VALUES ('Test', 'email') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := f.db.Get(&f.convID, `INSERT INTO conversations (contact_id, inbox_id, status_id, assigned_user_id) VALUES ($1, $2, (SELECT id FROM conversation_statuses LIMIT 1), $1) RETURNING id`, f.userID, inboxID); err != nil {
		t.Fatal(err)
	}
	for i := range 2 {
		var id int
		if err := f.db.Get(&id, `INSERT INTO conversation_messages (conversation_id, sender_id, sender_type, type, status, created_at) VALUES ($1, $2, 'contact', 'incoming', 'received', $3) RETURNING id`, f.convID, f.userID, time.Date(2026, 1, 1, 0, 0, i, 0, time.UTC)); err != nil {
			t.Fatal(err)
		}
		f.messageIDs = append(f.messageIDs, id)
	}
	lo := logf.New(logf.Opts{})
	var err error
	f.manager, err = NewUserNotificationManager(UserNotificationOpts{DB: f.db, Lo: &lo, I18n: testutil.NewI18n(t)})
	if err != nil {
		t.Fatal(fmt.Errorf("notification manager: %w", err))
	}
	f.agents = &replyAgentStore{agent: umodels.User{ID: f.userID, Enabled: true, Permissions: []string{"conversations:read", "conversations:read_all"}}}
	f.checker = NewReplyDeliveryChecker(f.manager, f.agents, fakePreferences{channels: map[int][]models.NotificationChannel{f.userID: {models.NotificationChannelEmail, models.NotificationChannelPush}}})
	return f
}

func (f replyFixture) markSeen(t *testing.T, messageIndex int) {
	t.Helper()
	f.db.MustExec(`INSERT INTO conversation_last_seen (user_id, conversation_id, last_seen_at)
		SELECT $1, conversation_id, created_at FROM conversation_messages WHERE id = $2
		ON CONFLICT (conversation_id, user_id) DO UPDATE SET last_seen_at = EXCLUDED.last_seen_at`, f.userID, f.messageIDs[messageIndex])
}

func (s *replyAgentStore) GetAgentCachedOrLoad(int) (umodels.User, error) { return s.agent, nil }

func TestReplyReadFollowsLastSeen(t *testing.T) {
	f := newReplyFixture(t, "reply_read_last_seen")
	create := func(typ models.NotificationType, messageID int) {
		t.Helper()
		if _, err := f.manager.Create(f.userID, typ, "Reply", null.String{}, null.IntFrom(f.convID), null.IntFrom(messageID), null.Int{}, nil); err != nil {
			t.Fatal(err)
		}
	}
	for _, typ := range models.ReplyTypes {
		create(typ, f.messageIDs[0])
	}
	create(models.NotificationTypeNewReply, f.messageIDs[1])
	create(models.NotificationTypeMention, f.messageIDs[0])
	create(models.NotificationTypeAssignment, f.messageIDs[0])
	f.markSeen(t, 0)
	late, err := f.manager.Create(f.userID, models.NotificationTypeNewReply, "Reply", null.String{}, null.IntFrom(f.convID), null.IntFrom(f.messageIDs[0]), null.Int{}, nil)
	if err != nil || !late.IsRead {
		t.Fatalf("reply created after it was read: %+v err=%v", late, err)
	}
	stats, err := f.manager.GetStats(f.userID)
	if err != nil || stats.UnreadCount != 3 || stats.TotalCount != 7 {
		t.Fatalf("stats=%+v err=%v", stats, err)
	}
	notifications, err := f.manager.GetAll(f.userID, 30, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range notifications {
		want := models.IsReply(n.NotificationType) && n.MessageID.Int == f.messageIDs[0]
		if n.IsRead != want {
			t.Fatalf("notification=%+v want read=%v", n, want)
		}
	}
}

func TestReplyDeliveryRechecksEligibility(t *testing.T) {
	f := newReplyFixture(t, "reply_delivery_eligibility")
	ref := models.NotificationReference{Type: models.NotificationTypeNewReply, ConversationID: null.IntFrom(f.convID), MessageID: null.IntFrom(f.messageIDs[1])}
	check := func(want bool) {
		t.Helper()
		for _, channel := range []models.NotificationChannel{models.NotificationChannelEmail, models.NotificationChannelPush} {
			got, err := f.checker.ShouldDeliver(f.userID, ref, channel)
			if err != nil || got != want {
				t.Fatalf("channel=%s deliver=%v want=%v err=%v", channel, got, want, err)
			}
		}
	}
	check(true)
	f.agents.agent.Enabled = false
	check(false)
	f.agents.agent.Enabled = true
	f.agents.agent.Permissions = nil
	check(false)
	f.agents.agent.Permissions = []string{"conversations:read", "conversations:read_all"}
	f.db.MustExec(`UPDATE conversations SET assigned_user_id = NULL WHERE id = $1`, f.convID)
	check(false)
	f.db.MustExec(`INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1, $2)`, f.convID, f.userID)
	ref.Type = models.NotificationTypeNewReplyParticipating
	check(true)
	f.checker.prefs = fakePreferences{}
	check(false)
	f.checker.prefs = fakePreferences{channels: map[int][]models.NotificationChannel{f.userID: {models.NotificationChannelEmail, models.NotificationChannelPush}}}
	f.markSeen(t, 1)
	check(false)
	ref.MessageID = null.Int{}
	check(false)
	if err := f.db.Close(); err != nil {
		t.Fatal(err)
	}
	ref.MessageID = null.IntFrom(f.messageIDs[1])
	if _, err := f.checker.ShouldDeliver(f.userID, ref, models.NotificationChannelEmail); err == nil {
		t.Fatal("database error was ignored")
	}
}
