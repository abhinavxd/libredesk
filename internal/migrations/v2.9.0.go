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
	// Until now an integration overwrote every identity field of the contacts it created on each
	// visit, so what those contacts hold is what it last supplied. Recording that keeps them
	// following the integration; with an empty record a later change would look like an agent's
	// correction and be kept forever. Only rows nothing has recorded yet are touched.
	if _, err := db.Exec(`
		UPDATE users
		SET external_sync = jsonb_strip_nulls(jsonb_build_object(
			'first_name', NULLIF(first_name, ''),
			'last_name', NULLIF(last_name, ''),
			'email', NULLIF(email, ''),
			'phone_number', NULLIF(phone_number, ''),
			'phone_number_country_code', NULLIF(phone_number_country_code, '')))
		WHERE type = 'contact' AND external_user_id IS NOT NULL AND deleted_at IS NULL AND external_sync = '{}'::jsonb;
	`); err != nil {
		return err
	}

	return nil
}
