package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V1_0_2 updates the database schema to v1.0.2.
func V1_0_2(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		ALTER TABLE inboxes ADD COLUMN IF NOT EXISTS from_name_template TEXT NOT NULL DEFAULT '';
	`)
	return err
}
