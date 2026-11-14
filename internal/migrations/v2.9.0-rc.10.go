package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_9_0_RC10(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		ALTER TABLE macros DROP CONSTRAINT IF EXISTS message_content_length;
		ALTER TABLE macros ADD CONSTRAINT message_content_length CHECK (length(message_content) <= 100000);
	`)
	return err
}
