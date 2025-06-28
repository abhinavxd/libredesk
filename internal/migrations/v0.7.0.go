package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V0_7_0 updates the database schema to v0.7.0.
func V0_7_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	// Create webhook_event enum type if it doesn't exist
	_, err := db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_type WHERE typname = 'webhook_event'
			) THEN
				CREATE TYPE webhook_event AS ENUM (
					'conversation.created',
					'conversation.status_changed',
					'conversation.tags_changed',
					'conversation.assigned',
					'conversation.unassigned',
					'message.created',
					'message.updated'
				);
			END IF;
		END
		$$;
	`)
	if err != nil {
		return err
	}

	// Create webhooks table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS webhooks (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			name TEXT NOT NULL,
			url TEXT NOT NULL,
			events webhook_event[] NOT NULL DEFAULT '{}',
			secret TEXT DEFAULT '',
			is_active BOOLEAN DEFAULT true,
			CONSTRAINT constraint_webhooks_on_name CHECK (length(name) <= 255),
			CONSTRAINT constraint_webhooks_on_url CHECK (length(url) <= 2048),
			CONSTRAINT constraint_webhooks_on_secret CHECK (length(secret) <= 255),
			CONSTRAINT constraint_webhooks_on_events_not_empty CHECK (array_length(events, 1) > 0)
		);
	`)
	if err != nil {
		return err
	}

	// Add webhooks:manage permission to Admin role
	_, err = db.Exec(`
		UPDATE roles
		SET permissions = array_append(permissions, 'webhooks:manage')
		WHERE name = 'Admin' AND NOT ('webhooks:manage' = ANY(permissions));
	`)
	if err != nil {
		return err
	}

	// Add API key authentication fields to users table
	_, err = db.Exec(`
		ALTER TABLE users 
		ADD COLUMN IF NOT EXISTS api_key TEXT NULL,
		ADD COLUMN IF NOT EXISTS api_secret TEXT NULL,
		ADD COLUMN IF NOT EXISTS api_key_last_used_at TIMESTAMPTZ NULL;
	`)
	if err != nil {
		return err
	}

	// Create index for API key field
	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS index_users_on_api_key ON users(api_key);
	`)
	if err != nil {
		return err
	}

	// Create article_categories table
    _, err = db.Exec(`
        DROP TABLE IF EXISTS article_categories CASCADE;
        CREATE TABLE article_categories (
            id SERIAL PRIMARY KEY,
            created_at TIMESTAMPTZ DEFAULT NOW(),
            updated_at TIMESTAMPTZ DEFAULT NOW(),
            "name" TEXT NOT NULL UNIQUE,
            CONSTRAINT constraint_article_categories_on_name CHECK (length("name") <= 140)
        );
    `)
    if err != nil {
        return err
    }

    // Create article_sections table
    _, err = db.Exec(`
        DROP TABLE IF EXISTS article_sections CASCADE;
        CREATE TABLE article_sections (
            id SERIAL PRIMARY KEY,
            created_at TIMESTAMPTZ DEFAULT NOW(),
            updated_at TIMESTAMPTZ DEFAULT NOW(),
            "name" TEXT NOT NULL UNIQUE,
            category_id INT REFERENCES article_categories(id) ON DELETE CASCADE ON UPDATE CASCADE,
            CONSTRAINT constraint_article_sections_on_name CHECK (length("name") <= 140)
        );
    `)
    if err != nil {
        return err
    }

	// Create articles table
    _, err = db.Exec(`
        DROP TABLE IF EXISTS articles CASCADE;
        CREATE TABLE articles (
            id SERIAL PRIMARY KEY,
            created_at TIMESTAMPTZ DEFAULT NOW(),
            updated_at TIMESTAMPTZ DEFAULT NOW(),
            title TEXT NOT NULL,
            content TEXT,
            section_id INT REFERENCES article_sections(id) ON DELETE CASCADE ON UPDATE CASCADE,
            published_at TIMESTAMPTZ,
            is_published BOOLEAN DEFAULT FALSE,
            CONSTRAINT constraint_articles_on_title CHECK (length(title) <= 255)
        );
    `)
    if err != nil {
        return err
    }

    // Create index for articles section_id
    _, err = db.Exec(`
        CREATE INDEX IF NOT EXISTS index_articles_on_section_id ON articles (section_id);
    `)
    if err != nil {
        return err
    }

    // Create index for article_sections category_id (you also have this in your SQL)
    _, err = db.Exec(`
        CREATE INDEX IF NOT EXISTS index_sections_on_category_id ON article_sections (category_id);
    `)
    if err != nil {
        return err
    }

	// Add article management permissions to Admin role
    _, err = db.Exec(`
        UPDATE roles
        SET permissions = array_cat(permissions, 
            ARRAY['article_category:manage', 'article_section:manage', 'article:manage', 'article_setting:manage']::text[]
        )
        WHERE name = 'Admin' 
        AND NOT (permissions @> ARRAY['article_category:manage', 'article_section:manage', 'article:manage', 'article_setting:manage']::text[]);
    `)
    if err != nil {
        return err
    }

	// Add article settings
    _, err = db.Exec(`
        INSERT INTO settings (key, value)
        VALUES 
            ('article.site_title', '""'::jsonb),
            ('article.site_description', '""'::jsonb),
            ('article.custom_script', '""'::jsonb),
            ('article.primary_font', '"inter"'::jsonb),
            ('article.logo_url', '""'::jsonb)
        ON CONFLICT (key) DO NOTHING;
    `)
    if err != nil {
        return err
    }

	return nil
}
