package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_12_0 adds a per-form toggle for the guided-form "talk to a human" skip escape hatch,
// defaulting existing forms to the previous always-on behavior.
func V2_12_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`ALTER TABLE guided_forms ADD COLUMN IF NOT EXISTS allow_skip_to_human BOOLEAN NOT NULL DEFAULT true;`)
	return err
}
