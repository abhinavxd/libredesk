package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_9_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		ALTER TABLE users
		ADD COLUMN IF NOT EXISTS default_inbox TEXT NOT NULL DEFAULT 'assigned';
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		ALTER TABLE users DROP CONSTRAINT IF EXISTS constraint_users_on_default_inbox;
		ALTER TABLE users ADD CONSTRAINT constraint_users_on_default_inbox
			CHECK (default_inbox IN ('assigned', 'mentioned', 'unassigned', 'all'));
	`)
	return err
}
