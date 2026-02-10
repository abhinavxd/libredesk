package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V1_0_1 updates the database schema to v1.0.1.
func V1_0_1(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	// Backfill enable_plus_addressing to true for existing email inboxes
	// that don't have this field in their config JSON.
	_, err := db.Exec(`
		UPDATE inboxes
		SET config = jsonb_set(config, '{enable_plus_addressing}', 'true'::jsonb, true)
		WHERE channel = 'email'
		AND NOT (config ? 'enable_plus_addressing');
	`)
	return err
}
