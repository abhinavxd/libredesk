package migrations

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
	"github.com/lib/pq"
)

const lastResolvedAtBatchSize = 10000

func V2_9_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`ALTER TYPE channels ADD VALUE IF NOT EXISTS 'whatsapp';`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`ALTER TABLE inboxes ADD COLUMN IF NOT EXISTS reopen_window_hours INT DEFAULT 0 NOT NULL;`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`ALTER TABLE conversations ADD COLUMN IF NOT EXISTS last_inbound_at TIMESTAMPTZ NULL;`)
	if err != nil {
		return err
	}

	if err := createIndexConcurrently(db, "index_conversations_on_last_inbound_at", `CREATE INDEX CONCURRENTLY IF NOT EXISTS index_conversations_on_last_inbound_at ON conversations (last_inbound_at)`); err != nil {
		return err
	}

	_, err = db.Exec(`ALTER TABLE conversations ADD COLUMN IF NOT EXISTS last_resolved_at TIMESTAMPTZ NULL;`)
	if err != nil {
		return err
	}

	if _, err := backfillLastResolvedAt(db, lastResolvedAtBatchSize); err != nil {
		return err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS whatsapp_templates (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
			updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
			inbox_id INT REFERENCES inboxes(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
			meta_template_id TEXT NULL,
			name TEXT NOT NULL,
			language TEXT NOT NULL,
			category TEXT NOT NULL,
			status TEXT DEFAULT 'PENDING' NOT NULL,
			header_type TEXT NULL,
			header_content TEXT NULL,
			body_content TEXT NOT NULL,
			footer_content TEXT NULL,
			buttons JSONB DEFAULT '[]'::jsonb NOT NULL,
			sample_values JSONB DEFAULT '{}'::jsonb NOT NULL,
			rejection_reason TEXT NULL,
			CONSTRAINT constraint_whatsapp_templates_on_name CHECK (length(name) <= 512),
			CONSTRAINT constraint_whatsapp_templates_on_language CHECK (length(language) <= 20),
			CONSTRAINT constraint_whatsapp_templates_on_category CHECK (length(category) <= 32),
			CONSTRAINT constraint_whatsapp_templates_on_status CHECK (length(status) <= 32),
			CONSTRAINT constraint_whatsapp_templates_on_header_type CHECK (length(header_type) <= 32)
		);
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS index_unique_whatsapp_templates_on_inbox_name_language ON whatsapp_templates (inbox_id, name, language);`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS index_whatsapp_templates_on_inbox_id ON whatsapp_templates (inbox_id);`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS index_whatsapp_templates_on_meta_template_id ON whatsapp_templates (meta_template_id);`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS contact_channel_identities (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			contact_id BIGINT REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
			channel channels NOT NULL,
			identifier TEXT NOT NULL,
			CONSTRAINT constraint_contact_channel_identities_on_identifier CHECK (length(identifier) <= 1000)
		);
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS index_unique_contact_channel_identities_on_channel_identifier ON contact_channel_identities (channel, identifier);`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS index_contact_channel_identities_on_contact_id ON contact_channel_identities (contact_id);`)
	if err != nil {
		return err
	}

	if err := createIndexConcurrently(db, "index_tgrm_users_on_phone_number", `CREATE INDEX CONCURRENTLY IF NOT EXISTS index_tgrm_users_on_phone_number ON users USING GIN (phone_number gin_trgm_ops)`); err != nil {
		return err
	}

	if err := createIndexConcurrently(db, "index_conversation_messages_on_source_id", `CREATE INDEX CONCURRENTLY IF NOT EXISTS index_conversation_messages_on_source_id ON conversation_messages (source_id)`); err != nil {
		return err
	}
	return nil
}

func backfillLastResolvedAt(db *sqlx.DB, batchSize int) (int, error) {
	batches := 0
	for {
		res, err := db.Exec(`
			UPDATE conversations SET last_resolved_at = resolved_at
			WHERE id IN (
				SELECT id FROM conversations
				WHERE resolved_at IS NOT NULL AND last_resolved_at IS NULL
				LIMIT $1
			);`, batchSize)
		if err != nil {
			return batches, err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return batches, err
		}
		if n == 0 {
			return batches, nil
		}
		batches++
	}
}

// createIndexConcurrently drops an invalid leftover first, which IF NOT EXISTS would otherwise accept.
func createIndexConcurrently(db *sqlx.DB, name, ddl string) error {
	var invalid bool
	err := db.Get(&invalid, `SELECT NOT indisvalid FROM pg_index WHERE indexrelid = to_regclass($1)`, name)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if err == nil && invalid {
		if _, err := db.Exec(`DROP INDEX IF EXISTS ` + pq.QuoteIdentifier(name)); err != nil {
			return err
		}
	}
	_, err = db.Exec(ddl)
	return err
}
