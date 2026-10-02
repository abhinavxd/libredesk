package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V3_9_0 retires search and conversation status management. The trigram index
// existed only for message search, and nobody can change a conversation's
// status, edit the status list or receive the status-changed webhook any more.
func V3_9_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		DROP INDEX IF EXISTS index_trgm_conversation_messages_on_text_content;
		UPDATE roles SET permissions = array_remove(array_remove(permissions, 'conversations:update_status'), 'status:manage')
		WHERE 'conversations:update_status' = ANY(permissions) OR 'status:manage' = ANY(permissions);
		-- Snoozing is retired: reopen anything still snoozed rather than leave it parked.
		UPDATE conversations SET snoozed_until = NULL, status_id = (SELECT id FROM conversation_statuses WHERE name = 'Open')
		WHERE status_id = (SELECT id FROM conversation_statuses WHERE name = 'Snoozed');
		-- Nothing fires the status event any more; the enum value stays for history.
		UPDATE webhooks SET events = array_remove(events, 'conversation.status_changed')
		WHERE 'conversation.status_changed' = ANY(events);
	`)
	return err
}
