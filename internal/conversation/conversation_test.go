package conversation

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	am "github.com/abhinavxd/libredesk/internal/authz/models"
	"github.com/abhinavxd/libredesk/internal/automation"
	automodels "github.com/abhinavxd/libredesk/internal/automation/models"
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/csat"
	customAttribute "github.com/abhinavxd/libredesk/internal/custom_attribute"
	custommodels "github.com/abhinavxd/libredesk/internal/custom_attribute/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/inbox"
	tmodels "github.com/abhinavxd/libredesk/internal/team/models"
	"github.com/abhinavxd/libredesk/internal/template"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/abhinavxd/libredesk/internal/user"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/abhinavxd/libredesk/internal/ws"
	wsmodels "github.com/abhinavxd/libredesk/internal/ws/models"
	"github.com/stretchr/testify/require"
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

func TestAutomationCustomAttributeActions(t *testing.T) {
	m, conv, actor := automationActionFixture(t, "automation_attributes")
	for _, tc := range []struct {
		name, scope, dataType, value string
		want                         any
	}{
		{"text", "conversation", "text", "Café 日本語", "Café 日本語"},
		{"number", "conversation", "number", "1.5", float64(1.5)},
		{"zero", "contact", "number", "0", float64(0)},
		{"false", "conversation", "checkbox", "false", false},
		{"date", "contact", "date", "2026-10-25", "2026-10-25"},
		{"list", "contact", "list", "vip", "vip"},
		{"link", "conversation", "link", "https://example.com", "https://example.com"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			attribute, err := m.customAttributeStore.(*customAttribute.Manager).Create(custommodels.CustomAttribute{
				Name: tc.name, Key: "field_" + tc.name, AppliesTo: tc.scope, DataType: tc.dataType, Values: []string{"vip", "standard"}, Description: "Test",
			})
			require.NoError(t, err)
			id := attribute.ID
			actionType, removeType := automodels.ActionSetConversationAttribute, automodels.ActionRemoveConversationAttribute
			if tc.scope == "contact" {
				actionType, removeType = automodels.ActionSetContactAttribute, automodels.ActionRemoveContactAttribute
			}
			for range 2 {
				err = m.ApplyAction(automodels.RuleAction{Type: actionType, Value: []string{strconv.Itoa(id), tc.value}}, conv, actor)
				require.NoError(t, err)
			}
			fresh, err := m.GetConversation(0 /** id **/, conv.UUID, "" /** refNum **/)
			require.NoError(t, err)
			raw := fresh.CustomAttributes
			if tc.scope == "contact" {
				raw = fresh.Contact.CustomAttributes
			}
			var values map[string]any
			require.NoError(t, json.Unmarshal(raw, &values))
			require.Equal(t, tc.want, values["field_"+tc.name])
			require.Equal(t, "untouched", values["keep"])
			for range 2 {
				require.NoError(t, m.ApplyAction(automodels.RuleAction{Type: removeType, Value: []string{strconv.Itoa(id)}}, conv, actor))
			}
			fresh, err = m.GetConversation(0 /** id **/, conv.UUID, "" /** refNum **/)
			require.NoError(t, err)
			raw = fresh.CustomAttributes
			if tc.scope == "contact" {
				raw = fresh.Contact.CustomAttributes
			}
			values = nil
			require.NoError(t, json.Unmarshal(raw, &values))
			_, exists := values["field_"+tc.name]
			require.False(t, exists)
			require.Equal(t, "untouched", values["keep"])
		})
	}
	attribute, err := m.customAttributeStore.(*customAttribute.Manager).Create(custommodels.CustomAttribute{
		Name: "Scoped", Key: "scoped", AppliesTo: "contact", DataType: "text", Description: "Test", Values: []string{},
	})
	require.NoError(t, err)
	id := attribute.ID
	require.Error(t, m.ApplyAction(automodels.RuleAction{Type: automodels.ActionSetConversationAttribute, Value: []string{strconv.Itoa(id), "wrong scope"}}, conv, actor))
	for _, id := range []string{"", "0", "-1", "invalid", "999999999"} {
		require.Error(t, m.ApplyAction(automodels.RuleAction{Type: automodels.ActionSetContactAttribute, Value: []string{id, "value"}}, conv, actor))
	}
	require.Error(t, m.ApplyAction(automodels.RuleAction{Type: automodels.ActionSetContactAttribute, Value: []string{strconv.Itoa(id)}}, conv, actor))
}

