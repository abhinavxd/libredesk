package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_10_0_RC3(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		ALTER TABLE teams ADD COLUMN IF NOT EXISTS skip_offline_agents BOOLEAN DEFAULT false NOT NULL;

		ALTER TABLE ai_tools ADD COLUMN IF NOT EXISTS display_name TEXT NOT NULL DEFAULT '';
		UPDATE ai_tools SET display_name = name WHERE display_name = '';
		ALTER TABLE ai_tools DROP CONSTRAINT IF EXISTS constraint_ai_tools_on_display_name;
		ALTER TABLE ai_tools ADD CONSTRAINT constraint_ai_tools_on_display_name CHECK (length(display_name) <= 140);
	`)
	return err
}
