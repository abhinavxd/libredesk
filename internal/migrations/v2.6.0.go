package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_6_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	for _, permission := range []string{"contacts:delete", "contacts:export"} {
		if _, err := db.Exec(`
			UPDATE roles
			SET permissions = array_append(permissions, $1)
			WHERE name = 'Admin' AND NOT ($1 = ANY(permissions));
		`, permission); err != nil {
			return err
		}
	}

	if _, err := db.Exec(`ALTER TYPE activity_log_type ADD VALUE IF NOT EXISTS 'contact_deleted';`); err != nil {
		return err
	}
	if _, err := db.Exec(`ALTER TYPE activity_log_type ADD VALUE IF NOT EXISTS 'contact_data_exported';`); err != nil {
		return err
	}
	return nil
}
