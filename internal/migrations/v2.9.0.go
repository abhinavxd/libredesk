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

	// Deleting a conversation is irreversible and can take the originating mails off the
	// mail server with it, so only the built-in Admin role is granted it on upgrade.
	if _, err := db.Exec(`
		UPDATE roles
		SET permissions = array_append(permissions, 'conversations:delete')
		WHERE name = 'Admin' AND NOT ('conversations:delete' = ANY(permissions));
	`); err != nil {
		return err
	}

	return nil
}
