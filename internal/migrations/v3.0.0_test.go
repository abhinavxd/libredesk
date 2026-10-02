package migrations

import (
	"github.com/jakedolan443/fernmail/internal/testutil"
	"testing"
)

func TestMailOnlyUpgradePreservesMessagesAndDisablesRetiredFeatures(t *testing.T) {
	db := testutil.NewDB(t, "mail_only_upgrade")
	db.MustExec(`
 CREATE TYPE view_visibility AS ENUM ('all', 'team', 'user');
 CREATE TABLE views(id SERIAL PRIMARY KEY, name TEXT NOT NULL, visibility view_visibility NOT NULL,
   filters JSONB NOT NULL DEFAULT '[]', user_id BIGINT, team_id INTEGER);
 CREATE TABLE macros(id int);
 CREATE TABLE contact_notes(id int);
 INSERT INTO inboxes (name, channel, csat_enabled) VALUES ('Mail', 'email', true), ('Old chat', 'livechat', true);
 INSERT INTO users (type, email, first_name) VALUES ('contact', 'sender@example.com', 'Sender');
 INSERT INTO conversations (contact_id, inbox_id, status_id, priority_id) SELECT (SELECT id FROM users WHERE email = 'sender@example.com'), (SELECT id FROM inboxes WHERE name = 'Mail'), (SELECT id FROM conversation_statuses WHERE name = 'Open'), 5;
 INSERT INTO conversation_drafts(conversation_id, user_id, type, content, meta) SELECT id, (SELECT id FROM users WHERE email = 'sender@example.com'), 'reply', 'Keep this draft', '{"macro_id": 1, "macro_actions": [], "attachments": []}' FROM conversations;
 `)
	for range 2 {
		if err := V3_0_0(db, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	for _, check := range []struct {
		query string
		want  int
	}{
		{`SELECT count(*) FROM conversations WHERE priority_id IS NULL AND assigned_user_id IS NULL`, 1},
		{`SELECT count(*) FROM conversation_drafts WHERE content = 'Keep this draft' AND NOT meta ? 'macro_id' AND meta ? 'attachments'`, 1},
		{`SELECT count(*) FROM inboxes WHERE channel = 'email' AND NOT csat_enabled`, 1},
		{`SELECT count(*) FROM inboxes WHERE channel <> 'email' AND enabled`, 0},
		{`SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name IN ('macros','contact_notes','ai_tools','sla_policies','help_articles')`, 0},
	} {
		if err := db.Get(&count, check.query); err != nil {
			t.Fatal(err)
		}
		if count != check.want {
			t.Errorf("%s: got %d want %d", check.query, count, check.want)
		}
	}
}
