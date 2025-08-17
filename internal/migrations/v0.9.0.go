package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V0_9_0 updates the database schema to v0.9.0.
func V0_9_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmts := []string{
		// Enable pgvector extension
		`CREATE EXTENSION IF NOT EXISTS vector`,

		// Create AI knowledge type enum
		`DROP TYPE IF EXISTS "ai_knowledge_type" CASCADE;
		CREATE TYPE "ai_knowledge_type" AS ENUM ('snippet')`,

		// Create help_centers table
		`CREATE TABLE IF NOT EXISTS help_centers (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			name VARCHAR(255) NOT NULL,
			slug VARCHAR(100) UNIQUE NOT NULL,
			page_title VARCHAR(255) NOT NULL,
			view_count INTEGER DEFAULT 0,
			default_locale VARCHAR(10) DEFAULT 'en' NOT NULL
		)`,

		// Create article_collections table
		`
		CREATE TABLE IF NOT EXISTS article_collections (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			help_center_id INTEGER NOT NULL REFERENCES help_centers(id) ON DELETE CASCADE,
			slug VARCHAR(255) NOT NULL,
			parent_id INTEGER REFERENCES article_collections(id) ON DELETE CASCADE,
			locale VARCHAR(10) NOT NULL DEFAULT 'en',
			name VARCHAR(255) NOT NULL,
			description TEXT,
			sort_order INTEGER DEFAULT 0,
			is_published BOOLEAN DEFAULT false
		);

		CREATE UNIQUE INDEX IF NOT EXISTS index_article_collections_slug_per_helpcenter_locale
		ON article_collections(help_center_id, slug, locale);
		`,

		// Create help articles table with status enum
		`CREATE TABLE IF NOT EXISTS help_articles (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			collection_id INTEGER NOT NULL REFERENCES article_collections(id) ON DELETE CASCADE,
			slug VARCHAR(255) NOT NULL,
			locale VARCHAR(10) NOT NULL DEFAULT 'en',
			title VARCHAR(255) NOT NULL,
			content TEXT NOT NULL,
			sort_order INTEGER DEFAULT 0,
			status VARCHAR(20) DEFAULT 'draft' CHECK (status IN ('draft', 'published')),
			view_count INTEGER DEFAULT 0,
			ai_enabled BOOLEAN DEFAULT false
		);
		CREATE UNIQUE INDEX IF NOT EXISTS index_help_articles_slug_per_collection_locale
		ON help_articles(collection_id, slug, locale);
		`,

		// Create AI knowledge base table
		`CREATE TABLE IF NOT EXISTS ai_knowledge_base (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			type ai_knowledge_type NOT NULL DEFAULT 'snippet',
			content TEXT NOT NULL,
			enabled BOOLEAN DEFAULT true
		)`,

		// Create embeddings table
		`CREATE TABLE IF NOT EXISTS embeddings (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			source_type TEXT NOT NULL, -- 'article', 'knowledge_sources'
			source_id BIGINT NOT NULL,
			chunk_text TEXT NOT NULL,
			embedding vector(1536),
			meta JSONB DEFAULT '{}' NOT NULL
		)`,

		// Create indexes for help_centers
		`CREATE INDEX IF NOT EXISTS index_help_centers_slug ON help_centers(slug)`,

		// Create indexes for collections
		`CREATE INDEX IF NOT EXISTS index_collections_help_center_id ON article_collections(help_center_id)`,
		`CREATE INDEX IF NOT EXISTS index_collections_parent_id ON article_collections(parent_id)`,
		`CREATE INDEX IF NOT EXISTS index_collections_locale ON article_collections(help_center_id, locale, is_published)`,
		`CREATE INDEX IF NOT EXISTS index_collections_ordering ON article_collections(help_center_id, parent_id, sort_order)`,

		// Create indexes for articles
		`CREATE INDEX IF NOT EXISTS index_articles_collection_id ON help_articles(collection_id)`,
		`CREATE INDEX IF NOT EXISTS index_articles_locale ON help_articles(collection_id, locale, status)`,
		`CREATE INDEX IF NOT EXISTS index_articles_ordering ON help_articles(collection_id, sort_order)`,

		// Create index for ai_knowledge_base
		`CREATE INDEX IF NOT EXISTS index_ai_knowledge_base_type_enabled ON ai_knowledge_base(type, enabled)`,

		// Create index for embeddings
		`CREATE INDEX IF NOT EXISTS index_embeddings_on_source_type_source_id ON embeddings(source_type, source_id)`,

		// Create HNSW index for vector similarity search on embeddings table
		`CREATE INDEX IF NOT EXISTS index_embeddings_embedding ON embeddings USING hnsw (embedding vector_cosine_ops)`,
	}

	for _, stmt := range stmts {
		if _, err = tx.Exec(stmt); err != nil {
			return err
		}
	}

	// Create the function to enforce collection max depth
	_, err = tx.Exec(`
		CREATE OR REPLACE FUNCTION enforce_collection_max_depth()
		RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN
			IF NEW.parent_id IS NOT NULL AND EXISTS (
				SELECT 1 FROM article_collections p1
				JOIN article_collections p2 ON p1.parent_id = p2.id
				WHERE p1.id = NEW.parent_id AND p2.parent_id IS NOT NULL
			) THEN
				RAISE EXCEPTION 'Collections can only be nested up to 3 levels deep';
			END IF;
			RETURN NEW;
		END;
		$$;
	`)
	if err != nil {
		return err
	}

	// Create the trigger
	_, err = tx.Exec(`
		DROP TRIGGER IF EXISTS trg_enforce_collection_depth_limit ON article_collections;
		CREATE TRIGGER trg_enforce_collection_depth_limit
		BEFORE INSERT OR UPDATE ON article_collections
		FOR EACH ROW EXECUTE FUNCTION enforce_collection_max_depth()
	`)
	if err != nil {
		return err
	}

	// Add 'ai_assistant' type to `user_type` enum if it doesn't exist
	var exists bool
	err = db.Get(&exists, `
		SELECT EXISTS (
			SELECT 1
			FROM pg_enum
			WHERE enumlabel = 'ai_assistant'
			AND enumtypid = (
				SELECT oid FROM pg_type WHERE typname = 'user_type'
			)
		)
	`)
	if err != nil {
		return err
	}
	if !exists {
		_, err = db.Exec(`ALTER TYPE user_type ADD VALUE 'ai_assistant'`)
		if err != nil {
			return err
		}
	}

	// Add 'meta' column to users table if it does not exist
	_, err = tx.Exec(`
		ALTER TABLE users ADD COLUMN IF NOT EXISTS meta JSONB DEFAULT '{}'::jsonb NOT NULL;
	`)
	if err != nil {
		return err
	}

	// Add 'help_center_id' column to inboxes table if it does not exist
	_, err = tx.Exec(`
		ALTER TABLE inboxes ADD COLUMN IF NOT EXISTS help_center_id INT REFERENCES help_centers(id);
	`)
	if err != nil {
		return err
	}

	return tx.Commit()
}
