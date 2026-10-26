package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_9_1_RC2(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		ALTER TABLE teams ADD COLUMN IF NOT EXISTS skip_offline_agents BOOLEAN DEFAULT false NOT NULL;
	`)
	return err
}
