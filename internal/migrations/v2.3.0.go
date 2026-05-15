package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_3_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	// Add 'telegram' to the channels enum type.
	_, err := db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_enum WHERE enumlabel = 'telegram' AND enumtypid = (SELECT oid FROM pg_type WHERE typname = 'channels')) THEN
				ALTER TYPE channels ADD VALUE 'telegram';
			END IF;
		END$$;
	`)
	if err != nil {
		return err
	}

	return nil
}
