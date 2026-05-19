package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_0_0 updates the database schema to v2.0.0 (Live Chat feature).
func V2_0_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`ALTER TYPE channels ADD VALUE IF NOT EXISTS 'livechat'`)
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

	tx2, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx2.Rollback()

	stmts := []string{
		`ALTER TABLE inboxes ADD COLUMN IF NOT EXISTS secret TEXT NULL`,

		`ALTER TABLE users ADD COLUMN IF NOT EXISTS external_user_id TEXT NULL`,

		`DROP INDEX IF EXISTS index_unique_users_on_email_and_type_when_deleted_at_is_null`,

		`
		CREATE UNIQUE INDEX IF NOT EXISTS index_unique_users_on_email_when_type_is_agent
		ON users (email)
		WHERE type = 'agent' AND deleted_at IS NULL;
		`,

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

	for _, q := range stmts {
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

	// Add column to track last continuity email sent
	_, err = db.Exec(`
		ALTER TABLE conversations ADD COLUMN IF NOT EXISTS last_continuity_email_sent_at TIMESTAMPTZ NULL;
	`)
	if err != nil {
		return err
	}

	// Add index for continuity email tracking
	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS index_conversations_on_last_continuity_email_sent_at
		ON conversations (last_continuity_email_sent_at);
	`)
	if err != nil {
		return err
	}

	// Add last_message_sender_id column to track who sent the last message
	_, err = db.Exec(`
		ALTER TABLE conversations ADD COLUMN IF NOT EXISTS last_message_sender_id BIGINT REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE;
	`)
	if err != nil {
		return err
	}

	// Add last_interaction_sender_id column to track who sent the last interaction (for widget display)
	_, err = db.Exec(`
		ALTER TABLE conversations ADD COLUMN IF NOT EXISTS last_interaction_sender_id BIGINT REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE;
	`)
	if err != nil {
		return err
	}

	// Add 'visitor' to user_type enum
	_, err = db.Exec(`ALTER TYPE user_type ADD VALUE IF NOT EXISTS 'visitor'`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		ALTER TABLE teams DROP CONSTRAINT IF EXISTS constraint_teams_on_emoji;
		ALTER TABLE teams ADD CONSTRAINT constraint_teams_on_emoji CHECK (length(emoji) <= 50);
	`)
	if err != nil {
		return err
	}

	// Add context_links:manage permission to Admin role.
	_, err = db.Exec(`
		UPDATE roles
		SET permissions = array_append(permissions, 'context_links:manage')
		WHERE name = 'Admin' AND NOT ('context_links:manage' = ANY(permissions));
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		ALTER TABLE inboxes ADD COLUMN IF NOT EXISTS uuid UUID DEFAULT gen_random_uuid() NOT NULL UNIQUE;
	`)
	if err != nil {
		return err
	}

	// Add context_links table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS context_links (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			name TEXT NOT NULL,
			url_template TEXT NOT NULL,
			signing_secret TEXT NOT NULL DEFAULT '',
			token_expiry_seconds INT NOT NULL DEFAULT 1200,
			is_active BOOLEAN DEFAULT true,
			CONSTRAINT constraint_context_links_on_name CHECK (length(name) <= 255),
			CONSTRAINT constraint_context_links_on_url_template CHECK (length(url_template) <= 2048),
			CONSTRAINT constraint_context_links_on_signing_secret CHECK (length(signing_secret) <= 500)
		);
	`)
	if err != nil {
		return err
	}

	// Add meta column to csat_responses.
	_, err = db.Exec(`ALTER TABLE csat_responses ADD COLUMN IF NOT EXISTS meta JSONB DEFAULT '{}' NOT NULL;`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS index_csat_responses_on_conversation_id
		ON csat_responses (conversation_id);
	`)
	if err != nil {
		return err
	}

	// Add built-in CSAT request email template.
	_, err = db.Exec(`
		INSERT INTO templates ("type", body, is_default, "name", subject, is_builtin)
		SELECT
			'email_notification'::template_type,
			'
<p style="margin: 0 0 4px; font-size: 15px; color: #374151; text-align: center; line-height: 1.5;">
  Your conversation <strong style="color: #111827;">#{{ .Conversation.ReferenceNumber }}</strong> has been resolved.
</p>
<p style="margin: 0 0 28px; font-size: 13px; color: #9ca3af; text-align: center;">
  We would love to hear how it went.
</p>
<p style="margin: 0 0 20px; font-size: 14px; font-weight: 600; color: #374151; text-align: center;">
  How would you rate your experience?
</p>
<!-- Variable CSATUUID is also available -->
<div style="text-align: center; margin: 0 auto; max-width: 400px; font-size: 0;">
  <div style="display: inline-block; width: 72px; text-align: center; vertical-align: top; padding: 4px 0;">
    <a href="{{ .CSATLink }}?rating=1" style="text-decoration: none; display: block;">
      <span style="font-size: 34px; display: block; line-height: 1.4;">&#128546;</span>
      <span style="font-size: 10px; display: block; font-weight: 600; color: #b0b5bd; text-transform: uppercase; letter-spacing: 0.05em; margin-top: 4px;">Poor</span>
    </a>
  </div>
  <div style="display: inline-block; width: 72px; text-align: center; vertical-align: top; padding: 4px 0;">
    <a href="{{ .CSATLink }}?rating=2" style="text-decoration: none; display: block;">
      <span style="font-size: 34px; display: block; line-height: 1.4;">&#128533;</span>
      <span style="font-size: 10px; display: block; font-weight: 600; color: #b0b5bd; text-transform: uppercase; letter-spacing: 0.05em; margin-top: 4px;">Fair</span>
    </a>
  </div>
  <div style="display: inline-block; width: 72px; text-align: center; vertical-align: top; padding: 4px 0;">
    <a href="{{ .CSATLink }}?rating=3" style="text-decoration: none; display: block;">
      <span style="font-size: 34px; display: block; line-height: 1.4;">&#128522;</span>
      <span style="font-size: 10px; display: block; font-weight: 600; color: #b0b5bd; text-transform: uppercase; letter-spacing: 0.05em; margin-top: 4px;">Good</span>
    </a>
  </div>
  <div style="display: inline-block; width: 72px; text-align: center; vertical-align: top; padding: 4px 0;">
    <a href="{{ .CSATLink }}?rating=4" style="text-decoration: none; display: block;">
      <span style="font-size: 34px; display: block; line-height: 1.4;">&#128515;</span>
      <span style="font-size: 10px; display: block; font-weight: 600; color: #b0b5bd; text-transform: uppercase; letter-spacing: 0.05em; margin-top: 4px;">Great</span>
    </a>
  </div>
  <div style="display: inline-block; width: 72px; text-align: center; vertical-align: top; padding: 4px 0;">
    <a href="{{ .CSATLink }}?rating=5" style="text-decoration: none; display: block;">
      <span style="font-size: 34px; display: block; line-height: 1.4;">&#129321;</span>
      <span style="font-size: 10px; display: block; font-weight: 600; color: #b0b5bd; text-transform: uppercase; letter-spacing: 0.05em; margin-top: 4px;">Excellent</span>
    </a>
  </div>
</div>
',
			false,
			'CSAT request',
			'',
			true
		WHERE NOT EXISTS (
			SELECT 1 FROM templates WHERE name = 'CSAT request' AND is_builtin = true
		);
	`)
	if err != nil {
		return err
	}

	return nil
}
