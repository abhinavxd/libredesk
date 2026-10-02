package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V3_8_0 folds the favicon into the site logo and makes the Fernmail title the
// unset default. A custom favicon becomes the logo when no logo is set; the
// bundled favicon is dropped, as it is what an unset logo already shows.
func V3_8_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		WITH root AS (
			SELECT rtrim(COALESCE((SELECT value #>> '{}' FROM settings WHERE key = 'app.root_url'), ''), '/') AS url
		)
		UPDATE settings logo SET value = favicon.value, updated_at = now()
		FROM settings favicon, root
		WHERE logo.key = 'app.logo_url' AND COALESCE(logo.value #>> '{}', '') = ''
		AND favicon.key = 'app.favicon_url'
		AND COALESCE(favicon.value #>> '{}', '') NOT IN (
			'', '/favicon.svg', '/favicon.ico',
			'http://localhost:9000/favicon.svg', 'http://localhost:9000/favicon.ico',
			root.url || '/favicon.svg', root.url || '/favicon.ico'
		);
		DELETE FROM settings WHERE key = 'app.favicon_url';
		UPDATE settings SET value = '""'::jsonb, updated_at = now()
		WHERE key = 'app.site_name' AND value #>> '{}' = 'Fernmail';
	`)
	return err
}
