package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_10_0_RC1(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS index_conversations_on_last_message_at_desc ON conversations (last_message_at DESC NULLS LAST);
		CREATE INDEX IF NOT EXISTS index_conversation_messages_on_sender_id_and_created_at ON conversation_messages (sender_id, created_at);
		CREATE INDEX IF NOT EXISTS index_conversation_tags_on_tag_id_and_conversation_id ON conversation_tags (tag_id, conversation_id);
	`)
	return err
}
