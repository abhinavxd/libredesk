package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V1_1_0 adds the from_name_template column to the inboxes table.
func V1_1_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		ALTER TABLE inboxes ADD COLUMN IF NOT EXISTS from_name_template TEXT NOT NULL DEFAULT '';
	`)
	return err
}
