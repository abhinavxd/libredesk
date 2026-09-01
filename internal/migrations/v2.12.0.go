package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_12_0 adds helpdesk organizations and contact membership.
func V2_12_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS organizations (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			name TEXT NOT NULL,
			domains TEXT[] NOT NULL DEFAULT '{}',
			notes TEXT NOT NULL DEFAULT '',
			external_id TEXT NULL,
			custom_attributes JSONB DEFAULT '{}'::jsonb NOT NULL,
			CONSTRAINT organizations_name_len CHECK (length(name) <= 255),
			CONSTRAINT organizations_external_id_unique UNIQUE (external_id)
		);
		ALTER TABLE users
			ADD COLUMN IF NOT EXISTS organization_id INT REFERENCES organizations(id) ON DELETE SET NULL ON UPDATE CASCADE;
		CREATE INDEX IF NOT EXISTS index_users_on_organization_id ON users (organization_id);
		UPDATE roles
		SET permissions = array_append(permissions, 'organizations:manage')
		WHERE name = 'Admin' AND NOT ('organizations:manage' = ANY(permissions));
	`)
	return err
}
