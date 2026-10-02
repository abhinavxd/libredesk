package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V3_3_0 adopts the Fernmail defaults without changing custom branding.
func V3_3_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		UPDATE settings SET value = '"Fernmail"'::jsonb
		WHERE key = 'app.site_name' AND lower(value #>> '{}') = 'libredesk';
		UPDATE settings SET value = '"/favicon.svg"'::jsonb
		WHERE key = 'app.favicon_url'
		AND value IN ('"http://localhost:9000/favicon.ico"'::jsonb, '"/favicon.ico"'::jsonb);
	`)
	return err
}
