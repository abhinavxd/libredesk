package locales

import (
	"fmt"
	"io/fs"
	"strings"
	"sync"
	"testing"

	catalogs "github.com/abhinavxd/libredesk/locales"
	"github.com/leonelquinteros/gotext"
)

func TestReloadCatalogsPreservesReferences(t *testing.T) {
	locale, err := New("en")
	if err != nil {
		t.Fatal(err)
	}
	// Managers and template functions retain this original locale pointer.
	translate := locale.Get
	for _, code := range []string{"fr", "en", "de", "fr"} {
		if err := ReloadCatalogs(locale, code); err != nil {
			t.Fatal(err)
		}
		want, err := New(code)
		if err != nil {
			t.Fatal(err)
		}
		if got := translate("%[1]s empty", "email"); got != want.Get("%[1]s empty", "email") {
			t.Fatalf("%s: stale backend translation %q", code, got)
		}
		for _, n := range []int{0, 1, 2} {
			got := locale.GetND("templates", "%d result", "%d results", n, n)
			if got != want.GetND("templates", "%d result", "%d results", n, n) {
				t.Fatalf("%s: stale template plural n=%d: %q", code, n, got)
			}
		}
	}
}

func TestReloadCatalogsConcurrentReaders(t *testing.T) {
	locale, err := New("fr")
	if err != nil {
		t.Fatal(err)
	}
	var readers sync.WaitGroup
	for range 4 {
		readers.Go(func() {
			for range 500 {
				if got := locale.Get("%[1]s empty", "email"); got != "email vide" && got != "email empty" {
					t.Errorf("unexpected concurrent translation %q", got)
				}
				if got := locale.GetND("templates", "%d result",
					"%d results",
					0,
					0); got != "0 résultat" && got != "0 results" {
					t.Errorf("unexpected concurrent plural %q", got)
				}
			}
		})
	}
	for range 10 {
		for _, code := range []string{"en", "fr"} {
			if err := ReloadCatalogs(locale, code); err != nil {
				t.Error(err)
			}
		}
	}
	readers.Wait()
}

func TestLocaleResolution(t *testing.T) {
	for requested, want := range map[string]string{
		"fr": "fr-FR", "fr_CA": "fr-FR", " FR-fr ": "fr-FR",
		"de-AT": "de-DE", "pt-PT": "pt-PT", "pt": "pt-BR",
		"zh-Hans": "zh-CN", "zh-TW": "en-US", "zh-Hant": "en-US",
		"en-GB": "en-US", "unknown": "en-US", "": "en-US",
	} {
		if got := resolve(requested); got != want {
			t.Errorf("resolve(%q) = %q, want %q", requested, got, want)
		}
	}
}

func TestDomainsFormattingAndFallback(t *testing.T) {
	fr, err := New("fr-FR")
	if err != nil {
		t.Fatal(err)
	}
	if got := fr.Get("Error parsing request"); got != "Erreur lors de l'analyse de la requête" {
		t.Errorf("backend translation = %q", got)
	}
	if got := fr.GetD("templates", "Rate your recent interaction"); got == "Rate your recent interaction" {
		t.Errorf("template translation missing: %q", got)
	}
	if got := fr.Get("Rate your recent interaction"); got != "Rate your recent interaction" {
		t.Errorf("template-only message leaked into backend domain: %q", got)
	}
	if got := fr.Get("%[1]s empty", "email"); got != "email vide" {
		t.Errorf("formatted translation = %q", got)
	}
	// Arguments are data: braces and percent signs must not trigger further translation.
	if got := fr.Get("%[1]s empty", "{name} 100%"); got != "{name} 100% vide" {
		t.Errorf("argument was reinterpreted: %q", got)
	}
	if got := fr.Get("Untranslated %s", "message"); got != "Untranslated message" {
		t.Errorf("English fallback = %q", got)
	}
	en, err := New("en")
	if err != nil {
		t.Fatal(err)
	}
	if len(en.Domains) != 0 {
		t.Fatal("English should use source messages without catalogs")
	}
	for _, n := range []int{0, 1, 2} {
		want := fmt.Sprintf("%d results", n)
		if n == 1 {
			want = "1 result"
		}
		if got := en.GetND("templates", "%d result", "%d results", n, n); got != want {
			t.Errorf("English n=%d: %q, want %q", n, got, want)
		}
		want = fmt.Sprintf("%d résultat", n)
		if n > 1 {
			want += "s"
		}
		if got := fr.GetND("templates", "%d result", "%d results", n, n); got != want {
			t.Errorf("French n=%d: %q, want %q", n, got, want)
		}
	}
}

func TestCatalogsAndConcurrentLocales(t *testing.T) {
	files, err := fs.Glob(catalogs.LocaleFiles, "*/LC_MESSAGES/backend.po")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no backend source catalogs embedded")
	}
	// Frontend-only locales (including English) do not have backend catalogs.
	for _, filename := range files {
		code := strings.Split(filename, "/")[0]
		t.Run(code, func(t *testing.T) {
			t.Parallel()
			locale, err := New(code)
			if err != nil {
				t.Fatal(err)
			}
			for _, domain := range []string{"backend", "templates"} {
				if locale.Domains[domain] == nil {
					t.Fatalf("missing domain %s", domain)
				}
				po := gotext.NewPo()
				b, err := catalogs.LocaleFiles.ReadFile(code + "/LC_MESSAGES/" + domain + ".po")
				if err != nil {
					t.Fatal(err)
				}
				po.Parse(b)
				for id, translation := range po.GetDomain().GetTranslations() {
					// Check the loader against source translations, whether it used
					// a compiled MO or the development PO fallback.
					if got, want := locale.GetD(domain, id, []any{}...), po.Get(id, []any{}...); got != want {
						t.Errorf("%s: %q = %q, want %q", domain, id, got, want)
					}
					if translation.PluralID != "" {
						for _, n := range []int{0, 1, 2, 5, 11, 21, 100} {
							got := locale.GetND(domain, id, translation.PluralID, n, []any{}...)
							want := po.GetN(id, translation.PluralID, n, []any{}...)
							if got != want {
								t.Errorf("%s: %q n=%d = %q, want %q", domain, id, n, got, want)
							}
						}
					}
				}
			}
			got := locale.GetND("templates", "%d result", "%d results", 5, 5)
			if strings.Contains(got, "%!") || !strings.Contains(got, "5") {
				t.Errorf("bad plural formatting: %q", got)
			}
		})
	}
}
