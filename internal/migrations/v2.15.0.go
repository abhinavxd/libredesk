package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_15_0 adds skills, TOTP columns, and the Light agent role.
func V2_15_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		ALTER TABLE users
			ADD COLUMN IF NOT EXISTS totp_secret TEXT,
			ADD COLUMN IF NOT EXISTS totp_enabled BOOLEAN NOT NULL DEFAULT FALSE;

		CREATE TABLE IF NOT EXISTS skills (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			name TEXT NOT NULL UNIQUE
		);

		CREATE TABLE IF NOT EXISTS agent_skills (
			user_id BIGINT REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
			skill_id INT REFERENCES skills(id) ON DELETE CASCADE ON UPDATE CASCADE,
			PRIMARY KEY (user_id, skill_id)
		);
		CREATE INDEX IF NOT EXISTS index_agent_skills_on_skill_id ON agent_skills (skill_id);

		INSERT INTO roles ("name", description, permissions)
		SELECT
			'Light agent',
			'Can view tickets and add private notes. Cannot reply to customers or change assignees.',
			'{conversations:read_all,conversations:read_unassigned,conversations:read_assigned,conversations:read_team_inbox,conversations:read_team_all,conversations:read,messages:read,messages:write_private,view:manage}'
		WHERE NOT EXISTS (SELECT 1 FROM roles WHERE name = 'Light agent');
	`)
	return err
}
