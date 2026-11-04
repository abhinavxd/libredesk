package conversation

import (
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"fmt"
	"slices"
	"testing"
	"time"

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

func TestConversationListPaginationBeforeDetails(t *testing.T) {
	db := testutil.NewDB(t, "conversation_list_pagination")
	lo := logf.New(logf.Opts{})
	m := &Manager{db: db, lo: &lo, i18n: testutil.NewI18n(t), settingsStore: stubSettingsStore{}}
	if err := dbutil.ScanSQLFile("queries.sql", &m.q, db, efs); err != nil {
		t.Fatal(err)
	}
	var contact, agent, other, inbox, team, policy int
	for i, id := range []*int{&contact, &agent, &other} {
		if err := db.Get(id, `INSERT INTO users (type, first_name, last_name, email) VALUES ($1, $2, '', $3) RETURNING id`,
			[]string{"contact", "agent", "agent"}[i], fmt.Sprint(i), fmt.Sprintf("user%d@example.com", i)); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Get(&inbox, `INSERT INTO inboxes (name, channel) VALUES ('Inbox', 'email') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&team, `INSERT INTO teams (name, conversation_assignment_type) VALUES ('Team', 'Manual') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO team_members (team_id, user_id) VALUES ($1, $2)`, team, agent); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&policy, `INSERT INTO sla_policies (name, first_response_time, resolution_time) VALUES ('SLA', '1h', '1d') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	deadline := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	latest := map[string]int{}
	for n := 1; n <= 72; n++ {
		status := "Closed"
		if n > 64 {
			status = "Open"
		}
		assigned := 0
		if n%4 == 0 {
			assigned = agent
		} else if n%4 == 1 {
			assigned = other
		}
		assignedTeam := 0
		if n%4 == 2 {
			assignedTeam = team
		}
		var conversationID, appliedID, messageID int
		if err := db.Get(&conversationID, `INSERT INTO conversations
			(contact_id, inbox_id, status_id, subject, assigned_user_id, assigned_team_id, last_message_at, created_at)
			VALUES ($1, $2, (SELECT id FROM conversation_statuses WHERE name = $3), $4, NULLIF($5, 0), NULLIF($6, 0),
			$7::timestamptz + $8 * interval '1 minute', $7::timestamptz + $8 * interval '1 minute') RETURNING id`,
			contact, inbox, status, fmt.Sprint(n), assigned, assignedTeam, deadline, n); err != nil {
			t.Fatal(err)
		}
		for _, history := range []bool{true, false} {
			if err := db.Get(&appliedID, `INSERT INTO applied_slas
				(conversation_id, sla_policy_id, status, created_at, first_response_deadline_at, resolution_deadline_at)
				VALUES ($1, $2, CASE WHEN $3 THEN 'met'::applied_sla_status ELSE 'pending'::applied_sla_status END,
				$4::timestamptz - CASE WHEN $3 THEN interval '1 day' ELSE interval '0 days' END, $4, $4) RETURNING id`,
				conversationID, policy, history, deadline); err != nil {
				t.Fatal(err)
			}
		}
		latest[fmt.Sprint(n)] = appliedID
		if _, err := db.Exec(`INSERT INTO sla_events (applied_sla_id, sla_policy_id, type, deadline_at) VALUES ($1, $2, 'next_response', $3)`, appliedID, policy, deadline); err != nil {
			t.Fatal(err)
		}
		if err := db.Get(&messageID, `INSERT INTO conversation_messages (conversation_id, type, status, sender_id, sender_type) VALUES ($1, 'incoming', 'sent', $2, 'contact') RETURNING id`, conversationID, contact); err != nil {
			t.Fatal(err)
		}
		if n%4 == 0 {
			if _, err := db.Exec(`INSERT INTO conversation_mentions (conversation_id, message_id, mentioned_user_id, mentioned_by_user_id) VALUES ($1, $2, $3, $4)`, conversationID, messageID, agent, other); err != nil {
				t.Fatal(err)
			}
		}
		if n == 67 || n == 68 {
			if _, err := db.Exec(`INSERT INTO conversation_mentions (conversation_id, message_id, mentioned_team_id, mentioned_by_user_id) VALUES ($1, $2, $3, $4)`, conversationID, messageID, team, other); err != nil {
				t.Fatal(err)
			}
		}
	}
	var tags [2]int
	for n := range tags {
		if err := db.Get(&tags[n], `INSERT INTO tags (name) VALUES ($1) RETURNING id`, fmt.Sprintf("Tag %d", n)); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO conversation_tags (conversation_id, tag_id)
			SELECT id, $1 FROM conversations WHERE subject::int % $2 = 0`, tags[n], n+2); err != nil {
			t.Fatal(err)
		}
	}
	tagFilter := fmt.Sprintf(`[{"model":"conversations","field":"tags","operator":"contains","value":"[%d,%d,%d]"}]`, tags[0], tags[0], tags[1])
	closedFilter := `[{"model":"conversation_statuses","field":"name","operator":"equals","value":"Closed"}]`
	openFilter := `[{"model":"conversation_statuses","field":"name","operator":"equals","value":"Open"}]`
	for _, tt := range []struct {
		name, listType, order, orderBy, filters string
		page, total                             int
		want                                    []string
	}{
		{"closed second page", models.AllConversations, "desc", "", closedFilter, 2, 64, []string{"62", "61"}},
		{"ascending", models.AllConversations, "asc", "conversations.created_at", closedFilter, 1, 64, []string{"1", "2"}},
		{"assigned", models.AssignedConversations, "desc", "", closedFilter, 2, 16, []string{"56", "52"}},
		{"assigned sparse open", models.AssignedConversations, "desc", "", openFilter, 1, 2, []string{"72", "68"}},
		{"assigned sparse ascending", models.AssignedConversations, "asc", "", openFilter, 1, 2, []string{"68", "72"}},
		{"assigned sparse past last page", models.AssignedConversations, "desc", "", openFilter, 2, 0, nil},
		{"assigned tag union", models.AssignedConversations, "desc", "", tagFilter, 1, 18, []string{"72", "68"}},
		{"assigned no filters", models.AssignedConversations, "desc", "", "", 1, 18, []string{"72", "68"}},
		{"assigned no matches", models.AssignedConversations, "desc", "", `[{"model":"users","field":"email","operator":"equals","value":"missing@example.com"}]`, 1, 0, nil},
		{"team", models.TeamUnassignedConversations, "desc", "", closedFilter, 1, 16, []string{"62", "58"}},
		{"unassigned", models.UnassignedConversations, "desc", "", closedFilter, 1, 16, []string{"63", "59"}},
		{"mentioned", models.MentionedConversations, "desc", "", closedFilter, 1, 16, []string{"64", "60"}},
		{"mentioned open direct and team", models.MentionedConversations, "desc", "", openFilter, 1, 3, []string{"72", "68"}},
		{"mentioned open second page", models.MentionedConversations, "desc", "", openFilter, 2, 3, []string{"67"}},
		{"mentioned open ascending", models.MentionedConversations, "asc", "", openFilter, 1, 3, []string{"67", "68"}},
		{"mentioned past last page", models.MentionedConversations, "desc", "", openFilter, 3, 0, nil},
		{"mentioned no filters", models.MentionedConversations, "desc", "", "", 1, 19, []string{"72", "68"}},
		{"mentioned no matches", models.MentionedConversations, "desc", "", `[{"model":"users","field":"email","operator":"equals","value":"missing@example.com"}]`, 1, 0, nil},
		{"date filter", models.AllConversations, "desc", "", `[{"model":"conversations","field":"created_at","operator":"equals","value":"2026-01-01"}]`, 1, 72, []string{"72", "71"}},
		{"contact filter", models.AllConversations, "desc", "", `[{"model":"users","field":"email","operator":"equals","value":"user0@example.com"}]`, 1, 72, []string{"72", "71"}},
		{"tag union", models.AllConversations, "desc", "", tagFilter, 1, 48, []string{"72", "70"}},
		{"no filters", models.AllConversations, "", "", "", 1, 72, []string{"72", "71"}},
		{"last page", models.AllConversations, "desc", "", closedFilter, 32, 64, []string{"2", "1"}},
		{"past last page", models.AllConversations, "desc", "", closedFilter, 33, 0, nil},
		{"no matches", models.AllConversations, "desc", "", `[{"model":"users","field":"email","operator":"equals","value":"missing@example.com"}]`, 1, 0, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rows, err := m.GetConversations(agent, agent, []int{team}, []string{tt.listType}, tt.order, tt.orderBy, tt.filters, tt.page, 2 /** pageSize **/)
			if err != nil {
				t.Fatal(err)
			}
			var subjects []string
			for _, row := range rows {
				subjects = append(subjects, row.Subject.String)
				if row.Total != tt.total || row.AppliedSLAID.Int != latest[row.Subject.String] || row.UnreadMessageCount != 1 || !row.FirstResponseDueAt.Time.Equal(deadline) || !row.NextResponseDueAt.Time.Equal(deadline) {
					t.Fatalf("incorrect total, unread count, or SLA details: %+v", row)
				}
				if (tt.listType == models.MentionedConversations) != row.MentionedMessageUUID.Valid {
					t.Fatalf("incorrect mentioned message: %+v", row.MentionedMessageUUID)
				}
			}
			if !slices.Equal(subjects, tt.want) {
				t.Fatalf("subjects = %v, want %v", subjects, tt.want)
			}
		})
	}

	if _, err := db.Exec(`UPDATE conversations SET last_message_at = NULL WHERE subject = '68'`); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		listType, order string
		want            []string
	}{
		{models.AssignedConversations, "asc", []string{"72", "68"}},
		{models.AssignedConversations, "desc", []string{"72", "68"}},
		{models.MentionedConversations, "asc", []string{"67", "72", "68"}},
		{models.MentionedConversations, "desc", []string{"72", "67", "68"}},
	} {
		t.Run(tt.listType+" null activity "+tt.order, func(t *testing.T) {
			rows, err := m.GetConversations(agent, agent, nil /** teamIDs **/, []string{tt.listType}, tt.order, "", openFilter, 1 /** page **/, 3 /** pageSize **/)
			if err != nil {
				t.Fatal(err)
			}
			var subjects []string
			for _, row := range rows {
				subjects = append(subjects, row.Subject.String)
				if row.Total != len(tt.want) {
					t.Fatalf("total = %d, want %d", row.Total, len(tt.want))
				}
			}
			if !slices.Equal(subjects, tt.want) || rows[len(rows)-1].LastMessageAt.Valid {
				t.Fatalf("incorrect null ordering: %v", subjects)
			}
		})
	}
	if _, err := db.Exec(`UPDATE conversations SET last_message_at = created_at WHERE subject = '68'`); err != nil {
		t.Fatal(err)
	}

	for _, orderBy := range []string{"users.email", "users.external_user_id", "inboxes.channel", "conversation_statuses.name", "conversations.next_sla_deadline_at"} {
		for _, listType := range []string{models.AllConversations, models.AssignedConversations, models.MentionedConversations} {
			t.Run(listType+" "+orderBy, func(t *testing.T) {
				rows, err := m.GetConversations(agent, agent, nil /** teamIDs **/, []string{listType}, "desc", orderBy, closedFilter, 1 /** page **/, 2 /** pageSize **/)
				wantTotal := 16
				if listType == models.AllConversations {
					wantTotal = 64
				}
				if err != nil || len(rows) != 2 || rows[0].Total != wantTotal {
					t.Fatalf("joined ordering: rows=%d err=%v", len(rows), err)
				}
			})
		}
	}
	for _, orderBy := range []string{"conversations.id; SELECT 1", "missing.column"} {
		if _, _, err := m.makeConversationsListQuery(agent, agent, nil /** teamIDs **/, []string{models.AllConversations}, m.q.GetConversations, "desc", orderBy, 1 /** page **/, 2 /** pageSize **/, closedFilter); err == nil {
			t.Fatalf("invalid ordering accepted: %s", orderBy)
		}
	}

	type planNode struct {
		Relation string     `json:"Relation Name"`
		Subplan  string     `json:"Subplan Name"`
		Loops    int        `json:"Actual Loops"`
		Rows     int        `json:"Actual Rows"`
		Plans    []planNode `json:"Plans"`
	}
	for _, listType := range []string{models.AllConversations, models.AssignedConversations, models.MentionedConversations, models.UnassignedConversations, models.TeamUnassignedConversations, models.TeamAllConversations} {
		t.Run(listType+" plan", func(t *testing.T) {
			query, args, err := m.makeConversationsListQuery(agent, agent, []int{team}, []string{listType}, m.q.GetConversations, "desc", "", 1 /** page **/, 2 /** pageSize **/, closedFilter)
			if err != nil {
				t.Fatal(err)
			}
			var raw []byte
			if err := db.Get(&raw, "EXPLAIN (ANALYZE, FORMAT JSON) "+query, args...); err != nil {
				t.Fatal(err)
			}
			var plan []struct{ Plan planNode }
			if err := jsonv2.Unmarshal(raw, &plan); err != nil {
				t.Fatal(err)
			}
			pending := []planNode{plan[0].Plan}
			seen := map[string]bool{}
			materialized := false
			for len(pending) > 0 {
				node := pending[0]
				pending = append(pending[1:], node.Plans...)
				if node.Subplan == "CTE matching" {
					materialized = true
					want := 16
					if listType == models.AllConversations {
						want = 64
					}
					if node.Rows != want || node.Loops != 1 {
						t.Fatalf("filtered %d rows %d times, want %d rows once", node.Rows, node.Loops, want)
					}
				}
				if node.Relation == "applied_slas" || node.Relation == "sla_events" {
					seen[node.Relation] = true
					if node.Loops > 2 {
						t.Fatalf("%s looked up %d times for a two ticket page", node.Relation, node.Loops)
					}
				}
			}
			if !materialized {
				t.Fatalf("incorrect filtering boundary for %s: %s", listType, raw)
			}
			if len(seen) != 2 {
				t.Fatalf("missing SLA lookups in plan: %s", raw)
			}
		})
	}
	if _, err := db.Exec(`DELETE FROM applied_slas WHERE conversation_id IN (SELECT id FROM conversations WHERE subject = '64')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE conversations SET last_message_at = NULL WHERE subject = '64'`); err != nil {
		t.Fatal(err)
	}
	rows, err := m.GetAllConversationsList(agent, "desc", "", closedFilter, 32 /** page **/, 2 /** pageSize **/)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].Subject.String != "1" || rows[1].Subject.String != "64" || rows[1].Total != 64 || rows[1].LastMessageAt.Valid || rows[1].AppliedSLAID.Valid || rows[1].NextResponseDueAt.Valid {
		t.Fatalf("incorrect null ordering or missing SLA details: %+v", rows)
	}
	if _, err := db.Exec(`INSERT INTO conversation_statuses (name) VALUES ('Waiting for approval');
		UPDATE conversations SET status_id = (SELECT id FROM conversation_statuses WHERE name = 'Waiting for approval')
		WHERE subject IN ('1', '2', '3', '4')`); err != nil {
		t.Fatal(err)
	}
	rareFilter := `[{"model":"conversation_statuses","field":"name","operator":"equals","value":"Waiting for approval"}]`
	for _, tt := range []struct {
		name  string
		lists []string
		want  []string
	}{
		{"all", []string{models.AllConversations}, []string{"4", "3", "2", "1"}},
		{"assigned", []string{models.AssignedConversations}, []string{"4"}},
		{"mentioned", []string{models.MentionedConversations}, []string{"4"}},
		{"unassigned", []string{models.UnassignedConversations}, []string{"3"}},
		{"team inbox", []string{models.TeamUnassignedConversations}, []string{"2"}},
		{"team all", []string{models.TeamAllConversations}, []string{"2"}},
		{"combined view", []string{models.AssignedConversations, models.UnassignedConversations, models.TeamUnassignedConversations}, []string{"4", "3", "2"}},
		{"overlapping view", []string{models.AssignedConversations, models.MentionedConversations}, []string{"4"}},
	} {
		t.Run("rare status "+tt.name, func(t *testing.T) {
			rows, err := m.GetViewConversationsList(agent, agent, []int{team}, tt.lists, "desc", "", rareFilter, 1 /** page **/, 25 /** pageSize **/)
			if err != nil {
				t.Fatal(err)
			}
			var subjects []string
			for _, row := range rows {
				subjects = append(subjects, row.Subject.String)
				if row.Total != len(tt.want) {
					t.Fatalf("total = %d, want %d", row.Total, len(tt.want))
				}
			}
			if !slices.Equal(subjects, tt.want) {
				t.Fatalf("subjects = %v, want %v", subjects, tt.want)
			}
		})
	}
	viewFilter := fmt.Sprintf(`{"logic":"AND","rules":[
		{"model":"conversation_statuses","field":"name","operator":"equals","value":"Waiting for approval"},
		{"logic":"OR","rules":[
			{"model":"conversations","field":"tags","operator":"contains","value":"[%d,%d]"},
			{"model":"users","field":"email","operator":"equals","value":"missing@example.com"}
		]},
		{"model":"inboxes","field":"channel","operator":"equals","value":"email"}
	]}`, tags[0], tags[0])
	rows, err = m.GetViewConversationsList(agent, agent, []int{team}, []string{models.AllConversations}, "desc", "", viewFilter, 1 /** page **/, 25 /** pageSize **/)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].Subject.String != "4" || rows[1].Subject.String != "2" || rows[0].Total != 2 {
		t.Fatalf("incorrect filtered view: %+v", rows)
	}
}

func TestAutoassignmentQuerySkipsManualTeams(t *testing.T) {
	db := testutil.NewDB(t, "autoassignment_query")
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, db, efs); err != nil {
		t.Fatal(err)
	}
	var contact, inbox int
	if err := db.Get(&contact, `INSERT INTO users (type, first_name) VALUES ('contact', 'Contact') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&inbox, `INSERT INTO inboxes (name, channel) VALUES ('Inbox', 'email') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	var want []string
	for _, kind := range []string{"Manual", "Round robin"} {
		var team int
		if err := db.Get(&team, `INSERT INTO teams (name, conversation_assignment_type) VALUES ($1::text, $1::text::conversation_assignment_type) RETURNING id`, kind); err != nil {
			t.Fatal(err)
		}
		var uuid string
		if err := db.Get(&uuid, `INSERT INTO conversations (contact_id, inbox_id, status_id, assigned_team_id) VALUES ($1, $2, (SELECT id FROM conversation_statuses LIMIT 1), $3) RETURNING uuid`, contact, inbox, team); err != nil {
			t.Fatal(err)
		}
		if kind == "Round robin" {
			want = append(want, uuid)
		}
		if _, err := db.Exec(`INSERT INTO conversations (contact_id, inbox_id, status_id, assigned_team_id, assigned_user_id) VALUES ($1, $2, (SELECT id FROM conversation_statuses LIMIT 1), $3, $1)`, contact, inbox, team); err != nil {
			t.Fatal(err)
		}
	}
	var rows []models.Conversation
	if err := q.GetUnassignedConversations.Select(&rows); err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, row := range rows {
		got = append(got, row.UUID)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("assignment candidates = %v, want %v", got, want)
	}
}
