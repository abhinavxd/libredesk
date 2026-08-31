package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_7_0 adds webhook delivery types so Discord incoming webhooks can
// receive formatted embeds instead of raw Libredesk JSON.
func V2_7_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		ALTER TABLE webhooks
			ADD COLUMN IF NOT EXISTS delivery TEXT NOT NULL DEFAULT 'http';

		ALTER TABLE webhooks DROP CONSTRAINT IF EXISTS constraint_webhooks_on_delivery;
		ALTER TABLE webhooks
			ADD CONSTRAINT constraint_webhooks_on_delivery CHECK (delivery IN ('http', 'discord'));

		UPDATE webhooks
		SET delivery = 'discord'
		WHERE delivery = 'http'
		  AND (
			url ~* '^https://((ptb|canary)\.)?discord(app)?\.com/api/webhooks/'
			OR url ~* '^https://((ptb|canary)\.)?discord(app)?\.com/api/v\d+/webhooks/'
		  );
	`)
	return err
}
