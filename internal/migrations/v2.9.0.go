package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_9_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`
		INSERT INTO settings ("key", value)
		VALUES
			('portal.enabled', 'false'::jsonb),
			('portal.inbox_id', '0'::jsonb),
			('portal.help_center_id', '0'::jsonb),
			('portal.livechat_inbox_id', '0'::jsonb),
			('portal.tickets_from_article_only', 'false'::jsonb),
			('portal.form_id', '0'::jsonb)
		ON CONFLICT ("key") DO NOTHING;
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS portal_forms (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			"name" TEXT NOT NULL,
			ask_subject BOOLEAN NOT NULL DEFAULT true,
			fields JSONB NOT NULL DEFAULT '[]'::jsonb,
			CONSTRAINT constraint_portal_forms_on_name CHECK (length("name") <= 140)
		);
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		DELETE FROM settings WHERE "key" IN ('portal.login_url', 'portal.login_label');
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		ALTER TABLE oidc
		ADD COLUMN IF NOT EXISTS enabled_for_portal BOOLEAN NOT NULL DEFAULT false;
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		ALTER TABLE help_articles
		ADD COLUMN IF NOT EXISTS portal_form_id INTEGER NULL REFERENCES portal_forms(id) ON DELETE SET NULL;
	`); err != nil {
		return err
	}

	return nil
}
