package migrations

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
)

func TestWidgetCampaignMigration(t *testing.T) {
	db := testutil.NewDB(t, "widget_migration")
	db.MustExec(`DROP TABLE widget_campaign_deliveries`)
	for range 2 {
		if err := V2_10_0(db, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	if err := db.Get(&count, `SELECT COUNT(*) FROM pg_indexes WHERE tablename = 'widget_campaign_deliveries'`); err != nil {
		t.Fatal(err)
	}
	if count != 4 {
		t.Fatalf("expected primary key and three indexes, got %d", count)
	}
}

func TestHelpArticleTranslationGroupMigration(t *testing.T) {
	db := testutil.NewDB(t, "article_translation_group_migration")
	db.MustExec(`
		INSERT INTO help_centers (name, slug, allowed_locales) VALUES ('Docs', 'docs', '["en", "fr"]');
		INSERT INTO article_collections (help_center_id, slug, locale, name) VALUES
			(1, 'general', 'en', 'General'),
			(1, 'general', 'fr', 'Général'),
			(1, 'other', 'en', 'Other');
		INSERT INTO help_articles (collection_id, slug, locale, title) VALUES
			(1, 'billing', 'en', 'Billing'),
			(2, 'billing', 'fr', 'Facturation'),
			(1, 'refunds', 'en', 'Refunds'),
			(1, 'overview', 'en', 'Overview'),
			(3, 'overview', 'en', 'Another overview');
		ALTER TABLE help_articles DROP COLUMN translation_group_id;
	`)

	for range 2 {
		if err := V2_10_0(db, nil, nil); err != nil {
			t.Fatal(err)
		}
	}

	var grouped int
	if err := db.Get(&grouped, `
		SELECT COUNT(DISTINCT translation_group_id)
		FROM help_articles
		WHERE slug = 'billing'
	`); err != nil {
		t.Fatal(err)
	}
	if grouped != 1 {
		t.Fatalf("expected same-slug translations in one group, got %d groups", grouped)
	}

	var totalGroups int
	if err := db.Get(&totalGroups, `SELECT COUNT(DISTINCT translation_group_id) FROM help_articles`); err != nil {
		t.Fatal(err)
	}
	if totalGroups != 4 {
		t.Fatalf("expected unrelated articles in separate groups, got %d groups", totalGroups)
	}

	var ambiguousGroups int
	if err := db.Get(&ambiguousGroups, `
		SELECT COUNT(DISTINCT translation_group_id)
		FROM help_articles
		WHERE slug = 'overview'
	`); err != nil {
		t.Fatal(err)
	}
	if ambiguousGroups != 2 {
		t.Fatalf("expected ambiguous same-locale articles to remain separate, got %d groups", ambiguousGroups)
	}
}