func TestAutomationAssignmentSubjectAndTagActions(t *testing.T) {
	m, conv, actor := automationActionFixture(t, "automation_actions")
	var teamID int
	require.NoError(t, m.db.Get(&teamID, `INSERT INTO teams (name, conversation_assignment_type) VALUES ('Support', 'Manual') RETURNING id`))
	for _, action := range []string{automodels.ActionUnassignUser, automodels.ActionUnassignTeam} {
		t.Run(action, func(t *testing.T) {
			_, err := m.db.Exec(`UPDATE conversations SET assigned_user_id = $1, assigned_team_id = $2 WHERE id = $3`, actor.ID, teamID, conv.ID)
			require.NoError(t, err)
			for range 2 {
				require.NoError(t, m.ApplyAction(automodels.RuleAction{Type: action}, conv, actor))
			}
			fresh, err := m.GetConversation(0 /** id **/, conv.UUID, "" /** refNum **/)
			require.NoError(t, err)
			if action == automodels.ActionUnassignUser {
				require.False(t, fresh.AssignedUserID.Valid)
				require.Equal(t, teamID, fresh.AssignedTeamID.Int)
			} else {
				require.False(t, fresh.AssignedTeamID.Valid)
				require.Equal(t, actor.ID, fresh.AssignedUserID.Int)
			}
		})
	}
	require.NoError(t, m.ApplyAction(automodels.RuleAction{Type: automodels.ActionSetSubject, Value: []string{"Updated subject"}}, conv, actor))
	fresh, err := m.GetConversation(0 /** id **/, conv.UUID, "" /** refNum **/)
	require.NoError(t, err)
	require.Equal(t, "Updated subject", fresh.Subject.String)
	require.Error(t, m.ApplyAction(automodels.RuleAction{Type: automodels.ActionSetSubject, Value: []string{"  "}}, conv, actor))
	_, err = m.db.Exec(`INSERT INTO tags (name) VALUES ('vip'), ('billing')`)
	require.NoError(t, err)
	require.NoError(t, m.SetConversationTags(conv.UUID, automodels.ActionAddTags, []string{"vip", "billing"}, actor))
	for range 2 {
		require.NoError(t, m.ApplyAction(automodels.RuleAction{Type: automodels.ActionClearTags}, conv, actor))
	}
	var remaining int
	require.NoError(t, m.db.Get(&remaining, `SELECT COUNT(*) FROM conversation_tags WHERE conversation_id = $1`, conv.ID))
	require.Zero(t, remaining)
}

func TestAutomationAttributeValueValidation(t *testing.T) {
	for _, tc := range []struct{ dataType, value string }{
		{"number", "NaN"}, {"number", "Inf"}, {"number", "text"},
		{"checkbox", "0"}, {"date", "2026-02-30"}, {"list", "missing"}, {"unknown", "value"},
	} {
		t.Run(tc.dataType+"/"+tc.value, func(t *testing.T) {
			_, err := parseAutomationAttributeValue(custommodels.CustomAttribute{DataType: tc.dataType, Values: []string{"vip"}}, tc.value)
			require.Error(t, err)
		})
	}
	_, err := parseAutomationAttributeValue(custommodels.CustomAttribute{DataType: "text", Regex: "^valid$"}, "invalid")
	require.Error(t, err)
	_, err = parseAutomationAttributeValue(custommodels.CustomAttribute{DataType: "text", Regex: "["}, "value")
	require.Error(t, err)
}

