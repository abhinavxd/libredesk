package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

var notificationTypesV2_10_0 = []string{
	"new_reply",
	"new_reply_participating",
	"sla_first_response_warning",
	"sla_first_response_breach",
	"sla_next_response_warning",
	"sla_next_response_breach",
	"sla_resolution_warning",
	"sla_resolution_breach",
	"conversation_reopened",
}

func V2_10_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	for _, notificationType := range notificationTypesV2_10_0 {
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

	_, err := db.Exec(`
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

		CREATE TABLE IF NOT EXISTS notification_push_subscriptions (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			user_id BIGINT REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
			endpoint TEXT NOT NULL UNIQUE,
			p256dh TEXT NOT NULL,
			auth TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS index_notification_push_subscriptions_on_user_id ON notification_push_subscriptions(user_id);

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
			CONSTRAINT constraint_uniq_notification_email_queue UNIQUE (user_id, notification_type, conversation_id)
		);
		CREATE INDEX IF NOT EXISTS index_notification_email_queue_on_send_at ON notification_email_queue(send_at);

		INSERT INTO settings ("key", value) VALUES
			('notification.push.vapid_public_key', '""'::jsonb),
			('notification.push.vapid_private_key', '""'::jsonb)
		ON CONFLICT ("key") DO NOTHING;

		INSERT INTO templates ("type", body, is_default, "name", subject, is_builtin)
		SELECT 'email_notification'::template_type, '
<p>{{ .Author.FullName }} replied to a conversation assigned to you:</p>

<div>
    Reference number: {{ .Conversation.ReferenceNumber }} <br>
    Subject: {{ .Conversation.Subject }}
</div>

<blockquote>{{ .Message.Content }}</blockquote>

<p>
    <a href="{{ RootURL }}/inboxes/assigned/conversation/{{ .Conversation.UUID }}">View Conversation</a>
</p>

<div>
    Best regards,<br>
    Libredesk
</div>

', false, 'New reply from contact', 'New reply on conversation #{{ .Conversation.ReferenceNumber }}', true
		WHERE NOT EXISTS (SELECT 1 FROM templates WHERE "name" = 'New reply from contact');

		INSERT INTO templates ("type", body, is_default, "name", subject, is_builtin)
		SELECT 'email_notification'::template_type, '
<p>{{ .Author.FullName }} replied to a conversation you are participating in:</p>

<div>
    Reference number: {{ .Conversation.ReferenceNumber }} <br>
    Subject: {{ .Conversation.Subject }}
</div>

<blockquote>{{ .Message.Content }}</blockquote>

<p>
    <a href="{{ RootURL }}/inboxes/assigned/conversation/{{ .Conversation.UUID }}">View Conversation</a>
</p>

<div>
    Best regards,<br>
    Libredesk
</div>

', false, 'New reply on participating conversation', 'New reply on conversation #{{ .Conversation.ReferenceNumber }}', true
		WHERE NOT EXISTS (SELECT 1 FROM templates WHERE "name" = 'New reply on participating conversation');

		INSERT INTO templates ("type", body, is_default, "name", subject, is_builtin)
		SELECT 'email_notification'::template_type, '
<p>{{ .Author.FullName }} replied and reopened a conversation assigned to you:</p>

<div>
    Reference number: {{ .Conversation.ReferenceNumber }} <br>
    Subject: {{ .Conversation.Subject }}
</div>

<blockquote>{{ .Message.Content }}</blockquote>

<p>
    <a href="{{ RootURL }}/inboxes/assigned/conversation/{{ .Conversation.UUID }}">View Conversation</a>
</p>

<div>
    Best regards,<br>
    Libredesk
</div>

', false, 'Conversation reopened', 'Conversation #{{ .Conversation.ReferenceNumber }} reopened', true
		WHERE NOT EXISTS (SELECT 1 FROM templates WHERE "name" = 'Conversation reopened');
	`)
	return err
}
