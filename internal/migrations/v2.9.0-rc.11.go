package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_9_0_RC11(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`ALTER TABLE oidc ADD COLUMN IF NOT EXISTS verify_email BOOLEAN DEFAULT TRUE NOT NULL;`)
	return err
}
