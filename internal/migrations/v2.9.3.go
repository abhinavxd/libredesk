package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_9_3(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if err := createIndexConcurrently(db, "index_conversations_on_last_message_at_desc", `CREATE INDEX CONCURRENTLY IF NOT EXISTS index_conversations_on_last_message_at_desc ON conversations (last_message_at DESC NULLS LAST)`); err != nil {
		return err
	}
	if err := createIndexConcurrently(db, "index_conversation_messages_on_sender_id_and_created_at", `CREATE INDEX CONCURRENTLY IF NOT EXISTS index_conversation_messages_on_sender_id_and_created_at ON conversation_messages (sender_id, created_at)`); err != nil {
		return err
	}
	return createIndexConcurrently(db, "index_conversation_tags_on_tag_id_and_conversation_id", `CREATE INDEX CONCURRENTLY IF NOT EXISTS index_conversation_tags_on_tag_id_and_conversation_id ON conversation_tags (tag_id, conversation_id)`)
}
