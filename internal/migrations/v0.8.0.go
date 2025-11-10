package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V0_8_0 updates the database schema to v0.8.0.
// This migration adds the organizations table and organization_id to users table.
func V0_8_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	// Create organizations table if it doesn't exist
	_, err := db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM information_schema.tables
				WHERE table_name = 'organizations'
			) THEN
				CREATE TABLE organizations (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					created_at TIMESTAMPTZ DEFAULT NOW(),
					updated_at TIMESTAMPTZ DEFAULT NOW(),
					"name" TEXT NOT NULL,
					website TEXT NULL,
					email_domain TEXT NULL,
					phone TEXT NULL,
					CONSTRAINT constraint_organizations_on_name CHECK (length("name") <= 255),
					CONSTRAINT constraint_organizations_on_website CHECK (length(website) <= 255),
					CONSTRAINT constraint_organizations_on_email_domain CHECK (length(email_domain) <= 255),
					CONSTRAINT constraint_organizations_on_phone CHECK (length(phone) <= 50)
				);

				CREATE INDEX index_organizations_on_name ON organizations USING btree ("name");
				CREATE INDEX index_organizations_on_email_domain ON organizations USING btree (email_domain);
			END IF;
		END $$;
	`)
	if err != nil {
		return err
	}

	// Add organization_id column to users table if it doesn't exist
	_, err = db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = 'users' AND column_name = 'organization_id'
			) THEN
				ALTER TABLE users
				ADD COLUMN organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL ON UPDATE CASCADE NULL;

				CREATE INDEX index_users_on_organization_id ON users(organization_id);
			END IF;
		END $$;
	`)
	if err != nil {
		return err
	}

	// Add organizations:manage permission to Admin role if not already present
	_, err = db.Exec(`
		UPDATE roles
		SET permissions = array_append(permissions, 'organizations:manage')
		WHERE name = 'Admin'
		AND NOT ('organizations:manage' = ANY(permissions));
	`)
	if err != nil {
		return err
	}

	return nil
}
