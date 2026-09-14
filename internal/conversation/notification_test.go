package conversation

import (
	"errors"
	"fmt"
	htmltemplate "html/template"
	"slices"
	"sync"
	"testing"
	"time"

	authzmodels "github.com/abhinavxd/libredesk/internal/authz/models"
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	notifier "github.com/abhinavxd/libredesk/internal/notification"
	nmodels "github.com/abhinavxd/libredesk/internal/notification/models"
	"github.com/abhinavxd/libredesk/internal/template"
	"github.com/abhinavxd/libredesk/internal/testutil"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/alicebob/miniredis/v2"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

type replyUserStore struct {
	userStore
	agent umodels.User
	err   error
}

type replyPreferences struct {
	recipients []int
	channels   []nmodels.NotificationChannel
}

type replyPush struct {
	reject bool
	count  int
}

func (p *replyPush) Send(int, notifier.PushPayload) bool {
	if p.reject {
		return false
	}
	p.count++
	return true
}

func (s replyUserStore) Get(int, string, []string) (umodels.User, error) {
	return umodels.User{FirstName: "Contact"}, nil
}

func (s replyUserStore) GetAgent(int, string) (umodels.User, error) { return s.agent, s.err }

func newReplyRedis(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	return redis.NewClient(&redis.Options{Addr: mr.Addr()}), mr
}

func (p *replyPreferences) EnabledChannels(ids []int, _ nmodels.NotificationType) map[int][]nmodels.NotificationChannel {
	p.recipients = append(p.recipients, ids...)
	enabled := make(map[int][]nmodels.NotificationChannel)
	if len(p.channels) > 0 {
		for _, id := range ids {
			enabled[id] = slices.Clone(p.channels)
		}
	}
	return enabled
}

func TestNotifyNewReplyChecksParticipantAccess(t *testing.T) {
	db := testutil.NewDB(t, "reply_access")
	var agentID, inboxID int
	if err := db.Get(&agentID, `INSERT INTO users (type, email, first_name, last_name) VALUES ('agent', 'participant@example.com', 'Agent', '') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&inboxID, `INSERT INTO inboxes (name, channel) VALUES ('Test', 'email') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	var conv models.Conversation
	if err := db.Get(&conv, `INSERT INTO conversations (contact_id, inbox_id, status_id) VALUES ($1, $2, (SELECT id FROM conversation_statuses LIMIT 1)) RETURNING id, uuid`, agentID, inboxID); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1, $2)`, conv.ID, agentID)
	var q struct {
		Participants *sqlx.Stmt `query:"get-conversation-participant-agents"`
	}
	if err := dbutil.ScanSQLFile("queries.sql", &q, db, efs); err != nil {
		t.Fatal(err)
	}
	lo := logf.New(logf.Opts{})
	for _, tt := range []struct {
		name        string
		permissions []string
		enabled     bool
		err         error
		want        bool
	}{
		{"former assignee", []string{authzmodels.PermConversationsRead, authzmodels.PermConversationsReadAssigned}, true, nil, false},
		{"read all", []string{authzmodels.PermConversationsRead, authzmodels.PermConversationsReadAll}, true, nil, true},
		{"missing base permission", []string{authzmodels.PermConversationsReadAll}, true, nil, false},
		{"disabled", []string{authzmodels.PermConversationsRead, authzmodels.PermConversationsReadAll}, false, nil, false},
		{"load failed", nil, true, errors.New("unavailable"), false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			prefs := &replyPreferences{}
			rdb, _ := newReplyRedis(t)
			m := &Manager{
				lo:   &lo,
				rdb:  rdb,
				i18n: testutil.NewI18n(t),
				userStore: replyUserStore{
					agent: umodels.User{ID: agentID, Enabled: tt.enabled, Permissions: tt.permissions},
					err:   tt.err,
				},
				dispatcher: notifier.NewDispatcher(notifier.DispatcherOpts{Prefs: prefs}),
			}
			m.q.GetConversationParticipantAgents = q.Participants
			conv.AssignedUserID = null.IntFrom(agentID + 1)
			m.NotifyNewReply(conv, models.Message{SenderID: agentID + 1}, false)
			if got := slices.Contains(prefs.recipients, agentID); got != tt.want {
				t.Fatalf("participant eligible = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNotifyNewReplyChecksAssigneeAccess(t *testing.T) {
	db := testutil.NewDB(t, "reply_assignee_access")
	var q struct {
		Participants *sqlx.Stmt `query:"get-conversation-participant-agents"`
	}
	if err := dbutil.ScanSQLFile("queries.sql", &q, db, efs); err != nil {
		t.Fatal(err)
	}
	lo := logf.New(logf.Opts{})
	for _, tt := range []struct {
		name        string
		enabled     bool
		permissions []string
		want        bool
	}{
		{"assigned access", true, []string{authzmodels.PermConversationsRead, authzmodels.PermConversationsReadAssigned}, true},
		{"disabled", false, []string{authzmodels.PermConversationsRead, authzmodels.PermConversationsReadAssigned}, false},
		{"missing base permission", true, []string{authzmodels.PermConversationsReadAssigned}, false},
		{"missing assignment access", true, []string{authzmodels.PermConversationsRead}, false},
	} {
		for _, reopened := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/reopened=%v", tt.name, reopened), func(t *testing.T) {
				prefs := &replyPreferences{}
				rdb, _ := newReplyRedis(t)
				m := &Manager{lo: &lo, rdb: rdb, i18n: testutil.NewI18n(t), userStore: replyUserStore{agent: umodels.User{ID: 42, Enabled: tt.enabled, Permissions: tt.permissions}}, dispatcher: notifier.NewDispatcher(notifier.DispatcherOpts{Prefs: prefs})}
				m.q.GetConversationParticipantAgents = q.Participants
				m.NotifyNewReply(models.Conversation{UUID: "00000000-0000-0000-0000-000000000000", AssignedUserID: null.IntFrom(42)}, models.Message{SenderID: 7}, reopened)
				if got := slices.Contains(prefs.recipients, 42); got != tt.want {
					t.Fatalf("assignee eligible = %v, want %v", got, tt.want)
				}
			})
		}
	}
}

func TestReplyNotificationSuppressedUntilConversationRead(t *testing.T) {
	db := testutil.NewDB(t, "reply_notification_history")
	var userID, inboxID int
	if err := db.Get(&userID, `INSERT INTO users (type, email, first_name, last_name) VALUES ('agent', 'history@example.com', 'Agent', '') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&inboxID, `INSERT INTO inboxes (name, channel) VALUES ('Test', 'email') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	var conv models.Conversation
	if err := db.Get(&conv, `INSERT INTO conversations (contact_id, inbox_id, status_id) VALUES ($1, $2, (SELECT id FROM conversation_statuses LIMIT 1)) RETURNING id, uuid`, userID, inboxID); err != nil {
		t.Fatal(err)
	}
	var messageID int
	if err := db.Get(&messageID, `INSERT INTO conversation_messages (conversation_id, sender_id, sender_type, type, status, content, text_content, created_at) VALUES ($1, $2, 'contact', 'incoming', 'received', 'Old reply', 'Old reply', $3) RETURNING id`, conv.ID, userID, time.Now().Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}

	var q struct {
		Participants *sqlx.Stmt `query:"get-conversation-participant-agents"`
		LastSeen     *sqlx.Stmt `query:"upsert-user-last-seen"`
	}
	if err := dbutil.ScanSQLFile("queries.sql", &q, db, efs); err != nil {
		t.Fatal(err)
	}
	lo := logf.New(logf.Opts{})
	i18n := testutil.NewI18n(t)
	inApp, err := notifier.NewUserNotificationManager(notifier.UserNotificationOpts{DB: db, Lo: &lo, I18n: i18n})
	if err != nil {
		t.Fatal(err)
	}
	emailQueue, err := notifier.NewEmailQueue(notifier.EmailQueueOpts{DB: db, Lo: &lo})
	if err != nil {
		t.Fatal(err)
	}
	templates, err := template.New(&lo, db, nil, nil, htmltemplate.FuncMap{"RootURL": func() string { return "http://localhost" }}, i18n)
	if err != nil {
		t.Fatal(err)
	}
	conv.AssignedUserID = null.IntFrom(userID)
	for _, channel := range []nmodels.NotificationChannel{nmodels.NotificationChannelInApp, nmodels.NotificationChannelEmail, nmodels.NotificationChannelPush} {
		t.Run(string(channel), func(t *testing.T) {
			db.MustExec(`DELETE FROM conversation_last_seen`)
			db.MustExec(`DELETE FROM notification_email_queue`)
			db.MustExec(`DELETE FROM user_notifications`)
			prefs := &replyPreferences{}
			push := &replyPush{}
			rdb, mr := newReplyRedis(t)
			m := &Manager{lo: &lo, rdb: rdb, i18n: i18n, template: templates,
				userStore:  replyUserStore{agent: umodels.User{ID: userID, Email: null.StringFrom("history@example.com"), Enabled: true, Permissions: []string{authzmodels.PermConversationsRead, authzmodels.PermConversationsReadAssigned}}},
				dispatcher: notifier.NewDispatcher(notifier.DispatcherOpts{Lo: &lo, Prefs: prefs, Push: push, EmailQueue: emailQueue, EmailEnabled: true, InApp: inApp}),
			}
			m.q.GetConversationParticipantAgents = q.Participants
			m.q.UpsertUserLastSeen = q.LastSeen

			delivered := func() bool {
				var count int
				switch channel {
				case nmodels.NotificationChannelInApp:
					db.Get(&count, `SELECT count(*) FROM user_notifications`)
					db.MustExec(`DELETE FROM user_notifications`)
				case nmodels.NotificationChannelEmail:
					db.Get(&count, `SELECT count(*) FROM notification_email_queue`)
					db.MustExec(`DELETE FROM notification_email_queue`)
				case nmodels.NotificationChannelPush:
					count, push.count = push.count, 0
				}
				return count > 0
			}
			claimed := func() bool {
				return mr.Exists(replyNotificationKey(conv.UUID, userID))
			}

			message := models.Message{ID: messageID, SenderID: userID + 1, CreatedAt: time.Now().UTC().Truncate(time.Microsecond)}
			m.NotifyNewReply(conv, message, false)
			if claimed() {
				t.Fatal("an agent with every channel disabled was marked notified")
			}

			prefs.channels = []nmodels.NotificationChannel{channel}
			m.NotifyNewReply(conv, message, false)
			if !delivered() || !claimed() {
				t.Fatal("first reply was not notified")
			}

			next := message
			next.CreatedAt = message.CreatedAt.Add(time.Second)
			m.NotifyNewReply(conv, next, false)
			if delivered() {
				t.Fatal("second unread reply sent another notification")
			}

			if err := m.UpdateUserLastSeen(conv.UUID, userID); err != nil {
				t.Fatal(err)
			}
			if claimed() {
				t.Fatal("reading the conversation did not clear the notification")
			}
			m.NotifyNewReply(conv, next, false)
			if !delivered() {
				t.Fatal("reading the conversation did not allow the next notification")
			}

			m.NotifyNewReply(conv, next, true)
			if !delivered() {
				t.Fatal("reopened conversation alert was suppressed")
			}

			if channel != nmodels.NotificationChannelPush {
				return
			}
			mr.FlushAll()
			push.reject = true
			m.NotifyNewReply(conv, next, false)
			if claimed() {
				t.Fatal("failed push enqueue suppressed later replies")
			}
			push.reject = false
			push.count = 0
			var wg sync.WaitGroup
			for range 8 {
				wg.Go(func() { m.NotifyNewReply(conv, next, false) })
			}
			wg.Wait()
			if push.count != 1 {
				t.Fatalf("concurrent replies queued %d pushes, want 1", push.count)
			}
		})
	}
}
