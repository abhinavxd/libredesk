package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_10_0 adds optional inbox/team/user filters so one webhook can
// target a Discord channel (or HTTP endpoint) for a subset of conversations.
func V2_10_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		ALTER TABLE webhooks
			ADD COLUMN IF NOT EXISTS inbox_ids INTEGER[] NOT NULL DEFAULT '{}',
			ADD COLUMN IF NOT EXISTS team_ids INTEGER[] NOT NULL DEFAULT '{}',
			ADD COLUMN IF NOT EXISTS user_ids INTEGER[] NOT NULL DEFAULT '{}';
	`)
	return err
}
