package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_8_2 restores custom-attribute management on legacy Admin roles.
func V2_8_2(db *sqlx.DB, _ stuffbin.FileSystem, _ *koanf.Koanf) error {
	_, err := db.Exec(`
		UPDATE roles
		SET permissions = array_append(permissions, 'custom_attributes:manage')
		WHERE name = 'Admin'
		  AND NOT ('custom_attributes:manage' = ANY(permissions));
	`)
	return err
}
