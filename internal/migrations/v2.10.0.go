package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_10_0 adds the guided pre-chat form feature: a new user type for the guided-form bot identity
// and the tables backing per-inbox guided forms and their handoff/completion events.
func V2_10_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	// ALTER TYPE ... ADD VALUE must run outside the transaction that uses the new value, so it is a
	// standalone statement.
	if _, err := db.Exec(`ALTER TYPE user_type ADD VALUE IF NOT EXISTS 'guided_form_bot';`); err != nil {
		return err
	}

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS guided_forms (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			user_id BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
			name TEXT NOT NULL DEFAULT '',
			inbox_id INTEGER NOT NULL REFERENCES inboxes(id) ON DELETE CASCADE,
			enabled BOOLEAN NOT NULL DEFAULT true,
			start_step_id TEXT NOT NULL DEFAULT '',
			steps JSONB NOT NULL DEFAULT '[]'::jsonb,
			on_complete_action TEXT NOT NULL DEFAULT 'team',
			on_complete_assistant_id INTEGER NULL REFERENCES ai_assistants(id) ON DELETE SET NULL,
			on_complete_team_id INTEGER NULL REFERENCES teams(id) ON DELETE SET NULL,
			completion_message TEXT NOT NULL DEFAULT '',
			CONSTRAINT constraint_guided_forms_on_complete_action CHECK (on_complete_action IN ('team', 'ai_assistant', 'unassigned'))
		);
		CREATE INDEX IF NOT EXISTS index_guided_forms_on_user_id ON guided_forms(user_id);
		CREATE INDEX IF NOT EXISTS index_guided_forms_on_inbox_id ON guided_forms(inbox_id);

		CREATE TABLE IF NOT EXISTS guided_form_events (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			form_id INTEGER NOT NULL REFERENCES guided_forms(id) ON DELETE CASCADE,
			conversation_id BIGINT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
			type TEXT NOT NULL,
			CONSTRAINT constraint_guided_form_events_on_type CHECK (type IN ('completed', 'handoff'))
		);
		CREATE INDEX IF NOT EXISTS index_guided_form_events_on_form_type_created ON guided_form_events(form_id, type, created_at);
		CREATE INDEX IF NOT EXISTS index_guided_form_events_on_conversation_id ON guided_form_events(conversation_id);
	`)
	return err
}
