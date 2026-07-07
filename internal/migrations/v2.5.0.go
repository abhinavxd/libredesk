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
	return nil
}
