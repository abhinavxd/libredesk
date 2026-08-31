package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_7_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`
		ALTER TABLE media ADD COLUMN IF NOT EXISTS uploaded_by INT NULL REFERENCES users(id) ON DELETE SET NULL;
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS index_media_on_uploaded_by ON media(uploaded_by);
	`); err != nil {
		return err
	}

	return nil
}
