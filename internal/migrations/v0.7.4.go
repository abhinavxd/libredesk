package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V0_7_4 updates the database schema to v0.7.4.
func V0_7_4(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	// Rename phone_number_calling_code to phone_number_country_code
	// This column will now store country codes (US, CA, GB) instead of calling codes (+1, +44)
	_, err := db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = 'users' AND column_name = 'phone_number_country_code'
			) AND EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = 'users' AND column_name = 'phone_number_calling_code'
			) THEN
				ALTER TABLE users
				RENAME COLUMN phone_number_calling_code TO phone_number_country_code;
			END IF;
		END $$;
	`)
	if err != nil {
		return err
	}

	// Rename the constraint to match the new column name
	_, err = db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM information_schema.constraint_column_usage
				WHERE constraint_name = 'constraint_users_on_phone_number_country_code'
			) AND EXISTS (
				SELECT 1 FROM information_schema.constraint_column_usage
				WHERE constraint_name = 'constraint_users_on_phone_number_calling_code'
			) THEN
				ALTER TABLE users
				RENAME CONSTRAINT constraint_users_on_phone_number_calling_code TO constraint_users_on_phone_number_country_code;
			END IF;
		END $$;
	`)
	if err != nil {
		return err
	}

	return nil
}