package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_5_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`INSERT INTO settings (key, value) VALUES ('app.set_away_on_login', 'false'::jsonb) ON CONFLICT (key) DO NOTHING;`); err != nil {
		return err
	}
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
	return nil
}
