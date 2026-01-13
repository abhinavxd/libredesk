package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V0_10_0 updates the database schema to v0.10.0.
func V0_10_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		ALTER TABLE conversations
		ADD COLUMN IF NOT EXISTS last_interaction TEXT NULL,
		ADD COLUMN IF NOT EXISTS last_interaction_sender message_sender_type NULL,
		ADD COLUMN IF NOT EXISTS last_interaction_at TIMESTAMPTZ NULL;

		CREATE INDEX IF NOT EXISTS index_conversations_on_last_interaction_at
		ON conversations(last_interaction_at);
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS conversation_mentions (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			conversation_id BIGINT REFERENCES conversations(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
			message_id BIGINT REFERENCES conversation_messages(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
			mentioned_user_id BIGINT REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
			mentioned_team_id INT REFERENCES teams(id) ON DELETE CASCADE ON UPDATE CASCADE,
			mentioned_by_user_id BIGINT REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
			CONSTRAINT constraint_mention_target CHECK (
				(mentioned_user_id IS NOT NULL AND mentioned_team_id IS NULL) OR
				(mentioned_user_id IS NULL AND mentioned_team_id IS NOT NULL)
			)
		);

		CREATE INDEX IF NOT EXISTS index_conversation_mentions_on_mentioned_user_id ON conversation_mentions(mentioned_user_id);
		CREATE INDEX IF NOT EXISTS index_conversation_mentions_on_mentioned_team_id ON conversation_mentions(mentioned_team_id);
		CREATE INDEX IF NOT EXISTS index_conversation_mentions_on_conversation_id ON conversation_mentions(conversation_id);
	`)
	if err != nil {
		return err
	}

	// Add email notification template for mentions
	_, err = db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM templates WHERE "name" = 'Mentioned in conversation') THEN
				INSERT INTO templates
					("type", body, is_default, "name", subject, is_builtin)
					VALUES (
					'email_notification'::template_type,
'<p>{{ .MentionedBy.FullName }} mentioned you in a private note on conversation #{{ .Conversation.ReferenceNumber }}.</p>

<blockquote style="background-color: #f5f5f5; padding: 12px; margin: 16px 0; border-left: 4px solid #ddd;">
{{ .Message.Content }}
</blockquote>

<p>
<a href="{{ RootURL }}/inboxes/mentioned/conversation/{{ .Conversation.UUID }}?scrollTo={{ .Message.UUID }}">View Conversation</a>
</p>

<p>
Best regards,<br>
Libredesk
</p>',
					false,
					'Mentioned in conversation',
					'{{ .MentionedBy.FullName }} mentioned you in conversation #{{ .Conversation.ReferenceNumber }}',
					true
				);
			END IF;
		END$$;
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_notification_type') THEN
				CREATE TYPE user_notification_type AS ENUM ('mention', 'assignment', 'sla_warning', 'sla_breach');
			END IF;
		END$$;
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS user_notifications (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			user_id BIGINT REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
			notification_type user_notification_type NOT NULL,
			title TEXT NOT NULL,
			body TEXT NULL,
			is_read BOOLEAN DEFAULT FALSE NOT NULL,
			conversation_id BIGINT REFERENCES conversations(id) ON DELETE CASCADE ON UPDATE CASCADE,
			message_id BIGINT REFERENCES conversation_messages(id) ON DELETE CASCADE ON UPDATE CASCADE,
			actor_id BIGINT REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE,
			meta JSONB DEFAULT '{}'::jsonb NOT NULL,
			CONSTRAINT constraint_user_notifications_on_title CHECK (length(title) <= 500),
			CONSTRAINT constraint_user_notifications_on_body CHECK (length(body) <= 2000)
		);

		CREATE INDEX IF NOT EXISTS index_user_notifications_on_user_id ON user_notifications(user_id);
		CREATE INDEX IF NOT EXISTS index_user_notifications_on_user_id_is_read ON user_notifications(user_id, is_read);
		CREATE INDEX IF NOT EXISTS index_user_notifications_on_created_at ON user_notifications(created_at);
		CREATE INDEX IF NOT EXISTS index_user_notifications_on_conversation_id ON user_notifications(conversation_id);
	`)
	if err != nil {
		return err
	}

	// Add logo_url column to oidc table for custom provider logos
	_, err = db.Exec(`
		ALTER TABLE oidc ADD COLUMN IF NOT EXISTS logo_url TEXT NOT NULL DEFAULT '';
	`)
	if err != nil {
		return err
	}

	return nil
}
