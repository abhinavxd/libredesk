package migrations

import (
	_ "embed"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

//go:embed activation_keys.sql
var activationKeysSQL string

// V3_10_0 adds Key Distribution: activation key pools per app, the setting
// that turns it on, and the Admin permission that manages it.
func V3_10_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(activationKeysSQL); err != nil {
		return err
	}
	return tx.Commit()
}
