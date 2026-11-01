// Package locales loads the backend and server-rendered template Gettext domains.
// English messages live in source code; only translated languages need catalogs.
package locales

import (
	"fmt"
	"io/fs"
	"path"
	"strings"

	catalogs "github.com/abhinavxd/libredesk/locales"
	"github.com/leonelquinteros/gotext"
	"golang.org/x/text/language"
)

// New returns an independent locale with backend as its default domain. Catalogs
// are embedded in ordinary Go builds as well as in the distributed binary.
func New(code string) (*gotext.Locale, error) {
	resolved := resolve(code)
	locale := gotext.NewLocale("", strings.ReplaceAll(resolved, "-", "_"))
	locale.SetDomain("backend")
	if resolved == "en-US" {
		return locale, nil
	}
	for _, domain := range []string{"backend", "templates"} {
		base := path.Join(resolved, "LC_MESSAGES", domain)
		// Prefer compiled catalogs. PO remains usable during development.
		if b, err := catalogs.LocaleFiles.ReadFile(base + ".mo"); err == nil {
			tr := gotext.NewMo()
			tr.Parse(b)
			locale.AddTranslator(domain, tr)
		} else if b, err := catalogs.LocaleFiles.ReadFile(base + ".po"); err == nil {
			tr := gotext.NewPo()
			tr.Parse(b)
			locale.AddTranslator(domain, tr)
		} else {
			return nil, fmt.Errorf("loading %s Gettext catalog for %s: %w", domain, resolved, err)
		}
	}
	return locale, nil
}

// ReloadCatalogs updates the shared locale in place so existing managers and
// template closures see the new translations. Load everything before taking
// the locale lock; readers observe either the old or the new catalog set.
// gotext's construction-time language is only used by its filesystem loader;
// our embedded translators carry their own language and plural rules.
func ReloadCatalogs(locale *gotext.Locale, code string) error {
	next, err := New(code)
	if err != nil {
		return err
	}
	locale.Lock()
	locale.Domains = next.Domains
	locale.Unlock()
	return nil
}

func resolve(code string) string {
	wanted, err := language.All.Parse(strings.TrimSpace(strings.ReplaceAll(code, "_", "-")))
	if err != nil || wanted.IsRoot() {
		return "en-US"
	}
	base, _ := wanted.Base()
	if base.String() == "en" {
		return "en-US"
	}
	script, _ := wanted.Script()
	fallback := ""
	entries, _ := fs.ReadDir(catalogs.LocaleFiles, ".")
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		tag, err := language.All.Parse(entry.Name())
		if err != nil {
			continue
		}
		if wanted.String() == tag.String() {
			return entry.Name()
		}
		b, _ := tag.Base()
		s, _ := tag.Script()
		if fallback == "" && base == b && script == s {
			fallback = entry.Name()
		}
	}
	if fallback != "" {
		return fallback
	}
	return "en-US"
}
