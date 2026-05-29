package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_3_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	langMap := map[string]string{
		"da": "da-DK",
		"de": "de-DE",
		"en": "en-US",
		"es": "es-ES",
		"fa": "fa-IR",
		"fr": "fr-FR",
		"it": "it-IT",
		"ja": "ja-JP",
		"mr": "mr-IN",
	}

	for localeCode, localeRegion := range langMap {
		if _, err := db.Exec(`
			UPDATE settings SET value = to_jsonb($1::text), updated_at = now()
			WHERE key = 'app.lang' AND value = to_jsonb($2::text);
		`, localeRegion, localeCode); err != nil {
			return err
		}

		if _, err := db.Exec(`
			UPDATE inboxes
			SET config = jsonb_set(config, '{language}', to_jsonb($1::text)), updated_at = now()
			WHERE channel = 'livechat' AND config->>'language' = $2;
		`, localeRegion, localeCode); err != nil {
			return err
		}

		if _, err := db.Exec(`
			UPDATE inboxes
			SET config = jsonb_set(config, '{fallback_language}', to_jsonb($1::text)), updated_at = now()
			WHERE channel = 'livechat' AND config->>'fallback_language' = $2;
		`, localeRegion, localeCode); err != nil {
			return err
		}
	}

	// Dedupe legacy pending applied_slas per conversation before adding the partial unique index.
	if _, err := db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_indexes
				WHERE indexname = 'index_applied_slas_unique_pending_per_conv'
				  AND tablename = 'applied_slas'
			) THEN
				WITH ranked AS (
					SELECT id, row_number() OVER (PARTITION BY conversation_id ORDER BY created_at DESC, id DESC) AS rn
					FROM applied_slas
					WHERE status = 'pending'
				)
				DELETE FROM applied_slas WHERE id IN (SELECT id FROM ranked WHERE rn > 1);

				CREATE UNIQUE INDEX index_applied_slas_unique_pending_per_conv
				ON applied_slas(conversation_id) WHERE status = 'pending';
			END IF;
		END $$;
	`); err != nil {
		return err
	}

	return nil
}
