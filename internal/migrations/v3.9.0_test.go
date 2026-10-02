package migrations

import (
	"testing"

	"github.com/jakedolan443/fernmail/internal/testutil"
)

func TestRetiringSearchAndStatusesDropsIndexAndPermissions(t *testing.T) {
	db := testutil.NewDB(t, "retire_search_statuses")
	// Recreate the v3.8 shape: the search index and the status permissions.
	db.MustExec(`
		CREATE INDEX index_trgm_conversation_messages_on_text_content ON conversation_messages USING GIN (text_content gin_trgm_ops);
		UPDATE roles SET permissions = permissions || '{conversations:update_status}' WHERE name IN ('Admin', 'Agent');
		UPDATE roles SET permissions = permissions || '{status:manage}' WHERE name = 'Admin';
		INSERT INTO roles (name, description, permissions) VALUES ('Custom', '', '{conversations:read,conversations:update_status}');
		INSERT INTO webhooks (name, url, events) VALUES ('Hook', 'https://example.test/hook', '{conversation.created,conversation.status_changed}');
		INSERT INTO users (type, email, first_name) VALUES ('contact', 'sender@example.test', 'Sender');
		INSERT INTO inboxes (name, channel) VALUES ('Mail', 'email');
		INSERT INTO conversations (contact_id, inbox_id, status_id, snoozed_until)
		SELECT u.id, i.id, s.id, NOW() + INTERVAL '3 days' FROM users u, inboxes i, conversation_statuses s WHERE s.name = 'Snoozed';
	`)
	for range 2 {
		if err := V3_9_0(db, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	var ok bool
	for _, query := range []string{
		`SELECT to_regclass('index_trgm_conversation_messages_on_text_content') IS NULL`,
		`SELECT NOT EXISTS(SELECT 1 FROM roles WHERE 'conversations:update_status' = ANY(permissions) OR 'status:manage' = ANY(permissions))`,
		`SELECT permissions = '{conversations:read}' FROM roles WHERE name = 'Custom'`,
		`SELECT 'users:manage' = ANY(permissions) FROM roles WHERE name = 'Admin'`,
		`SELECT events = '{conversation.created}' FROM webhooks WHERE name = 'Hook'`,
		`SELECT s.name = 'Open' AND c.snoozed_until IS NULL FROM conversations c JOIN conversation_statuses s ON s.id = c.status_id`,
	} {
		if err := db.Get(&ok, query); err != nil || !ok {
			t.Fatalf("%s: %v %v", query, ok, err)
		}
	}
}
