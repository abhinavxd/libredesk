package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_1_0 updates the database schema to v2.1.0.
func V2_1_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		ALTER TABLE inboxes ADD COLUMN IF NOT EXISTS prompt_tags_on_reply bool DEFAULT false NOT NULL;
	`)
	if err != nil {
		return err
	}

	return nil
}
