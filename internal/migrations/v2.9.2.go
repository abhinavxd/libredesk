package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_9_2(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	// Deleting a conversation is irreversible and can take the originating mails off the
	// mail server with it, so only the built-in Admin role is granted it on upgrade.
	_, err := db.Exec(`
		UPDATE roles
		SET permissions = array_append(permissions, 'conversations:delete')
		WHERE name = 'Admin' AND NOT ('conversations:delete' = ANY(permissions));
	`)
	return err
}
