package migrations

import (
	"fmt"
	"log"
	"net/mail"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_9_0 adds globally unique ownership and verification state for inbox email addresses.
func V2_9_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`
		UPDATE roles
		SET permissions = array_append(permissions, 'messages:write_private')
		WHERE 'messages:write' = ANY(permissions)
		AND NOT ('messages:write_private' = ANY(permissions));

		CREATE TABLE IF NOT EXISTS inbox_email_addresses (
			id BIGSERIAL PRIMARY KEY,
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
		CREATE UNIQUE INDEX IF NOT EXISTS index_unique_inbox_email_addresses_on_email
			ON inbox_email_addresses (LOWER(email));
		CREATE UNIQUE INDEX IF NOT EXISTS index_unique_inbox_email_addresses_on_primary
			ON inbox_email_addresses (inbox_id) WHERE kind = 'primary';
		CREATE INDEX IF NOT EXISTS index_inbox_email_addresses_on_inbox_id
			ON inbox_email_addresses (inbox_id);
		ALTER TABLE inbox_email_addresses
			ADD COLUMN IF NOT EXISTS verification_status TEXT NOT NULL DEFAULT 'not_verified',
			ADD COLUMN IF NOT EXISTS verification_token TEXT NULL,
			ADD COLUMN IF NOT EXISTS verification_started_at TIMESTAMPTZ NULL,
			ADD COLUMN IF NOT EXISTS verified_at TIMESTAMPTZ NULL,
			DROP COLUMN IF EXISTS receive,
			DROP COLUMN IF EXISTS send;

		UPDATE inbox_email_addresses
		SET verification_status = 'verified'
		WHERE kind = 'primary';
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

	for _, inb := range inboxes {
		addr, err := mail.ParseAddress(inb.From)
		if err != nil || addr.Address == "" {
			log.Printf("WARNING: Skipping email inbox %d during address migration: invalid from address %q", inb.ID, inb.From)
			continue
		}
		email := strings.ToLower(strings.TrimSpace(addr.Address))
		if _, err := tx.Exec(`INSERT INTO inbox_email_addresses (inbox_id, email, kind, position, verification_status) VALUES ($1, $2, 'primary', 0, 'verified')`, inb.ID, email); err != nil {
			return fmt.Errorf("cannot migrate email inbox %d with address %q: the address is already owned by another inbox; make inbox From addresses unique and retry: %w", inb.ID, email, err)
		}
	}

	return tx.Commit()
}
