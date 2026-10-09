package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_9_0_RC9(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`ALTER TABLE notification_email_queue ADD COLUMN IF NOT EXISTS message_id BIGINT REFERENCES conversation_messages(id) ON DELETE CASCADE ON UPDATE CASCADE`); err != nil {
		return err
	}
	_, err := db.Exec(`UPDATE notification_email_queue q SET message_id = n.message_id FROM user_notifications n WHERE q.notification_id = n.id AND q.message_id IS NULL`)
	return err
}
