package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V0_9_0 adds AI settings and RAG tables with pgvector support.
func V0_9_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	// Add AI Settings (these work regardless of pgvector)
	_, err := db.Exec(`
		INSERT INTO settings ("key", value) VALUES
			('ai.enabled', 'false'::jsonb),
			('ai.provider', '"openai"'::jsonb),
			('ai.openai_api_key', '""'::jsonb),
			('ai.openrouter_api_key', '""'::jsonb),
			('ai.openrouter_model', '"anthropic/claude-3.5-sonnet"'::jsonb),
			('ai.embedding_model', '"text-embedding-3-small"'::jsonb),
			('ai.max_context_chunks', '5'::jsonb),
			('ai.similarity_threshold', '0.7'::jsonb),
			('ai.system_prompt', '"You are a helpful customer support assistant."'::jsonb)
		ON CONFLICT ("key") DO NOTHING;
	`)
	if err != nil {
		return err
	}

	// Try to enable pgvector extension - if it fails, skip RAG tables
	_, err = db.Exec(`CREATE EXTENSION IF NOT EXISTS vector;`)
	if err != nil {
		// pgvector not available - skip RAG tables
		return nil
	}

	// Create RAG tables (only if pgvector is available)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS rag_sources (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			name TEXT NOT NULL,
			source_type TEXT NOT NULL,
			config JSONB DEFAULT '{}'::jsonb NOT NULL,
			enabled BOOL DEFAULT TRUE NOT NULL,
			last_synced_at TIMESTAMPTZ NULL,
			CONSTRAINT constraint_rag_sources_on_name CHECK (length(name) <= 255),
			CONSTRAINT constraint_rag_sources_on_source_type CHECK (source_type IN ('macro', 'webpage', 'custom'))
		);

		CREATE TABLE IF NOT EXISTS rag_documents (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			source_id INT REFERENCES rag_sources(id) ON DELETE CASCADE NOT NULL,
			source_ref TEXT NULL,
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			content_hash TEXT NOT NULL,
			embedding vector(1536),
			metadata JSONB DEFAULT '{}'::jsonb NOT NULL,
			CONSTRAINT constraint_rag_documents_on_title CHECK (length(title) <= 500)
		);

		CREATE INDEX IF NOT EXISTS rag_documents_embedding_idx
		ON rag_documents USING hnsw (embedding vector_cosine_ops);

		CREATE INDEX IF NOT EXISTS rag_documents_source_id_idx ON rag_documents(source_id);
		CREATE INDEX IF NOT EXISTS rag_documents_source_ref_idx ON rag_documents(source_ref);
		CREATE INDEX IF NOT EXISTS rag_documents_content_hash_idx ON rag_documents(content_hash);

		CREATE UNIQUE INDEX IF NOT EXISTS rag_documents_source_ref_unique_idx 
		ON rag_documents(source_id, source_ref) WHERE source_ref IS NOT NULL;
	`)
	return err
}
