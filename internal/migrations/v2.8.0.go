package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_8_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`
		CREATE OR REPLACE FUNCTION help_article_search_config(locale TEXT)
		RETURNS regconfig AS $$
			SELECT CASE split_part(locale, '-', 1)
				WHEN 'ar' THEN 'arabic'
				WHEN 'da' THEN 'danish'
				WHEN 'nl' THEN 'dutch'
				WHEN 'en' THEN 'english'
				WHEN 'fi' THEN 'finnish'
				WHEN 'fr' THEN 'french'
				WHEN 'de' THEN 'german'
				WHEN 'el' THEN 'greek'
				WHEN 'hu' THEN 'hungarian'
				WHEN 'id' THEN 'indonesian'
				WHEN 'ga' THEN 'irish'
				WHEN 'it' THEN 'italian'
				WHEN 'lt' THEN 'lithuanian'
				WHEN 'ne' THEN 'nepali'
				WHEN 'no' THEN 'norwegian'
				WHEN 'pt' THEN 'portuguese'
				WHEN 'ro' THEN 'romanian'
				WHEN 'ru' THEN 'russian'
				WHEN 'es' THEN 'spanish'
				WHEN 'sv' THEN 'swedish'
				WHEN 'ta' THEN 'tamil'
				WHEN 'tr' THEN 'turkish'
				ELSE 'simple'
			END::regconfig;
		$$ LANGUAGE sql IMMUTABLE;
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS help_centers (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			name TEXT NOT NULL,
			slug TEXT NOT NULL UNIQUE,
			page_title TEXT NOT NULL DEFAULT '',
			meta_description TEXT NOT NULL DEFAULT '',
			custom_css TEXT NOT NULL DEFAULT '',
			custom_js TEXT NOT NULL DEFAULT '',
			default_locale TEXT NOT NULL DEFAULT 'en',
			allowed_locales JSONB NOT NULL DEFAULT '["en"]',
			is_active BOOLEAN NOT NULL DEFAULT true,
			theme JSONB NOT NULL DEFAULT '{}',
			custom_domain TEXT NOT NULL DEFAULT '',
			template TEXT NOT NULL DEFAULT 'classic',
			CONSTRAINT constraint_help_centers_on_template CHECK (template IN ('docs', 'classic'))
		);
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS article_collections (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			help_center_id INTEGER NOT NULL REFERENCES help_centers(id) ON DELETE CASCADE,
			slug TEXT NOT NULL,
			parent_id INTEGER NULL REFERENCES article_collections(id) ON DELETE CASCADE,
			locale TEXT NOT NULL DEFAULT 'en',
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			icon TEXT NOT NULL DEFAULT '',
			sort_order INTEGER NOT NULL DEFAULT 0,
			is_published BOOLEAN NOT NULL DEFAULT false
		);
		CREATE UNIQUE INDEX IF NOT EXISTS index_unique_article_collections_on_help_center_slug_locale ON article_collections(help_center_id, slug, locale);
		CREATE INDEX IF NOT EXISTS index_article_collections_on_help_center_id ON article_collections(help_center_id);
		CREATE INDEX IF NOT EXISTS index_article_collections_on_parent_id ON article_collections(parent_id);
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS help_articles (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			collection_id INTEGER NOT NULL REFERENCES article_collections(id) ON DELETE CASCADE,
			author_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
			created_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
			slug TEXT NOT NULL,
			locale TEXT NOT NULL DEFAULT 'en',
			title TEXT NOT NULL,
			content TEXT NOT NULL DEFAULT '',
			excerpt TEXT NOT NULL DEFAULT '',
			meta_title TEXT NOT NULL DEFAULT '',
			meta_description TEXT NOT NULL DEFAULT '',
			meta_image_url TEXT NOT NULL DEFAULT '',
			sort_order INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'draft',
			view_count INTEGER NOT NULL DEFAULT 0,
			ai_enabled BOOLEAN NOT NULL DEFAULT false,
			embedded_fingerprint TEXT NOT NULL DEFAULT '',
			-- left() caps the indexed body below the 1MB tsvector limit so oversized articles still save.
			search_tsv TSVECTOR GENERATED ALWAYS AS (
				setweight(to_tsvector(help_article_search_config(locale), title), 'A') ||
				setweight(to_tsvector(help_article_search_config(locale), excerpt), 'B') ||
				setweight(to_tsvector(help_article_search_config(locale), left(content, 100000)), 'C')
			) STORED,
			CONSTRAINT constraint_help_articles_on_status CHECK (status IN ('draft', 'published'))
		);
		CREATE UNIQUE INDEX IF NOT EXISTS index_unique_help_articles_on_collection_slug_locale ON help_articles(collection_id, slug, locale);
		CREATE INDEX IF NOT EXISTS index_help_articles_on_collection_id ON help_articles(collection_id);
		CREATE INDEX IF NOT EXISTS index_help_articles_on_author_id ON help_articles(author_id);
		CREATE INDEX IF NOT EXISTS index_help_articles_on_title_trgm ON help_articles USING gin (title gin_trgm_ops);
		CREATE INDEX IF NOT EXISTS index_help_articles_on_content_trgm ON help_articles USING gin (content gin_trgm_ops);
		CREATE INDEX IF NOT EXISTS index_help_articles_on_search_tsv ON help_articles USING gin (search_tsv);
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS help_article_feedback (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			article_id INTEGER NOT NULL REFERENCES help_articles(id) ON DELETE CASCADE,
			is_helpful BOOLEAN NOT NULL
		);
		CREATE INDEX IF NOT EXISTS index_help_article_feedback_on_article_id ON help_article_feedback(article_id);
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS help_search_queries (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			help_center_id INTEGER NOT NULL REFERENCES help_centers(id) ON DELETE CASCADE,
			query TEXT NOT NULL,
			results_count INTEGER NOT NULL DEFAULT 0
		);
		CREATE INDEX IF NOT EXISTS index_help_search_queries_on_help_center_id ON help_search_queries(help_center_id);
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`ALTER TABLE media ADD COLUMN IF NOT EXISTS private BOOLEAN NOT NULL DEFAULT true;`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		UPDATE media SET private = false
		WHERE model_type = 'users'
		AND model_id IN (SELECT id FROM users WHERE type IN ('agent', 'ai_assistant'));
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		UPDATE roles
		SET permissions = array_append(permissions, 'help_center:manage')
		WHERE name = 'Admin' AND NOT ('help_center:manage' = ANY(permissions));
	`); err != nil {
		return err
	}

	for _, permission := range []string{"contacts:delete", "contacts:export"} {
		if _, err := db.Exec(`
			UPDATE roles
			SET permissions = array_append(permissions, $1)
			WHERE name = 'Admin' AND NOT ($1 = ANY(permissions));
		`, permission); err != nil {
			return err
		}
	}

	if _, err := db.Exec(`ALTER TYPE activity_log_type ADD VALUE IF NOT EXISTS 'contact_deleted';`); err != nil {
		return err
	}
	if _, err := db.Exec(`ALTER TYPE activity_log_type ADD VALUE IF NOT EXISTS 'contact_data_exported';`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS index_users_on_availability_status_when_agent ON users(availability_status) WHERE type = 'agent' AND deleted_at IS NULL;
		CREATE INDEX IF NOT EXISTS index_conversations_on_resolved_at ON conversations(resolved_at);
		CREATE INDEX IF NOT EXISTS index_applied_slas_on_created_at ON applied_slas(created_at);
		CREATE INDEX IF NOT EXISTS index_sla_events_on_created_at ON sla_events(created_at);
		CREATE INDEX IF NOT EXISTS index_csat_responses_on_created_at ON csat_responses(created_at);
	`); err != nil {
		return err
	}

	return nil
}
