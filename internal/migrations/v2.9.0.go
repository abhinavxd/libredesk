package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_9_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`
		UPDATE roles
		SET permissions = array_append(permissions, 'messages:write_private')
		WHERE 'messages:write' = ANY(permissions)
		AND NOT ('messages:write_private' = ANY(permissions));
	`); err != nil {
		return err
	}

	// Remembers the identity an external integration last supplied for a contact, so a later sync
	// can tell its own value from one an agent corrected by hand.
	if _, err := db.Exec(`
		ALTER TABLE users ADD COLUMN IF NOT EXISTS external_sync JSONB DEFAULT '{}'::jsonb NOT NULL;
	`); err != nil {
		return err
	}

	return nil
}
