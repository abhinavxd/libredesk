package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_11_0 adds ticket merge: source conversations point at the survivor.
func V2_11_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		ALTER TABLE conversations
			ADD COLUMN IF NOT EXISTS merged_into_uuid UUID REFERENCES conversations(uuid) ON DELETE SET NULL ON UPDATE CASCADE;
		CREATE INDEX IF NOT EXISTS index_conversations_on_merged_into_uuid
			ON conversations (merged_into_uuid)
			WHERE merged_into_uuid IS NOT NULL;
		UPDATE roles
		SET permissions = array_append(permissions, 'contacts:merge')
		WHERE name = 'Admin' AND NOT ('contacts:merge' = ANY(permissions));
	`)
	return err
}
