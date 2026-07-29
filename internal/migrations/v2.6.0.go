package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_6_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	for _, v := range []string{
		"new_reply",
		"new_reply_participating",
		"sla_first_response_warning",
		"sla_first_response_breach",
		"sla_next_response_warning",
		"sla_next_response_breach",
		"sla_resolution_warning",
		"sla_resolution_breach",
	} {
		if _, err := db.Exec(`ALTER TYPE user_notification_type ADD VALUE IF NOT EXISTS '` + v + `';`); err != nil {
			return err
		}
	}

	if _, err := db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'notification_channel') THEN
				CREATE TYPE notification_channel AS ENUM ('in_app', 'email');
			END IF;
		END$$;
	`); err != nil {
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
		INSERT INTO templates ("type", body, is_default, "name", subject, is_builtin)
		SELECT 'email_notification'::template_type, '
<p>{{ .Author.FullName }} replied to a conversation assigned to you:</p>

<div>
    Reference number: {{ .Conversation.ReferenceNumber }} <br>
    Subject: {{ .Conversation.Subject }}
</div>

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

	return nil
}
