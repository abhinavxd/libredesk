package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_7_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS help_centers (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			name TEXT NOT NULL,
			slug TEXT NOT NULL UNIQUE,
			page_title TEXT NOT NULL DEFAULT '',
			header_text TEXT NOT NULL DEFAULT '',
			meta_description TEXT NOT NULL DEFAULT '',
			logo_url TEXT NOT NULL DEFAULT '',
			color TEXT NOT NULL DEFAULT '#1f93ff',
			nav_links JSONB NOT NULL DEFAULT '[]',
			custom_css TEXT NOT NULL DEFAULT '',
			custom_js TEXT NOT NULL DEFAULT '',
			default_locale TEXT NOT NULL DEFAULT 'en',
			allowed_locales JSONB NOT NULL DEFAULT '["en"]',
			is_active BOOLEAN NOT NULL DEFAULT true,
			theme JSONB NOT NULL DEFAULT '{}',
			public_url TEXT NOT NULL DEFAULT ''
		);
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`ALTER TABLE help_centers ADD COLUMN IF NOT EXISTS allowed_locales JSONB NOT NULL DEFAULT '["en"]';`); err != nil {
		return err
	}
	if _, err := db.Exec(`ALTER TABLE help_centers ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT true;`); err != nil {
		return err
	}
	if _, err := db.Exec(`ALTER TABLE help_centers ADD COLUMN IF NOT EXISTS theme JSONB NOT NULL DEFAULT '{}';`); err != nil {
		return err
	}
	if _, err := db.Exec(`ALTER TABLE help_centers ADD COLUMN IF NOT EXISTS meta_description TEXT NOT NULL DEFAULT '';`); err != nil {
		return err
	}
	if _, err := db.Exec(`ALTER TABLE help_centers ADD COLUMN IF NOT EXISTS public_url TEXT NOT NULL DEFAULT '';`); err != nil {
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
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`ALTER TABLE article_collections ADD COLUMN IF NOT EXISTS icon TEXT NOT NULL DEFAULT '';`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS index_unique_article_collections_on_help_center_slug_locale ON article_collections(help_center_id, slug, locale);`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS index_article_collections_on_help_center_id ON article_collections(help_center_id);`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS index_article_collections_on_parent_id ON article_collections(parent_id);`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS help_articles (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			collection_id INTEGER NOT NULL REFERENCES article_collections(id) ON DELETE CASCADE,
			author_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
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
			CONSTRAINT constraint_help_articles_on_status CHECK (status IN ('draft', 'published', 'archived'))
		);
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS index_unique_help_articles_on_collection_slug_locale ON help_articles(collection_id, slug, locale);`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS index_help_articles_on_collection_id ON help_articles(collection_id);`); err != nil {
		return err
	}

	// Idempotent upgrades for installs that ran an earlier build of this migration.
	for _, stmt := range []string{
		`ALTER TABLE help_articles ADD COLUMN IF NOT EXISTS author_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL`,
		`ALTER TABLE help_articles ADD COLUMN IF NOT EXISTS created_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL`,
		`ALTER TABLE help_articles ADD COLUMN IF NOT EXISTS excerpt TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE help_articles ADD COLUMN IF NOT EXISTS meta_title TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE help_articles ADD COLUMN IF NOT EXISTS meta_description TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE help_articles ADD COLUMN IF NOT EXISTS meta_image_url TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE help_articles DROP CONSTRAINT IF EXISTS constraint_help_articles_on_status`,
		`ALTER TABLE help_articles ADD CONSTRAINT constraint_help_articles_on_status CHECK (status IN ('draft', 'published', 'archived'))`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS index_help_articles_on_author_id ON help_articles(author_id);`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS index_help_articles_on_title_trgm ON help_articles USING gin (title gin_trgm_ops);`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS index_help_articles_on_content_trgm ON help_articles USING gin (content gin_trgm_ops);`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS help_article_feedback (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			article_id INTEGER NOT NULL REFERENCES help_articles(id) ON DELETE CASCADE,
			is_helpful BOOLEAN NOT NULL
		);
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS index_help_article_feedback_on_article_id ON help_article_feedback(article_id);`); err != nil {
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
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS index_help_search_queries_on_help_center_id ON help_search_queries(help_center_id);`); err != nil {
		return err
	}

	if _, err := db.Exec(`ALTER TABLE media ADD COLUMN IF NOT EXISTS private BOOLEAN NOT NULL DEFAULT true;`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		UPDATE media SET private = false
		WHERE model_type = 'users'
		AND model_id IN (SELECT id FROM users WHERE type = 'agent');
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

	return nil
}
