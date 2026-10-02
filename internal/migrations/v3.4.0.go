package migrations

import (
	_ "embed"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

//go:embed inbox_access.sql
var inboxAccessSQL string

// V3_4_0 adds optional access restrictions to individual email inboxes.
func V3_4_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(inboxAccessSQL)
	return err
}
