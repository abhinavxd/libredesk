package migrations

import (
	"log"
	"net/mail"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_9_0_RC9 adds the inbox email addresses table with alias verification state.
func V2_9_0_RC9(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS inbox_email_addresses (
			id SERIAL PRIMARY KEY,
			inbox_id INTEGER NOT NULL REFERENCES inboxes(id) ON DELETE CASCADE,
			email TEXT NOT NULL,
			kind TEXT NOT NULL,
			position INTEGER NOT NULL DEFAULT 0,
			verification_status TEXT NOT NULL DEFAULT 'not_verified',
			verification_token TEXT NULL,
			verification_started_at TIMESTAMPTZ NULL,
			verified_at TIMESTAMPTZ NULL,
			CONSTRAINT constraint_inbox_email_addresses_on_kind CHECK (kind IN ('primary', 'alias'))
		);
		CREATE UNIQUE INDEX IF NOT EXISTS index_unique_inbox_email_addresses_on_inbox_email
			ON inbox_email_addresses (inbox_id, LOWER(email));
		CREATE UNIQUE INDEX IF NOT EXISTS index_unique_inbox_email_addresses_on_primary
			ON inbox_email_addresses (inbox_id) WHERE kind = 'primary';
	`); err != nil {
		return err
	}

	type inboxAddress struct {
		ID   int    `db:"id"`
		From string `db:"from"`
	}
	var inboxes []inboxAddress
	if err := tx.Select(&inboxes, `
		SELECT id, COALESCE("from", '') AS "from"
		FROM inboxes
		WHERE channel = 'email' AND deleted_at IS NULL
		  AND NOT EXISTS (
			SELECT 1 FROM inbox_email_addresses a
			WHERE a.inbox_id = inboxes.id AND a.kind = 'primary'
		  )
		ORDER BY id
	`); err != nil {
		return err
	}

	// Register each existing email inbox's From address as its primary address.
	for _, inb := range inboxes {
		addr, err := mail.ParseAddress(inb.From)
		if err != nil || addr.Address == "" {
			log.Printf("WARNING: Skipping email inbox %d during address migration: invalid from address %q", inb.ID, inb.From)
			continue
		}
		email := strings.ToLower(strings.TrimSpace(addr.Address))
		if _, err := tx.Exec(`INSERT INTO inbox_email_addresses (inbox_id, email, kind, position, verification_status) VALUES ($1, $2, 'primary', 0, 'verified')`, inb.ID, email); err != nil {
			return err
		}
	}

	return tx.Commit()
}