func TestAutomationSLABreachConditionData(t *testing.T) {
	m, conv, _ := automationActionFixture(t, "automation_sla_data")
	var policyID, appliedID int
	require.NoError(t, m.db.Get(&policyID, `INSERT INTO sla_policies (name, first_response_time, resolution_time) VALUES ('Test policy', '1h', '4h') RETURNING id`))
	require.NoError(t, m.db.Get(&appliedID, `INSERT INTO applied_slas (conversation_id, sla_policy_id, first_response_deadline_at)
		VALUES ($1, $2, NOW() - INTERVAL '1 hour') RETURNING id`, conv.ID, policyID))
	fresh, err := m.GetConversation(0 /** id **/, conv.UUID, "" /** refNum **/)
	require.NoError(t, err)
	require.True(t, fresh.SLABreached)
	_, err = m.db.Exec(`UPDATE applied_slas SET first_response_met_at = first_response_deadline_at - INTERVAL '1 minute' WHERE id = $1`, appliedID)
	require.NoError(t, err)
	fresh, err = m.GetConversation(0 /** id **/, conv.UUID, "" /** refNum **/)
	require.NoError(t, err)
	require.False(t, fresh.SLABreached)
	_, err = m.db.Exec(`INSERT INTO sla_events (applied_sla_id, sla_policy_id, type, deadline_at)
		VALUES ($1, $2, 'next_response', NOW() - INTERVAL '1 hour')`, appliedID, policyID)
	require.NoError(t, err)
	fresh, err = m.GetConversation(0 /** id **/, conv.UUID, "" /** refNum **/)
	require.NoError(t, err)
	require.True(t, fresh.SLABreached)
}

func TestAutomationChangeEvents(t *testing.T) {
	m, conv, actor := automationActionFixture(t, "automation_events")
	csats, err := csat.New(csat.Opts{DB: m.db, Lo: m.lo, I18n: m.i18n})
	require.NoError(t, err)
	m.csatStore = csats
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		m.automation.Run(ctx, 1)
		close(done)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
		m.automation.Close()
	})

	for _, event := range []string{automodels.EventConversationTagsChange, automodels.EventConversationAttributesChange,
		automodels.EventConversationContactAttributesChange, automodels.EventConversationCSATSubmitted} {
		t.Run(event, func(t *testing.T) {
			condition := automodels.RuleDetail{Field: automodels.ConversationSubject, Operator: automodels.RuleOperatorSet}
			marker := "Matched " + event
			if event == automodels.EventConversationCSATSubmitted {
				condition = automodels.RuleDetail{Field: automodels.ConversationCSATFeedback, Operator: automodels.RuleOperatorContains, Value: "help"}
			}
			rules, err := json.Marshal([]automodels.Rule{{
				GroupOperator: automodels.OperatorAnd,
				Groups:        []automodels.RuleGroup{{LogicalOp: automodels.OperatorAnd, Rules: []automodels.RuleDetail{condition}}},
				Actions:       []automodels.RuleAction{{Type: automodels.ActionSetSubject, Value: []string{marker}}},
			}})
			require.NoError(t, err)
			record, err := m.automation.CreateRule(automodels.RuleRecord{Name: event, Type: automodels.RuleTypeConversationUpdate, Events: []string{event}, Rules: rules})
			require.NoError(t, err)
			switch event {
			case automodels.EventConversationTagsChange:
				_, err = m.db.Exec(`INSERT INTO tags (name) VALUES ('vip')`)
				require.NoError(t, err)
				err = m.SetConversationTags(conv.UUID, automodels.ActionAddTags, []string{"vip", "vip"}, actor)
			case automodels.EventConversationAttributesChange:
				err = m.UpdateConversationCustomAttributes(conv.UUID, map[string]any{"issue": "billing"}, actor)
			case automodels.EventConversationContactAttributesChange:
				attribute, attrErr := m.customAttributeStore.(*customAttribute.Manager).Create(custommodels.CustomAttribute{
					Name: "Plan", Key: "plan", AppliesTo: "contact", DataType: "text", Description: "Test", Values: []string{},
				})
				require.NoError(t, attrErr)
				err = m.ApplyAction(automodels.RuleAction{Type: automodels.ActionSetContactAttribute, Value: []string{strconv.Itoa(attribute.ID), "vip"}}, conv, actor)
			case automodels.EventConversationCSATSubmitted:
				response, responseErr := csats.Create(conv.ID)
				require.NoError(t, responseErr)
				err = m.UpdateCSATResponse(response.UUID, 0 /** rating **/, "Need help", nil /** meta **/)
			}
			require.NoError(t, err)
			require.Eventually(t, func() bool {
				var subject string
				return m.db.Get(&subject, `SELECT subject FROM conversations WHERE id = $1`, conv.ID) == nil && subject == marker
			}, 2*time.Second, 10*time.Millisecond)
			require.NoError(t, m.automation.DeleteRule(record.ID))
		})
	}
}

