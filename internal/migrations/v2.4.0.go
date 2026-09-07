package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_4_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`ALTER TABLE inboxes ADD COLUMN IF NOT EXISTS from_name_template TEXT NOT NULL DEFAULT '';`)
	return err
}
