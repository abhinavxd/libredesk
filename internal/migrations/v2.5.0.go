package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_5_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`INSERT INTO settings (key, value) VALUES ('app.show_conversation_subject', 'true'::jsonb) ON CONFLICT (key) DO NOTHING;`); err != nil {
		return err
	}

	// Drafts are now stored separately per type (reply / private note).
	if _, err := db.Exec(`ALTER TABLE conversation_drafts ADD COLUMN IF NOT EXISTS type TEXT NOT NULL DEFAULT 'reply';`); err != nil {
		return err
	}
	if _, err := db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'constraint_conversation_drafts_on_type') THEN
				ALTER TABLE conversation_drafts ADD CONSTRAINT constraint_conversation_drafts_on_type CHECK (type IN ('reply', 'private_note'));
			END IF;
		END$$;
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`DROP INDEX IF EXISTS index_uniq_conversation_drafts_on_conversation_id_and_user_id;`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS index_uniq_conversation_drafts_on_conversation_id_and_user_id_and_type ON conversation_drafts (conversation_id, user_id, type);`); err != nil {
		return err
	}

// Seed default business hours and SLA policy. mirrors the INSERTs at the tail of schema.sql.
// guarded on emptiness, not name, so configured installs aren't given a stray row.


	if _, err := db.Exec(`
		INSERT INTO business_hours ("name", description, is_always_open, hours, holidays)
		SELECT
			'Default',
			'Default business hours, Monday to Friday, 09:00 to 17:00.',
			false,
			'{"Monday": {"open": "09:00", "close": "17:00"}, "Tuesday": {"open": "09:00", "close": "17:00"}, "Wednesday": {"open": "09:00", "close": "17:00"}, "Thursday": {"open": "09:00", "close": "17:00"}, "Friday": {"open": "09:00", "close": "17:00"}}'::jsonb,
			'[]'::jsonb
		WHERE NOT EXISTS (
			SELECT 1 FROM business_hours
		);
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`
		INSERT INTO sla_policies ("name", description, first_response_time, resolution_time, next_response_time, notifications)
		SELECT
			'Default',
			'Default SLA policy, first response within 1 hour and resolution within 24 hours.',
			'1h',
			'24h',
			NULL::text,
			'[]'::jsonb
		WHERE NOT EXISTS (
			SELECT 1 FROM sla_policies
		);
	`); err != nil {
		return err
	}
	return nil
}
