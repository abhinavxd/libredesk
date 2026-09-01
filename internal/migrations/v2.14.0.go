package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_14_0 links child and follow-up tickets to a parent conversation.
func V2_14_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		ALTER TABLE conversations
			ADD COLUMN IF NOT EXISTS parent_uuid UUID REFERENCES conversations(uuid) ON DELETE SET NULL ON UPDATE CASCADE,
			ADD COLUMN IF NOT EXISTS origin TEXT NOT NULL DEFAULT '';
		CREATE INDEX IF NOT EXISTS index_conversations_on_parent_uuid
			ON conversations (parent_uuid)
			WHERE parent_uuid IS NOT NULL;
	`)
	return err
}
