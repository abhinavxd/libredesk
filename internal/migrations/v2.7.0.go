package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_7_0 adds the permissionless customer portal role and backfills contacts.
func V2_7_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`
		INSERT INTO roles (name, description, permissions)
		VALUES ('User', 'Customer portal user', ARRAY[]::TEXT[])
		ON CONFLICT (name) DO NOTHING;
	`); err != nil {
		return err
	}

	_, err := db.Exec(`
		INSERT INTO user_roles (user_id, role_id)
		SELECT users.id, roles.id
		FROM users
		CROSS JOIN roles
		WHERE users.type = 'contact'
		  AND users.deleted_at IS NULL
		  AND roles.name = 'User'
		ON CONFLICT (user_id, role_id) DO NOTHING;
	`)
	return err
}
