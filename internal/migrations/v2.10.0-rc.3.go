package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_10_0_RC3 adds the conversation retention setting, off by default.
func V2_10_0_RC3(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`INSERT INTO settings ("key", value) VALUES
		('app.conversation_retention_days', '0'::jsonb),
		('app.conversation_retention_delete_contacts', 'false'::jsonb)
		ON CONFLICT ("key") DO NOTHING;`)
	return err
}
