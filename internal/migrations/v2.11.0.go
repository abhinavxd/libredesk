package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_11_0 enforces at most one active (enabled) guided form per inbox at the database level,
// matching the application now disabling any other enabled form on an inbox when one is enabled.
// Existing installs may already have more than one enabled form on the same inbox (the
// application only started enforcing this going forward), so the most recently updated one per
// inbox is kept enabled and the rest are disabled before the index is created.
func V2_11_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		UPDATE guided_forms SET enabled = false
		WHERE enabled = true AND id NOT IN (
			SELECT DISTINCT ON (inbox_id) id FROM guided_forms WHERE enabled = true
			ORDER BY inbox_id, updated_at DESC
		);
		CREATE UNIQUE INDEX IF NOT EXISTS index_unique_guided_forms_on_inbox_when_enabled ON guided_forms(inbox_id) WHERE enabled = true;
	`)
	return err
}
