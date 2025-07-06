package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V0_8_0 updates the database schema to v0.8.0.
func V0_8_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	// Add 'livechat' to the channels enum
	_, err := db.Exec(`
		ALTER TYPE channels ADD VALUE IF NOT EXISTS 'livechat';
	`)
	if err != nil {
		return err
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

	return nil
}
