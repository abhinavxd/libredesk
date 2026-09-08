package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_13_0 adds: a display name shown to visitors that can differ from a guided form's internal
// admin-facing name, a per-form abandoned-conversation auto-resolve timeout, and the "guided
// form bot" carve-out in the compact-agent listing queries so guided form bots are selectable
// in the normal "assign to" picker (the same carve-out AI assistants already had, since both
// have no email and were otherwise excluded by that WHERE clause).
func V2_13_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		ALTER TABLE guided_forms ADD COLUMN IF NOT EXISTS display_name TEXT NOT NULL DEFAULT '';
		ALTER TABLE guided_forms ADD COLUMN IF NOT EXISTS abandoned_timeout_minutes INTEGER NOT NULL DEFAULT 0;
		ALTER TABLE guided_form_events DROP CONSTRAINT IF EXISTS constraint_guided_form_events_on_type;
		ALTER TABLE guided_form_events ADD CONSTRAINT constraint_guided_form_events_on_type CHECK (type IN ('completed', 'handoff', 'abandoned'));
	`)
	return err
}
