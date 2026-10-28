package conversation

import (
	"encoding/json"
	"fmt"
	"testing"

	am "github.com/abhinavxd/libredesk/internal/authz/models"
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/csat"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	tmodels "github.com/abhinavxd/libredesk/internal/team/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/abhinavxd/libredesk/internal/ws"
	wsmodels "github.com/abhinavxd/libredesk/internal/ws/models"
	"github.com/zerodha/logf"
)

func TestCSATSubmissionBroadcastsSavedFeedback(t *testing.T) {
	db := testutil.NewDB(t, "csat_live_update")
	lo := logf.New(logf.Opts{})
	lang := testutil.NewI18n(t)
	csats, err := csat.New(csat.Opts{DB: db, Lo: &lo, I18n: lang})
	if err != nil {
		t.Fatal(err)
	}
	hub := ws.NewHub(&lo, nil /** userStore **/)
	m := &Manager{lo: &lo, i18n: lang, csatStore: csats, wsHub: hub}
	if err := dbutil.ScanSQLFile("queries.sql", &m.q, db, efs); err != nil {
		t.Fatal(err)
	}
	var contactID, inboxID int
	if err := db.Get(&contactID, `INSERT INTO users (type, first_name, last_name) VALUES ('contact', 'Customer', '') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&inboxID, `INSERT INTO inboxes (name, channel) VALUES ('Chat', 'livechat') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	for _, rating := range []int{0, 5} {
		t.Run(fmt.Sprint(rating), func(t *testing.T) {
			var conv models.Conversation
			if err := db.Get(&conv, `INSERT INTO conversations (contact_id, inbox_id, status_id)
				VALUES ($1, $2, (SELECT id FROM conversation_statuses LIMIT 1)) RETURNING id, uuid`, contactID, inboxID); err != nil {
				t.Fatal(err)
			}
			response, err := csats.Create(conv.ID)
			if err != nil {
				t.Fatal(err)
			}
			var messageUUID string
			if err := db.Get(&messageUUID, `INSERT INTO conversation_messages (conversation_id, type, status, sender_id, sender_type, meta)
				VALUES ($1, 'outgoing', 'sent', $2, 'agent', jsonb_build_object('is_csat', true, 'csat_uuid', $3::text)) RETURNING uuid`, conv.ID, contactID, response.UUID); err != nil {
				t.Fatal(err)
			}
			client := &ws.Client{Send: make(chan wsmodels.WSMessage, 2)}
			hub.SubscribeOpenConv(client, conv.UUID)
			feedback := "Helpful support. Café 日本語."
			if err := m.UpdateCSATResponse(response.UUID, rating, feedback, nil /** meta **/); err != nil {
				t.Fatal(err)
			}
			select {
			case update := <-client.Send:
				var payload struct {
					Type string `json:"type"`
					Data struct {
						UUID             string         `json:"uuid"`
						ConversationUUID string         `json:"conversation_uuid"`
						Meta             map[string]any `json:"meta"`
					} `json:"data"`
				}
				if err := json.Unmarshal(update.Data, &payload); err != nil {
					t.Fatal(err)
				}
				if payload.Type != wsmodels.MessageTypeMessageUpdate || payload.Data.UUID != messageUUID || payload.Data.ConversationUUID != conv.UUID {
					t.Fatalf("wrong message update: %+v", payload)
				}
				meta := payload.Data.Meta
				if meta["csat_submitted"] != true || meta["submitted_feedback"] != feedback || meta["csat_uuid"] != nil {
					t.Fatalf("incorrect or exposed CSAT metadata: %+v", meta)
				}
				if rating > 0 && meta["submitted_rating"] != float64(rating) {
					t.Fatalf("wrong rating: %+v", meta)
				}
			default:
				t.Fatal("no live CSAT update")
			}
			if err := m.UpdateCSATResponse(response.UUID, 1 /** rating **/, "duplicate", nil /** meta **/); err == nil {
				t.Fatal("duplicate response accepted")
			}
			if len(client.Send) != 0 {
				t.Fatal("rejected response broadcast an update")
			}
		})
	}
}

func TestPreviousConversationsRespectAccessBeforeLimit(t *testing.T) {
	db := testutil.NewDB(t, "previous_access")
	lo := logf.New(logf.Opts{})
	m := &Manager{lo: &lo, i18n: testutil.NewI18n(t)}
	if err := dbutil.ScanSQLFile("queries.sql", &m.q, db, efs); err != nil {
		t.Fatal(err)
	}
	var contact, actor, other, team, inbox int
	for i, id := range []*int{&contact, &actor, &other} {
		if err := db.Get(id, `INSERT INTO users (type, first_name, last_name) VALUES ('agent', $1, '') RETURNING id`, fmt.Sprintf("User %d", i)); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Get(&team, `INSERT INTO teams (name, conversation_assignment_type) VALUES ('Team', 'Manual') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&inbox, `INSERT INTO inboxes (name, channel) VALUES ('Inbox', 'email') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	for i, assignment := range [][2]int{{actor, 0}, {other, team}, {0, team}, {0, 0}, {other, 0}} {
		_, err := db.Exec(`INSERT INTO conversations (contact_id, inbox_id, status_id, subject, assigned_user_id, assigned_team_id, created_at)
		VALUES ($1, $2, (SELECT id FROM conversation_statuses LIMIT 1), $3, NULLIF($4, 0), NULLIF($5, 0), NOW() + $6 * interval '1 second')`, contact, inbox, fmt.Sprint(i), assignment[0], assignment[1], i)
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, tt := range []struct {
		permission string
		want       string
	}{
		{am.PermConversationsReadAssigned, "0"},
		{am.PermConversationsReadTeamAll, "2"},
		{am.PermConversationsReadTeamInbox, "2"},
		{am.PermConversationsReadUnassigned, "3"},
		{am.PermConversationsReadAll, "4"},
		{"", ""},
	} {
		t.Run(tt.permission, func(t *testing.T) {
			u := umodels.User{ID: actor, Enabled: true, Permissions: []string{am.PermConversationsRead, tt.permission}, Teams: tmodels.TeamsCompact{{ID: team}}}
			got, err := m.GetContactPreviousConversations(contact, 1, u)
			if err != nil {
				t.Fatal(err)
			}
			if tt.want == "" {
				if len(got) != 0 {
					t.Fatal("rows returned without access")
				}
			} else if len(got) != 1 || got[0].Subject != tt.want {
				t.Fatalf("unexpected visible rows: %+v", got)
			}
			u.Permissions = []string{tt.permission}
			got, err = m.GetContactPreviousConversations(contact, 1, u)
			if err != nil || len(got) != 0 {
				t.Fatalf("missing base permission: rows=%d err=%v", len(got), err)
			}
		})
	}
}
