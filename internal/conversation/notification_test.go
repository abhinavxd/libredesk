package conversation

import (
	"errors"
	htmltemplate "html/template"
	"slices"
	"sync"
	"testing"
	"time"

	authzmodels "github.com/abhinavxd/libredesk/internal/authz/models"
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	notifier "github.com/abhinavxd/libredesk/internal/notification"
	nchannels "github.com/abhinavxd/libredesk/internal/notification/channels"
	nmodels "github.com/abhinavxd/libredesk/internal/notification/models"
	"github.com/abhinavxd/libredesk/internal/template"
	"github.com/abhinavxd/libredesk/internal/testutil"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/jmoiron/sqlx"
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
	err        error
	mu         sync.Mutex
}

type countingPush struct {
	count int
}

func (p *countingPush) Send(int, nmodels.PushPayload) bool {
	p.count++
	return true
}

func (s replyUserStore) Get(int, string, []string) (umodels.User, error) {
	return umodels.User{FirstName: "Contact"}, nil
}

func (s replyUserStore) GetAgentCachedOrLoad(int) (umodels.User, error) { return s.agent, s.err }

func (p *replyPreferences) EnabledChannels(ids []int, _ nmodels.NotificationType) (map[int][]nmodels.NotificationChannel, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.recipients = append(p.recipients, ids...)
	if p.err != nil {
		return nil, p.err
	}
	enabled := make(map[int][]nmodels.NotificationChannel)
	if len(p.channels) > 0 {
		for _, id := range ids {
			enabled[id] = slices.Clone(p.channels)
		}
	}
	return enabled, nil
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
			m := &Manager{
				lo: &lo,
				userStore: replyUserStore{
					agent: umodels.User{ID: agentID, Enabled: tt.enabled, Permissions: tt.permissions},
					err:   tt.err,
				},
			}
			m.q.GetConversationParticipantAgents = q.Participants
			conv.AssignedUserID = null.IntFrom(agentID + 1)
			recipients := m.replyNotificationParticipants(conv, agentID+1)
			if got := slices.ContainsFunc(recipients, func(recipient umodels.User) bool { return recipient.ID == agentID }); got != tt.want {
				t.Fatalf("participant eligible = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNotifyNewReplyChecksAssigneeAccess(t *testing.T) {
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
		t.Run(tt.name, func(t *testing.T) {
			m := &Manager{lo: &lo, userStore: replyUserStore{agent: umodels.User{ID: 42, Enabled: tt.enabled, Permissions: tt.permissions}}}
			recipients := m.replyNotificationAssignee(models.Conversation{AssignedUserID: null.IntFrom(42)}, 7)
			if got := slices.ContainsFunc(recipients, func(recipient umodels.User) bool { return recipient.ID == 42 }); got != tt.want {
				t.Fatalf("assignee eligible = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReplyNotificationsAlertForEveryMessage(t *testing.T) {
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

	newManager := func(channels ...nmodels.NotificationChannel) *Manager {
		db.MustExec(`DELETE FROM notification_email_queue`)
		db.MustExec(`DELETE FROM user_notifications`)
		m := &Manager{lo: &lo, i18n: i18n, template: templates,
			userStore: replyUserStore{agent: umodels.User{ID: userID, Email: null.StringFrom("history@example.com"), Enabled: true, Permissions: []string{authzmodels.PermConversationsRead, authzmodels.PermConversationsReadAssigned}}},
			dispatcher: notifier.NewDispatcher(notifier.DispatcherOpts{
				Pipeline: nchannels.NewPipeline(nchannels.NewInApp(inApp, nil, &lo), nchannels.NewEmail(emailQueue)),
				Prefs:    &replyPreferences{channels: channels},
			}),
		}
		m.q.GetConversationParticipantAgents = q.Participants
		return m
	}
	notifications := func() int {
		var count int
		db.Get(&count, `SELECT count(*) FROM user_notifications`)
		return count
	}
	message := models.Message{ID: messageID, SenderID: userID + 1, CreatedAt: time.Now().UTC().Truncate(time.Microsecond)}

	t.Run("each reply creates an in-app alert", func(t *testing.T) {
		m := newManager(nmodels.NotificationChannelInApp)

		m.NotifyNewReply(conv, message, false)
		if notifications() != 1 {
			t.Fatalf("notifications after first reply = %d, want 1", notifications())
		}

		next := message
		next.CreatedAt = message.CreatedAt.Add(time.Second)
		m.NotifyNewReply(conv, next, false)
		if notifications() != 2 {
			t.Fatalf("notifications after second reply = %d, want 2", notifications())
		}

		m.NotifyNewReply(conv, next, true)
		if notifications() != 3 {
			t.Fatalf("notifications after reopened reply = %d, want 3", notifications())
		}
	})

	t.Run("replies arriving together alert individually", func(t *testing.T) {
		m := newManager(nmodels.NotificationChannelInApp)
		var wg sync.WaitGroup
		for range 8 {
			wg.Go(func() { m.NotifyNewReply(conv, message, false) })
		}
		wg.Wait()

		if notifications() != 8 {
			t.Fatalf("notifications from concurrent replies = %d, want 8", notifications())
		}
	})

	t.Run("push alerts for every reply", func(t *testing.T) {
		m := newManager(nmodels.NotificationChannelPush)
		push := &countingPush{}
		m.dispatcher = notifier.NewDispatcher(notifier.DispatcherOpts{
			Pipeline: nchannels.NewPipeline(nchannels.NewInApp(inApp, nil, &lo), nchannels.NewEmail(emailQueue), nchannels.NewPush(push)),
			Prefs:    &replyPreferences{channels: []nmodels.NotificationChannel{nmodels.NotificationChannelPush}},
		})
		m.NotifyNewReply(conv, message, false)
		m.NotifyNewReply(conv, message, false)
		if push.count != 2 {
			t.Fatalf("push notifications = %d, want 2", push.count)
		}
	})

	t.Run("agent with no channel enabled is not notified", func(t *testing.T) {
		m := newManager()
		m.NotifyNewReply(conv, message, false)
		if notifications() != 0 {
			t.Fatalf("notifications = %d, want 0", notifications())
		}
	})

	t.Run("preference failure sends nothing", func(t *testing.T) {
		m := newManager()
		m.dispatcher = notifier.NewDispatcher(notifier.DispatcherOpts{Prefs: &replyPreferences{err: errors.New("lookup failed")}})
		m.NotifyNewReply(conv, message, false)
		if notifications() != 0 {
			t.Fatalf("notifications = %d, want 0", notifications())
		}
	})

	t.Run("email replies are clubbed in the queue", func(t *testing.T) {
		m := newManager(nmodels.NotificationChannelEmail)
		m.NotifyNewReply(conv, message, false)
		next := message
		next.CreatedAt = message.CreatedAt.Add(time.Minute)
		m.NotifyNewReply(conv, next, false)

		var queued int
		db.Get(&queued, `SELECT count(*) FROM notification_email_queue`)
		if queued != 1 {
			t.Fatalf("queued emails = %d, want 1", queued)
		}
	})
}
