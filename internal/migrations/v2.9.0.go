package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_9_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		UPDATE roles
		SET permissions = array_append(permissions, 'messages:write_private')
		WHERE 'messages:write' = ANY(permissions)
		AND NOT ('messages:write_private' = ANY(permissions));
	`)
	return err
}