func TestAutomationPublicMessageTimestamps(t *testing.T) {
	m, conv, actor := automationActionFixture(t, "automation_public_message")
	publicAt := time.Now().Add(-26 * time.Hour).Truncate(time.Millisecond)
	_, err := m.db.Exec(`UPDATE conversations SET last_message_at = $2, last_interaction_at = $2,
		last_message_sender = 'contact', last_interaction_sender = 'contact' WHERE id = $1`, conv.ID, publicAt)
	require.NoError(t, err)
	for _, kind := range []string{models.MessageOutgoing, models.MessageActivity} {
		message := models.Message{
			ConversationID: conv.ID, ConversationUUID: conv.UUID, SenderID: actor.ID,
			SenderType: models.SenderTypeAgent, Type: kind, Private: true, Content: "Internal update",
		}
		require.NoError(t, m.InsertMessage(&message))
		fresh, err := m.GetConversation(0 /** id **/, conv.UUID, "" /** refNum **/)
		require.NoError(t, err)
		require.True(t, publicAt.Equal(fresh.LastInteractionAt.Time))
		require.Equal(t, "contact", fresh.LastInteractionSender.String)
		require.True(t, fresh.LastMessageAt.Time.After(publicAt))
		require.Equal(t, "agent", fresh.LastMessageSender.String)
	}
	publicReply := models.Message{
		ConversationID: conv.ID, ConversationUUID: conv.UUID, SenderID: actor.ID,
		SenderType: models.SenderTypeAgent, Type: models.MessageOutgoing, Status: models.MessageStatusSent, Content: "Public reply",
	}
	require.NoError(t, m.InsertMessage(&publicReply))
	fresh, err := m.GetConversation(0 /** id **/, conv.UUID, "" /** refNum **/)
	require.NoError(t, err)
	require.True(t, fresh.LastInteractionAt.Time.After(publicAt))
	require.Equal(t, "agent", fresh.LastInteractionSender.String)
}

func automationActionFixture(t *testing.T, name string) (*Manager, models.Conversation, umodels.User) {
	t.Helper()
	db := testutil.NewDB(t, name)
	lo := logf.New(logf.Opts{Level: logf.ErrorLevel})
	lang := testutil.NewI18n(t)
	users, err := user.New(lang, user.Opts{DB: db, Lo: &lo})
	require.NoError(t, err)
	attributes, err := customAttribute.New(customAttribute.Opts{DB: db, Lo: &lo, I18n: lang})
	require.NoError(t, err)
	inboxes, err := inbox.New(&lo, db, lang, "0123456789abcdef0123456789abcdef")
	require.NoError(t, err)
	engine, err := automation.New(automation.Opts{DB: db, Lo: &lo, I18n: lang})
	require.NoError(t, err)
	m := &Manager{db: db, lo: &lo, i18n: lang, userStore: users, inboxStore: inboxes, automation: engine,
		customAttributeStore: attributes, webhookStore: whatsAppReplyWebhook{}, wsHub: ws.NewHub(&lo, users),
		mediaStore: whatsAppReplyMedia{}, template: &template.Manager{}}
	require.NoError(t, dbutil.ScanSQLFile("queries.sql", &m.q, db, efs))
	engine.SetConversationStore(m)
	var contactID, actorID, inboxID int
	require.NoError(t, db.Get(&contactID, `INSERT INTO users (type, first_name, last_name, custom_attributes)
		VALUES ('contact', 'Customer', '', '{"keep":"untouched"}') RETURNING id`))
	require.NoError(t, db.Get(&actorID, `INSERT INTO users (type, first_name, last_name, email) VALUES ('agent', 'Agent', '', 'System') RETURNING id`))
	require.NoError(t, db.Get(&inboxID, `INSERT INTO inboxes (name, channel) VALUES ('Test chat', 'livechat') RETURNING id`))
	var uuid string
	require.NoError(t, db.Get(&uuid, `INSERT INTO conversations (contact_id, inbox_id, status_id, subject, custom_attributes)
		VALUES ($1, $2, (SELECT id FROM conversation_statuses WHERE category = 'open' LIMIT 1), 'Original', '{"keep":"untouched"}') RETURNING uuid`, contactID, inboxID))
	conv, err := m.GetConversation(0 /** id **/, uuid, "" /** refNum **/)
	require.NoError(t, err)
	engine.SetSystemUserID(actorID)
	return m, conv, umodels.User{ID: actorID, FirstName: "Agent"}
}
