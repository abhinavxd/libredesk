package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V1_0_4 adds the integrations table and integrations:manage permission to the Admin role.
func V1_0_4(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS integrations (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			provider TEXT NOT NULL UNIQUE,
			config JSONB DEFAULT '{}'::jsonb NOT NULL,
			enabled BOOLEAN DEFAULT true NOT NULL,
			CONSTRAINT constraint_integrations_on_provider CHECK (length(provider) <= 100)
		);

		UPDATE roles
		SET permissions = array_append(permissions, 'integrations:manage')
		WHERE name = 'Admin'
		AND NOT ('integrations:manage' = ANY(permissions));
	`)
	return err
}
