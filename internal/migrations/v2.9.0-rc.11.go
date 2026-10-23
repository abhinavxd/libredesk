package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_9_0_RC11(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		ALTER TABLE ai_assistants ADD COLUMN IF NOT EXISTS response_interval INTEGER NOT NULL DEFAULT 0
		CONSTRAINT constraint_ai_assistants_on_response_interval CHECK (response_interval >= 0 AND response_interval <= 300);
	`)
	return err
}
