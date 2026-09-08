package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_9_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`
		UPDATE roles
		SET permissions = array_append(permissions, 'messages:write_private')
		WHERE 'messages:write' = ANY(permissions)
		AND NOT ('messages:write_private' = ANY(permissions));
	`); err != nil {
		return err
	}

	// Grant the new subject permission to every role that can already change a conversation's
	// status, so existing Admin and Agent roles keep parity with a fresh install.
	_, err := db.Exec(`
		UPDATE roles
		SET permissions = array_append(permissions, 'conversations:update_subject')
		WHERE 'conversations:update_status' = ANY(permissions)
		AND NOT ('conversations:update_subject' = ANY(permissions));
	`)
	return err
}
