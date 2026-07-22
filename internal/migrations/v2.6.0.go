package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_6_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	// ALTER TYPE ADD VALUE cannot run inside a transaction; each Exec here is autocommit.
	if _, err := db.Exec(`ALTER TYPE user_type ADD VALUE IF NOT EXISTS 'ai_assistant';`); err != nil {
		return err
	}

	if _, err := db.Exec(`ALTER TABLE ai_providers ADD COLUMN IF NOT EXISTS type TEXT NOT NULL DEFAULT 'completion';`); err != nil {
		return err
	}
	if _, err := db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'constraint_ai_providers_on_type') THEN
				ALTER TABLE ai_providers ADD CONSTRAINT constraint_ai_providers_on_type CHECK (type IN ('completion', 'embedding'));
			END IF;
		END$$;
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS index_unique_ai_providers_on_type ON ai_providers(type);`); err != nil {
		return err
	}
	if _, err := db.Exec(`UPDATE ai_providers SET type = 'completion' WHERE name = 'openai' AND type IS DISTINCT FROM 'embedding';`); err != nil {
		return err
	}
	if _, err := db.Exec(`
		INSERT INTO ai_providers (name, provider, type, config, is_default)
		VALUES ('embedding', 'openai', 'embedding', '{"api_key": ""}'::jsonb, false)
		ON CONFLICT (name) DO NOTHING;
	`); err != nil {
		return err
	}
	// Older releases hardcoded temperature 0.7; keep that behavior for configured providers that predate the field.
	if _, err := db.Exec(`
		UPDATE ai_providers SET config = config || '{"temperature": 0.7}'::jsonb
		WHERE type = 'completion' AND COALESCE(config->>'api_key', '') <> '' AND NOT config ? 'temperature';
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'ai_knowledge_type') THEN
				CREATE TYPE ai_knowledge_type AS ENUM ('snippet');
			END IF;
		END$$;
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS ai_knowledge_base (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			type ai_knowledge_type NOT NULL DEFAULT 'snippet',
			title TEXT NOT NULL DEFAULT '',
			content TEXT NOT NULL,
			enabled BOOLEAN NOT NULL DEFAULT true,
			source TEXT NOT NULL DEFAULT 'manual',
			source_url TEXT NOT NULL DEFAULT '',
			embedded_fingerprint TEXT NOT NULL DEFAULT ''
		);
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`ALTER TABLE ai_knowledge_base ADD COLUMN IF NOT EXISTS source_url TEXT NOT NULL DEFAULT '';`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS index_ai_knowledge_base_on_type_enabled ON ai_knowledge_base(type, enabled);`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS embeddings (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			source_type TEXT NOT NULL,
			source_id BIGINT NOT NULL,
			chunk_text TEXT NOT NULL,
			embedding BYTEA,
			dimensions INTEGER NOT NULL DEFAULT 0,
			meta JSONB NOT NULL DEFAULT '{}'
		);
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS index_embeddings_on_source_type_source_id ON embeddings(source_type, source_id);`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS ai_tools (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			name TEXT NOT NULL UNIQUE,
			description TEXT NOT NULL DEFAULT '',
			url TEXT NOT NULL,
			method TEXT NOT NULL DEFAULT 'POST',
			auth JSONB NOT NULL DEFAULT '{}',
			parameters JSONB NOT NULL DEFAULT '{}',
			enabled BOOLEAN NOT NULL DEFAULT true,
			requires_verification BOOLEAN NOT NULL DEFAULT true,
			CONSTRAINT constraint_ai_tools_on_name CHECK (name ~ '^[a-zA-Z0-9_-]+$' AND length(name) <= 64)
		);
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`ALTER TABLE ai_tools ADD COLUMN IF NOT EXISTS requires_verification BOOLEAN NOT NULL DEFAULT true;`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS ai_assistants (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			user_id BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
			description TEXT NOT NULL DEFAULT '',
			instructions TEXT NOT NULL DEFAULT '',
			guardrails TEXT NOT NULL DEFAULT '',
			expectation TEXT NOT NULL DEFAULT '',
			tone TEXT NOT NULL DEFAULT 'professional',
			response_length TEXT NOT NULL DEFAULT 'balanced',
			max_turns INTEGER NOT NULL DEFAULT 6,
			fallback_team_id INTEGER NULL REFERENCES teams(id) ON DELETE SET NULL,
			handoff_enabled BOOLEAN NOT NULL DEFAULT true,
			languages TEXT[] NOT NULL DEFAULT '{}',
			enabled BOOLEAN NOT NULL DEFAULT true,
			CONSTRAINT constraint_ai_assistants_on_tone CHECK (tone IN ('friendly', 'professional', 'neutral', 'casual')),
			CONSTRAINT constraint_ai_assistants_on_response_length CHECK (response_length IN ('concise', 'balanced', 'detailed')),
			CONSTRAINT constraint_ai_assistants_on_max_turns CHECK (max_turns > 0 AND max_turns <= 20)
		);
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`ALTER TABLE ai_assistants ADD COLUMN IF NOT EXISTS handoff_enabled BOOLEAN NOT NULL DEFAULT true;`); err != nil {
		return err
	}
	if _, err := db.Exec(`ALTER TABLE ai_assistants ADD COLUMN IF NOT EXISTS languages TEXT[] NOT NULL DEFAULT '{}';`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS index_ai_assistants_on_user_id ON ai_assistants(user_id);`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS ai_assistant_tools (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			assistant_id INTEGER NOT NULL REFERENCES ai_assistants(id) ON DELETE CASCADE,
			tool_id INTEGER NOT NULL REFERENCES ai_tools(id) ON DELETE CASCADE
		);
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS index_unique_ai_assistant_tools ON ai_assistant_tools(assistant_id, tool_id);`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS index_ai_assistant_tools_on_assistant_id ON ai_assistant_tools(assistant_id);`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS ai_agent_events (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			assistant_id INTEGER NOT NULL REFERENCES ai_assistants(id) ON DELETE CASCADE,
			conversation_id BIGINT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
			type TEXT NOT NULL,
			CONSTRAINT constraint_ai_agent_events_on_type CHECK (type IN ('handoff', 'resolve'))
		);
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS index_ai_agent_events_on_assistant_type_created ON ai_agent_events(assistant_id, type, created_at);`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS index_ai_agent_events_on_conversation_id ON ai_agent_events(conversation_id);`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS copilot_messages (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			conversation_id BIGINT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			"role" TEXT NOT NULL,
			content TEXT NOT NULL,
			CONSTRAINT constraint_copilot_messages_on_role CHECK ("role" IN ('user', 'assistant'))
		);
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS index_copilot_messages_on_conversation_user ON copilot_messages(conversation_id, user_id, id);`); err != nil {
		return err
	}

	if _, err := db.Exec(`INSERT INTO settings ("key", value) VALUES ('app.copilot_name', '"Juno"'::jsonb) ON CONFLICT ("key") DO NOTHING;`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS ai_faq_suggestions (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			conversation_id BIGINT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
			question TEXT NOT NULL,
			answer TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			reviewed_by_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
			reviewed_at TIMESTAMPTZ NULL,
			CONSTRAINT constraint_ai_faq_suggestions_on_status CHECK (status IN ('pending', 'approved', 'rejected'))
		);
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS index_ai_faq_suggestions_on_status_created ON ai_faq_suggestions(status, created_at);`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS index_ai_faq_suggestions_on_conversation_id ON ai_faq_suggestions(conversation_id);`); err != nil {
		return err
	}

	if _, err := db.Exec(`INSERT INTO settings ("key", value) VALUES ('ai_agent.faq_learning_enabled', 'false'::jsonb) ON CONFLICT ("key") DO NOTHING;`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		INSERT INTO ai_prompts ("key", content, title)
		VALUES ('fix_grammar_spelling', 'Fix any spelling and grammar mistakes in the text while retaining the original meaning and tone.', 'Fix Grammar & Spelling')
		ON CONFLICT ("key") DO NOTHING;
	`); err != nil {
		return err
	}

	return nil
}
