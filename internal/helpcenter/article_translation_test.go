package helpcenter

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/helpcenter/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/zerodha/logf"
)

func TestCreateArticleTranslation(t *testing.T) {
	db := testutil.NewDB(t, "help_article_translation")
	db.MustExec(`
		INSERT INTO help_centers (name, slug, allowed_locales) VALUES ('Docs', 'docs', '["en", "fr"]');
		INSERT INTO article_collections (help_center_id, slug, locale, name, is_published) VALUES
			(1, 'general', 'en', 'General', true),
			(1, 'general', 'fr', 'Général', true);
	`)
	lo := logf.New(logf.Opts{})
	mgr, err := New(Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t)})
	if err != nil {
		t.Fatal(err)
	}

	source, err := mgr.CreateArticle(1, ArticleRequest{
		Slug: "billing", Locale: "en", Title: "Billing", Content: "English", Status: models.ArticleStatusPublished,
	})
	if err != nil {
		t.Fatal(err)
	}
	translation, err := mgr.CreateArticle(2, ArticleRequest{
		Slug: "facturation", Locale: "fr", Title: "Facturation", Content: "Français", Status: models.ArticleStatusPublished,
		TranslationOfID: &source.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if source.TranslationGroupID != translation.TranslationGroupID {
		t.Fatal("translation was created in a different group")
	}

	translations, err := mgr.GetPublishedArticleTranslations("docs", source.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(translations) != 2 || translations[0].Locale != "en" || translations[1].Locale != "fr" {
		t.Fatalf("unexpected translations: %#v", translations)
	}

	_, err = mgr.CreateArticle(2, ArticleRequest{
		Slug: "autre", Locale: "fr", Title: "Autre", Content: "Français", Status: models.ArticleStatusPublished,
		TranslationOfID: &source.ID,
	})
	if envErr, ok := err.(envelope.Error); !ok || envErr.ErrorType != envelope.ConflictError {
		t.Fatalf("expected duplicate-locale conflict, got %v", err)
	}
}
