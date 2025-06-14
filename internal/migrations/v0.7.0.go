package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V0_7_0 updates the database schema to v0.7.0.
func V0_7_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	// Create webhook_event enum type if it doesn't exist
	_, err := db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_type WHERE typname = 'webhook_event'
			) THEN
				CREATE TYPE webhook_event AS ENUM (
					'conversation.created',
					'conversation.status_changed',
					'conversation.tags_changed',
					'conversation.assigned',
					'conversation.unassigned',
					'message.created',
					'message.updated'
				);
			END IF;
		END
		$$;
	`)
	if err != nil {
		return err
	}

	// Create webhooks table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS webhooks (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			name TEXT NOT NULL,
			url TEXT NOT NULL,
			events webhook_event[] NOT NULL DEFAULT '{}',
			secret TEXT DEFAULT '',
			is_active BOOLEAN DEFAULT true,
			headers JSONB DEFAULT '{}',
			CONSTRAINT constraint_webhooks_on_name CHECK (length(name) <= 255),
			CONSTRAINT constraint_webhooks_on_url CHECK (length(url) <= 2048),
			CONSTRAINT constraint_webhooks_on_secret CHECK (length(secret) <= 255),
			CONSTRAINT constraint_webhooks_on_events_not_empty CHECK (array_length(events, 1) > 0)
		);
		CREATE INDEX IF NOT EXISTS index_webhooks_on_is_active ON webhooks (is_active);
	`)
	if err != nil {
		return err
	}

	// Add webhooks:manage permission to Admin role
	_, err = db.Exec(`
		UPDATE roles
		SET permissions = array_append(permissions, 'webhooks:manage')
		WHERE name = 'Admin' AND NOT ('webhooks:manage' = ANY(permissions));
	`)
	if err != nil {
		return err
	}

	return nil
}
