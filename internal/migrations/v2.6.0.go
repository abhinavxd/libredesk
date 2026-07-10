package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_6_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
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
			enabled BOOLEAN NOT NULL DEFAULT true
		);
	`); err != nil {
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
			CONSTRAINT constraint_ai_tools_on_name CHECK (name ~ '^[a-zA-Z0-9_-]+$' AND length(name) <= 64)
		);
	`); err != nil {
		return err
	}

	return nil
}
