package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_10_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
ALTER TABLE users ADD COLUMN IF NOT EXISTS merged_into_id BIGINT REFERENCES users(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS index_users_on_merged_into_id ON users(merged_into_id) WHERE merged_into_id IS NOT NULL;
ALTER TYPE activity_log_type ADD VALUE IF NOT EXISTS 'contact_merged';
UPDATE roles SET permissions = array_append(permissions, 'contacts:merge')
WHERE name = 'Admin' AND NOT ('contacts:merge' = ANY(permissions));
`)
	return err
}
