package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_6_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`ALTER TABLE inboxes ADD COLUMN IF NOT EXISTS disconnected_at TIMESTAMPTZ NULL;`)
	return err
}
