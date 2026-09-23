package migrations

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
	"github.com/lib/pq"
)

const lastResolvedAtBatchSize = 10000

var notificationTypesV2_9_0 = []string{
	"new_reply",
	"new_reply_participating",
	"sla_first_response_warning",
	"sla_first_response_breach",
	"sla_next_response_warning",
	"sla_next_response_breach",
	"sla_resolution_warning",
	"sla_resolution_breach",
	"conversation_reopened",
	"automation",
}

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
			component_types TEXT[] NULL,
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

	_, err = db.Exec(`ALTER TABLE whatsapp_templates ADD COLUMN IF NOT EXISTS component_types TEXT[] NULL;`)
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

	if _, err := db.Exec(`ALTER TABLE ai_tools ADD COLUMN IF NOT EXISTS copilot_enabled BOOLEAN NOT NULL DEFAULT false;`); err != nil {
		return err
	}
	if _, err := db.Exec(`ALTER TABLE ai_tools ADD COLUMN IF NOT EXISTS generate_reply_enabled BOOLEAN NOT NULL DEFAULT false;`); err != nil {
		return err
	}
	if _, err := db.Exec(`ALTER TABLE ai_tools ADD COLUMN IF NOT EXISTS requires_agent_approval BOOLEAN NOT NULL DEFAULT true;`); err != nil {
		return err
	}
	if _, err := db.Exec(`
		UPDATE roles
		SET permissions = array_append(permissions, 'messages:write_private')
		WHERE 'messages:write' = ANY(permissions)
		AND NOT ('messages:write_private' = ANY(permissions));
	`); err != nil {
		return err
	}

	for _, notificationType := range notificationTypesV2_9_0 {
		if _, err := db.Exec(`ALTER TYPE user_notification_type ADD VALUE IF NOT EXISTS '` + notificationType + `';`); err != nil {
			return err
		}
	}

	if _, err := db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'notification_channel') THEN
				CREATE TYPE notification_channel AS ENUM ('in_app', 'email', 'push');
			END IF;
		END$$;
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`ALTER TYPE notification_channel ADD VALUE IF NOT EXISTS 'push';`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS user_notification_preferences (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			user_id BIGINT REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
			notification_type user_notification_type NOT NULL,
			channel notification_channel NOT NULL,
			enabled BOOLEAN NOT NULL DEFAULT TRUE,
			CONSTRAINT constraint_uniq_user_notification_preferences UNIQUE (user_id, notification_type, channel)
		);
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS notification_push_subscriptions (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			user_id BIGINT REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
			endpoint TEXT NOT NULL UNIQUE,
			p256dh TEXT NOT NULL,
			auth TEXT NOT NULL
		);
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS index_notification_push_subscriptions_on_user_id ON notification_push_subscriptions(user_id);
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS notification_email_queue (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			user_id BIGINT REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
			notification_id BIGINT REFERENCES user_notifications(id) ON DELETE CASCADE ON UPDATE CASCADE,
			notification_type user_notification_type NOT NULL,
			conversation_id BIGINT REFERENCES conversations(id) ON DELETE CASCADE ON UPDATE CASCADE,
			recipient_email TEXT NOT NULL,
			subject TEXT NOT NULL,
			content TEXT NOT NULL,
			queued_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			send_at TIMESTAMPTZ NOT NULL,
			attempts INTEGER NOT NULL DEFAULT 0,
			CONSTRAINT constraint_uniq_notification_email_queue UNIQUE (user_id, notification_type, conversation_id)
		);
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS index_notification_email_queue_on_send_at ON notification_email_queue(send_at);
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`
		ALTER TABLE notification_email_queue ADD COLUMN IF NOT EXISTS attempts INTEGER NOT NULL DEFAULT 0;
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		INSERT INTO settings ("key", value) VALUES
			('notification.push.vapid_public_key', '""'::jsonb),
			('notification.push.vapid_private_key', '""'::jsonb)
		ON CONFLICT ("key") DO NOTHING;
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		INSERT INTO templates ("type", body, is_default, "name", subject, is_builtin)
		SELECT 'email_notification'::template_type, '
<p>{{ .Author.FullName }} replied to a conversation assigned to you:</p>

<div>
    Reference number: {{ .Conversation.ReferenceNumber }} <br>
    Subject: {{ .Conversation.Subject }}
</div>

<blockquote style="background-color: #f5f5f5; padding: 12px; margin: 16px 0; border-left: 4px solid #ddd;">
{{ .Message.Content }}
</blockquote>

<p>
    <a href="{{ RootURL }}/inboxes/assigned/conversation/{{ .Conversation.UUID }}">View Conversation</a>
</p>

<div>
    Best regards,<br>
    Libredesk
</div>

', false, 'New reply from contact', 'New reply on conversation #{{ .Conversation.ReferenceNumber }}', true
		WHERE NOT EXISTS (SELECT 1 FROM templates WHERE "name" = 'New reply from contact');
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		INSERT INTO templates ("type", body, is_default, "name", subject, is_builtin)
		SELECT 'email_notification'::template_type, '
<p>{{ .Author.FullName }} replied to a conversation you are participating in:</p>

<div>
    Reference number: {{ .Conversation.ReferenceNumber }} <br>
    Subject: {{ .Conversation.Subject }}
</div>

<blockquote style="background-color: #f5f5f5; padding: 12px; margin: 16px 0; border-left: 4px solid #ddd;">
{{ .Message.Content }}
</blockquote>

<p>
    <a href="{{ RootURL }}/inboxes/assigned/conversation/{{ .Conversation.UUID }}">View Conversation</a>
</p>

<div>
    Best regards,<br>
    Libredesk
</div>

', false, 'New reply on participating conversation', 'New reply on conversation #{{ .Conversation.ReferenceNumber }}', true
		WHERE NOT EXISTS (SELECT 1 FROM templates WHERE "name" = 'New reply on participating conversation');
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		INSERT INTO templates ("type", body, is_default, "name", subject, is_builtin)
		SELECT 'email_notification'::template_type, '
<p>{{ .Author.FullName }} replied and reopened a conversation assigned to you:</p>

<div>
    Reference number: {{ .Conversation.ReferenceNumber }} <br>
    Subject: {{ .Conversation.Subject }}
</div>

<blockquote style="background-color: #f5f5f5; padding: 12px; margin: 16px 0; border-left: 4px solid #ddd;">
{{ .Message.Content }}
</blockquote>

<p>
    <a href="{{ RootURL }}/inboxes/assigned/conversation/{{ .Conversation.UUID }}">View Conversation</a>
</p>

<div>
    Best regards,<br>
    Libredesk
</div>

', false, 'Conversation reopened', 'Conversation #{{ .Conversation.ReferenceNumber }} reopened', true
		WHERE NOT EXISTS (SELECT 1 FROM templates WHERE "name" = 'Conversation reopened');
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS widget_campaign_deliveries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id UUID NOT NULL,
    inbox_id INTEGER NOT NULL REFERENCES inboxes(id) ON DELETE CASCADE,
    browser_key UUID NOT NULL,
    session_key UUID NOT NULL,
    contact_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    snapshot JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    displayed BOOLEAN NOT NULL DEFAULT FALSE,
    opened BOOLEAN NOT NULL DEFAULT FALSE,
    dismissed BOOLEAN NOT NULL DEFAULT FALSE,
    replied BOOLEAN NOT NULL DEFAULT FALSE,
    conversation_uuid UUID REFERENCES conversations(uuid) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_widget_campaign_browser ON widget_campaign_deliveries(inbox_id, browser_key, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_widget_campaign_contact ON widget_campaign_deliveries(inbox_id, contact_id, created_at DESC) WHERE contact_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_widget_campaign_stats ON widget_campaign_deliveries(inbox_id, campaign_id, created_at);

ALTER TABLE help_centers ADD COLUMN IF NOT EXISTS livechat_inbox_id INTEGER NULL REFERENCES inboxes(id) ON DELETE SET NULL;

ALTER TABLE help_articles ADD COLUMN IF NOT EXISTS translation_group_id UUID NOT NULL DEFAULT gen_random_uuid();
CREATE UNIQUE INDEX IF NOT EXISTS index_unique_help_articles_on_translation_group_locale
    ON help_articles(translation_group_id, locale);
`); err != nil {
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
