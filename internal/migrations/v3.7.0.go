package migrations

import (
	_ "embed"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

//go:embed contributor_roles.sql
var contributorRolesSQL string

//go:embed outbound_reviews.sql
var outboundReviewsSQL string

// V3_7_0 adds the Contributor role and review queue, and retires open
// addresses. Access that was open before the upgrade becomes explicit grants.
func V3_7_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, sql := range []string{contributorRolesSQL, outboundReviewsSQL} {
		if _, err := tx.Exec(sql); err != nil {
			return err
		}
	}
	return tx.Commit()
}
