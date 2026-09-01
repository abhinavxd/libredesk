package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_13_0 adds side conversations (internal email threads on a ticket).
func V2_13_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS side_conversations (
			id SERIAL PRIMARY KEY,
			uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			conversation_id INT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE ON UPDATE CASCADE,
			subject TEXT NOT NULL DEFAULT '',
			recipients TEXT[] NOT NULL DEFAULT '{}',
			created_by INT NOT NULL REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE
		);
		CREATE INDEX IF NOT EXISTS index_side_conversations_on_conversation_id ON side_conversations (conversation_id);
		CREATE TABLE IF NOT EXISTS side_messages (
			id SERIAL PRIMARY KEY,
			uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
			created_at TIMESTAMPTZ DEFAULT NOW(),
			side_conversation_id INT NOT NULL REFERENCES side_conversations(id) ON DELETE CASCADE ON UPDATE CASCADE,
			sender_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
			direction TEXT NOT NULL DEFAULT 'outgoing',
			content TEXT NOT NULL DEFAULT '',
			content_type TEXT NOT NULL DEFAULT 'html',
			source_id TEXT NULL
		);
		CREATE INDEX IF NOT EXISTS index_side_messages_on_side_conversation_id ON side_messages (side_conversation_id);
	`)
	return err
}
