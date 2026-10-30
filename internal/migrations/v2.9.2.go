package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_9_2(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		INSERT INTO settings ("key", value) VALUES
			('app.time_format', '"12h"'::jsonb)
		ON CONFLICT ("key") DO NOTHING;
	`)
	return err
}
