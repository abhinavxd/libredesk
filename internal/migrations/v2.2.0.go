package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_2_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'conversation_status_category') THEN
				CREATE TYPE conversation_status_category AS ENUM ('open', 'waiting', 'resolved');
			END IF;
		END$$;
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		ALTER TABLE conversation_statuses
		ADD COLUMN IF NOT EXISTS category conversation_status_category NOT NULL DEFAULT 'open';
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`UPDATE conversation_statuses SET category = 'waiting'  WHERE name = 'Snoozed'`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`UPDATE conversation_statuses SET category = 'resolved' WHERE name IN ('Resolved', 'Closed')`)
	if err != nil {
		return err
	}

	return nil
}
