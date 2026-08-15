package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_8_1 adds portal intake configuration to conversation custom attributes.
func V2_8_1(db *sqlx.DB, _ stuffbin.FileSystem, _ *koanf.Koanf) error {
	_, err := db.Exec(`
		ALTER TABLE custom_attribute_definitions
		ADD COLUMN IF NOT EXISTS portal_required BOOLEAN DEFAULT FALSE NOT NULL;
	`)
	return err
}
