package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V0_8_0 updates the database schema to v0.8.0.
func V0_8_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	// Add 'livechat' to the channels enum if not already present
	var exists bool
	err := db.Get(&exists, `
		SELECT EXISTS (
			SELECT 1
			FROM pg_enum
			WHERE enumlabel = 'livechat'
			AND enumtypid = (
				SELECT oid FROM pg_type WHERE typname = 'channels'
			)
		)
	`)
	if err != nil {
		return err
	}
	if !exists {
		_, err = db.Exec(`ALTER TYPE channels ADD VALUE 'livechat'`)
		if err != nil {
			return err
		}
	}

	// Drop the foreign key constraint and column from conversations table first
	_, err = db.Exec(`
		ALTER TABLE conversations DROP CONSTRAINT IF EXISTS conversations_contact_channel_id_fkey;
	`)
	if err != nil {
		return err
	}

	// Drop the contact_channel_id column from conversations table
	_, err = db.Exec(`
		ALTER TABLE conversations DROP COLUMN IF EXISTS contact_channel_id;
	`)
	if err != nil {
		return err
	}

	// Drop contact_channels table
	_, err = db.Exec(`
		DROP TABLE IF EXISTS contact_channels CASCADE;
	`)
	if err != nil {
		return err
	}

	// Add contact_last_seen_at column if it doesn't exist
	_, err = db.Exec(`
		ALTER TABLE conversations ADD COLUMN IF NOT EXISTS contact_last_seen_at TIMESTAMPTZ DEFAULT NOW();
	`)
	if err != nil {
		return err
	}

	// Add last_interaction_at column if it doesn't exist
	_, err = db.Exec(`
		ALTER TABLE conversations ADD COLUMN IF NOT EXISTS last_interaction_at TIMESTAMPTZ NULL;
	`)
	if err != nil {
		return err
	}

	// Create index on last_interaction_at column if it doesn't exist
	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS index_conversations_on_last_interaction_at ON conversations (last_interaction_at);
	`)
	if err != nil {
		return err
	}

	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmts := []string{
		/* ── drop index for e‑mail uniqueness and add seperate indexes for type of user ── */
		`DROP INDEX IF EXISTS index_unique_users_on_email_and_type_when_deleted_at_is_null`,

		/* ── email for agents are unique ── */
		`CREATE UNIQUE INDEX IF NOT EXISTS
		index_unique_users_on_email_when_type_is_agent
			ON users(email)
			WHERE type = 'agent'  AND deleted_at IS NULL`,
	}

	for _, q := range stmts {
		if _, err = tx.Exec(q); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	tx2, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx2.Rollback()

	jwtStmts := []string{
		/* ── Add secret column to inboxes table for JWT signing (livechat only) ── */
		`ALTER TABLE inboxes ADD COLUMN IF NOT EXISTS secret TEXT NULL`,

		/* ── Add external_user_id column to users table for 3rd party user mapping ── */
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS external_user_id TEXT NULL`,

		/* ──  ── */
		`
		CREATE UNIQUE INDEX IF NOT EXISTS index_unique_users_on_ext_id_when_type_is_contact 
		ON users (external_user_id) 
		WHERE type = 'contact' AND deleted_at IS NULL AND external_user_id IS NOT NULL;
		`,

		`
		CREATE UNIQUE INDEX IF NOT EXISTS index_unique_users_on_email_when_no_ext_id_contact
		ON users (email) 
		WHERE type = 'contact' AND deleted_at IS NULL AND external_user_id IS NULL;
		`,
	}

	for _, q := range jwtStmts {
		if _, err = tx2.Exec(q); err != nil {
			return err
		}
	}

	if err := tx2.Commit(); err != nil {
		return err
	}

	// Add index on conversation_messages for conversation_id and created_at
	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS index_conversation_messages_on_conversation_id_and_created_at
		ON conversation_messages (conversation_id, created_at);
	`)
	if err != nil {
		return err
	}

	// Add inbox linking support for conversation continuity between chat and email
	_, err = db.Exec(`
		ALTER TABLE inboxes ADD COLUMN IF NOT EXISTS linked_email_inbox_id INT REFERENCES inboxes(id) ON DELETE SET NULL;
	`)
	if err != nil {
		return err
	}

	return nil
}
