package conversation

import (
	"fmt"
	"testing"

	am "github.com/abhinavxd/libredesk/internal/authz/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	tmodels "github.com/abhinavxd/libredesk/internal/team/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/jmoiron/sqlx"
	"github.com/zerodha/logf"
)

func TestPreviousConversationsRespectAccessBeforeLimit(t *testing.T) {
	db := testutil.NewDB(t, "previous_access")
	var q struct {
		Previous *sqlx.Stmt `query:"get-contact-previous-conversations"`
	}
	if err := dbutil.ScanSQLFile("queries.sql", &q, db, efs); err != nil {
		t.Fatal(err)
	}
	lo := logf.New(logf.Opts{})
	m := &Manager{lo: &lo, i18n: testutil.NewI18n(t)}
	m.q.GetContactPreviousConversations = q.Previous
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
