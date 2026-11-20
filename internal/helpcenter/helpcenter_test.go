package helpcenter

import (
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/abhinavxd/libredesk/internal/helpcenter/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/zerodha/logf"
)

func TestSearchPublishedArticlesPreviews(t *testing.T) {
	db := testutil.NewDB(t, "help_search_previews")
	db.MustExec(`
		INSERT INTO help_centers (name, slug) VALUES ('Docs', 'docs');
		INSERT INTO article_collections (help_center_id, slug, locale, name, is_published) VALUES
			(1, 'billing', 'en', 'Billing', true),
			(1, 'private', 'en', 'Private', false);
		INSERT INTO help_articles (collection_id, slug, locale, title, content, excerpt, status) VALUES
			(1, 'preview', 'en', 'Payment preview', '<p>Body text.</p>', 'An explicit preview.', 'published'),
			(1, 'fallback', 'en', 'Payment fallback', '<p>Pay <strong>online</strong> &amp; confirm.</p>', '', 'published'),
			(1, 'long', 'en', 'Payment long', '<p>' || repeat('日本語 ', 100) || '</p>', '', 'published'),
			(1, 'heading', 'en', 'Payment heading', '<h2>Payment heading</h2><p>Body after the heading.</p>', '', 'published'),
			(1, 'deep', 'en', 'Deep match', '<p>' || repeat('filler ', 60) || 'the payment clears overnight ' || repeat('tail ', 60) || '</p>', '', 'published'),
			(1, 'draft', 'en', 'Payment draft', 'Private body', '', 'draft'),
			(2, 'hidden', 'en', 'Payment hidden', 'Private body', '', 'published');
	`)
	lo := logf.New(logf.Opts{})
	mgr, err := New(Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t)})
	if err != nil {
		t.Fatal(err)
	}
	results, err := mgr.SearchPublishedArticles("docs", "payment", "en", 20 /** limit **/)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 5 {
		t.Fatalf("got %d results, want 5 published articles", len(results))
	}
	for _, a := range results {
		if a.CollectionName != "Billing" {
			t.Errorf("article %s has collection %q", a.Slug, a.CollectionName)
		}
		switch a.Slug {
		case "preview":
			if a.Content != "An explicit preview." {
				t.Errorf("explicit preview changed: %q", a.Content)
			}
		case "fallback":
			if a.Content != "Pay online & confirm." {
				t.Errorf("unexpected plain text fallback: %q", a.Content)
			}
		case "long":
			if !utf8.ValidString(a.Content) || utf8.RuneCountInString(a.Content) > maxSearchExcerptLen || strings.Contains(a.Content, "<") || !strings.HasSuffix(a.Content, "…") {
				t.Errorf("invalid preview: %q", a.Content)
			}
		case "heading":
			if a.Content != "Body after the heading." {
				t.Errorf("heading leaked into preview: %q", a.Content)
			}
		case "deep":
			if !strings.HasPrefix(a.Content, "…filler ") || !strings.Contains(a.Content, "the payment clears overnight") || utf8.RuneCountInString(a.Content) > maxSearchExcerptLen {
				t.Errorf("preview does not show the match: %q", a.Content)
			}
		default:
			t.Errorf("unexpected public result: %s", a.Slug)
		}
	}
}

func TestSearchSnippet(t *testing.T) {
	var (
		match = regexp.MustCompile("(?i)payment|clears")
		long  = strings.Repeat("filler ", 60) + "the Payment clears overnight " + strings.Repeat("tail ", 60)
		early = "A payment note. " + strings.Repeat("tail ", 80)
		cjk   = strings.Repeat("日本語", 120)
	)
	for _, tt := range []struct {
		name, text string
		match      *regexp.Regexp
		prefix     string
		suffix     string
		contains   string
	}{
		{"short text is returned whole", "Pay online & confirm.", match, "Pay online", "confirm.", ""},
		{"empty", "", match, "", "", ""},
		{"early match keeps the start", early, match, "A payment note.", "…", ""},
		{"deep match moves the window to it", long, match, "…filler ", "…", "the Payment clears overnight"},
		{"no match keeps the start", long, regexp.MustCompile("absent"), "filler filler", "…", ""},
		{"nil matcher keeps the start", long, nil, "filler filler", "…", ""},
		{"text without spaces", cjk + "payment" + cjk, match, "…", "…", "payment"},
		{"one-letter query words do not anchor the window", "a note. " + long, searchMatcher("a payment"), "…filler ", "…", "the Payment clears overnight"},
		{"query of one-letter words keeps the start", long, searchMatcher("a b"), "filler filler", "…", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := searchSnippet(tt.text, tt.match)
			if !utf8.ValidString(got) || utf8.RuneCountInString(got) > maxSearchExcerptLen {
				t.Fatalf("invalid or oversized snippet (%d runes): %q", utf8.RuneCountInString(got), got)
			}
			if !strings.HasPrefix(got, tt.prefix) || !strings.HasSuffix(got, tt.suffix) || !strings.Contains(got, tt.contains) {
				t.Errorf("searchSnippet() = %q, want prefix %q, suffix %q, containing %q", got, tt.prefix, tt.suffix, tt.contains)
			}
		})
	}
}

func TestGetCollectionPath(t *testing.T) {
	db := testutil.NewDB(t, "help_collection_path")
	db.MustExec(`
		INSERT INTO help_centers (name, slug) VALUES ('Docs', 'docs');
		INSERT INTO article_collections (help_center_id, slug, locale, name) VALUES (1, 'root', 'en', 'Root');
		INSERT INTO article_collections (help_center_id, slug, locale, name, parent_id) VALUES (1, 'child', 'en', 'Child', 1);
		INSERT INTO article_collections (help_center_id, slug, locale, name, parent_id) VALUES (1, 'leaf', 'en', 'Leaf', 2);
	`)
	lo := logf.New(logf.Opts{})
	mgr, err := New(Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t)})
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		id   int
		want string
	}{
		{1, "root"},
		{2, "root/child"},
		{3, "root/child/leaf"},
	} {
		collection, err := mgr.GetCollectionByID(tt.id)
		if err != nil {
			t.Fatal(err)
		}
		var slugs []string
		for _, c := range mgr.GetCollectionPath(collection) {
			slugs = append(slugs, c.Slug)
		}
		if got := strings.Join(slugs, "/"); got != tt.want {
			t.Errorf("GetCollectionPath(%d) = %q, want %q", tt.id, got, tt.want)
		}
	}
	if path := mgr.GetCollectionPath(models.Collection{}); path != nil {
		t.Errorf("a missing collection has path %v, want none", path)
	}
}
