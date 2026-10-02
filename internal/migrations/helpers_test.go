package migrations

import (
	"testing"

	"github.com/jmoiron/sqlx"
)

// restoreOpenAddressFlag puts back the v3.5–v3.6 email_addresses.restricted
// column that v3.7.0 removes, so earlier migrations run against the shape they
// actually upgraded.
func restoreOpenAddressFlag(t *testing.T, db *sqlx.DB) {
	t.Helper()
	db.MustExec(`ALTER TABLE email_addresses ADD COLUMN IF NOT EXISTS restricted BOOLEAN NOT NULL DEFAULT TRUE`)
}
