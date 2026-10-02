package migrations

import (
	_ "embed"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

//go:embed mail_reliability.sql
var mailReliabilitySQL string

//go:embed media_ownership.sql
var mediaOwnershipSQL string

// V3_6_0 adds durable mail processing and draft-upload ownership. Views remain
// retired; nothing in this migration restores their tables, permissions or UI.
func V3_6_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, sql := range []string{mailReliabilitySQL, mediaOwnershipSQL} {
		if _, err := tx.Exec(sql); err != nil {
			return err
		}
	}
	return tx.Commit()
}
