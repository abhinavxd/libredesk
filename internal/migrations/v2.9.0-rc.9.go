package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_9_0_RC9(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		ALTER TABLE users ADD COLUMN IF NOT EXISTS session_version INT NOT NULL DEFAULT 1;
		ALTER TABLE media ADD COLUMN IF NOT EXISTS uploaded_by BIGINT REFERENCES users(id) ON DELETE SET NULL;
	`)
	return err
}
