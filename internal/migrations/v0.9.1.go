package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V0_9_1 updates the database schema to v0.9.1.
func V0_9_1(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	// Create conversation_drafts table and index if they don't exist
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS conversation_drafts (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			conversation_id BIGINT REFERENCES conversations(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
			user_id BIGINT REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
			content TEXT NOT NULL,
			meta JSONB DEFAULT '{}'::jsonb NOT NULL
		);

		CREATE UNIQUE INDEX IF NOT EXISTS index_uniq_conversation_drafts_on_conversation_id_and_user_id 
		ON conversation_drafts (conversation_id, user_id);
	`)
	if err != nil {
		return err
	}

	// Add conversations:read_team_all permission to Admin and Agent roles
	_, err = db.Exec(`
		UPDATE roles
		SET permissions = array_append(permissions, 'conversations:read_team_all')
		WHERE name IN ('Admin')
		AND NOT ('conversations:read_team_all' = ANY(permissions));
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'view_visibility') THEN
				CREATE TYPE view_visibility AS ENUM ('all', 'team', 'user');
			END IF;
		END$$;
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		ALTER TABLE views
		ADD COLUMN IF NOT EXISTS visibility view_visibility NOT NULL DEFAULT 'user';
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		ALTER TABLE views
		ADD COLUMN IF NOT EXISTS team_id BIGINT REFERENCES teams(id) ON DELETE CASCADE ON UPDATE CASCADE NULL;
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		ALTER TABLE views
		ALTER COLUMN user_id DROP NOT NULL;
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint WHERE conname = 'constraint_views_visibility_user'
			) THEN
				ALTER TABLE views ADD CONSTRAINT constraint_views_visibility_user
					CHECK (visibility != 'user' OR user_id IS NOT NULL);
			END IF;
		END$$;
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint WHERE conname = 'constraint_views_visibility_team'
			) THEN
				ALTER TABLE views ADD CONSTRAINT constraint_views_visibility_team
					CHECK (visibility != 'team' OR team_id IS NOT NULL);
			END IF;
		END$$;
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS index_views_on_visibility ON views(visibility);
		CREATE INDEX IF NOT EXISTS index_views_on_team_id ON views(team_id);
	`)
	if err != nil {
		return err
	}

	// Add shared_views:manage permission to Admin role
	_, err = db.Exec(`
		UPDATE roles
		SET permissions = array_append(permissions, 'shared_views:manage')
		WHERE name = 'Admin'
		AND NOT ('shared_views:manage' = ANY(permissions));
	`)
	if err != nil {
		return err
	}

	// Add agent_password_set to activity_log_type enum.
	_, err = db.Exec(`ALTER TYPE activity_log_type ADD VALUE IF NOT EXISTS 'agent_password_set';`)
	if err != nil {
		return err
	}

	// Add agent_role_permissions_changed to activity_log_type enum.
	_, err = db.Exec(`ALTER TYPE activity_log_type ADD VALUE IF NOT EXISTS 'agent_role_permissions_changed';`)
	return err
}
