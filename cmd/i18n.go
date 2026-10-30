package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/locales"
	catalogs "github.com/abhinavxd/libredesk/locales"
	"github.com/leonelquinteros/gotext"
	"github.com/zerodha/fastglue"
	"golang.org/x/text/language"
)

const (
	defLang = "en-US"
)

var (
	localeI18nMu    sync.Mutex
	localeI18nCache = map[string]*gotext.Locale{}

	// Keyed by resolved language code
	i18nJSONMu    sync.Mutex
	i18nJSONCache = map[string][]byte{}
)

// handleGetI18nLang returns the JSON language pack for the given language code.
func handleGetI18nLang(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		lang = r.RequestCtx.UserValue("lang").(string)
	)
	b, err := i18nLangJSON(app, lang)
	if err != nil {
		app.lo.Error("error loading frontend language", "language", lang, "error", err)
		return sendErrorEnvelope(r, err)
	}
	r.RequestCtx.Response.Header.SetContentType("application/json")
	r.RequestCtx.Response.SetStatusCode(http.StatusOK)
	r.RequestCtx.Response.SetBodyRaw(b)
	return nil
}

// handleGetAvailableLanguages lists the compiled frontend catalogs.
func handleGetAvailableLanguages(r *fastglue.Request) error {
	app := r.Context.(*App)
	files, err := fs.Glob(catalogs.LocaleFiles, "*/LC_MESSAGES/frontend.json")
	if err != nil {
		app.lo.Error("error listing language files", "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, "error listing language files", nil))
	}

	type langInfo struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}

	var langs []langInfo
	for _, f := range files {
		code := strings.Split(f, "/")[0]
		b, err := catalogs.LocaleFiles.ReadFile(f)
		if err != nil {
			app.lo.Warn("error reading available language", "language", code, "error", err)
			continue
		}

		var meta frontendCatalog
		if err := json.Unmarshal(b, &meta); err != nil {
			app.lo.Warn("error parsing available language", "language", code, "error", err)
			continue
		}

		name := meta.Name
		if name == "" {
			name = code
		}
		langs = append(langs, langInfo{Code: code, Name: name})
	}

	sort.Slice(langs, func(i, j int) bool {
		return langs[i].Name < langs[j].Name
	})

	return r.SendEnvelope(langs)
}

// localeI18n returns an i18n instance for the given locale code, matching available
// language packs by exact code or language prefix, falling back to the app's i18n.
func localeI18n(app *App, locale string) *gotext.Locale {
	loc := strings.ToLower(strings.TrimSpace(locale))
	localeI18nMu.Lock()
	defer localeI18nMu.Unlock()
	if i, ok := localeI18nCache[loc]; ok {
		return i
	}
	i, err := locales.New(loc)
	if err != nil {
		app.lo.Warn("selected language unavailable, using default language", "language", loc, "error", err)
	}
	if err != nil || i == nil {
		i = app.i18n
	}
	localeI18nCache[loc] = i
	return i
}

// frontendCatalog carries compiled Lingui messages and a separate English fallback.
type frontendCatalog struct {
	Locale   string          `json:"locale"`
	Name     string          `json:"name"`
	Messages json.RawMessage `json:"messages"`
	English  json.RawMessage `json:"english,omitempty"`
}

// i18nLangJSON returns the language pack, cached per resolved language code.
func i18nLangJSON(_ *App, lang string) ([]byte, error) {
	code := matchLangFile(catalogs.LocaleFiles, strings.ToLower(strings.TrimSpace(lang)))
	i18nJSONMu.Lock()
	defer i18nJSONMu.Unlock()
	if b, ok := i18nJSONCache[code]; ok {
		return b, nil
	}
	// English is kept separate so the client can apply English plural rules to
	// untranslated messages, regardless of the selected interface language.
	b, err := catalogs.LocaleFiles.ReadFile(defLang + "/LC_MESSAGES/frontend.json")
	if err != nil {
		return nil, fmt.Errorf("load frontend catalogs (run make i18n-compile before building): %w", err)
	}
	var english frontendCatalog
	if err := json.Unmarshal(b, &english); err != nil {
		return nil, err
	}
	selected, err := catalogs.LocaleFiles.ReadFile(code + "/LC_MESSAGES/frontend.json")
	if err != nil {
		return nil, err
	}
	var catalog frontendCatalog
	if err := json.Unmarshal(selected, &catalog); err != nil {
		return nil, err
	}
	catalog.English = english.Messages
	b, err = json.Marshal(catalog)
	if err != nil {
		return nil, err
	}
	i18nJSONCache[code] = b
	return b, nil
}

func matchLangFile(filesystem fs.FS, loc string) string {
	wanted, err := language.All.Parse(strings.TrimSpace(loc))
	if err != nil || wanted.IsRoot() {
		return defLang
	}
	files, err := fs.Glob(filesystem, "*/LC_MESSAGES/frontend.json")
	if err != nil {
		return defLang
	}
	wantedBase, _ := wanted.Base()
	wantedScript, _ := wanted.Script()
	prefixMatch := ""
	for _, f := range files {
		code := strings.Split(f, "/")[0]
		available, err := language.All.Parse(code)
		if err != nil {
			continue
		}
		if available.String() == wanted.String() {
			return code
		}
		base, _ := available.Base()
		script, _ := available.Script()
		if prefixMatch == "" && base == wantedBase && script == wantedScript {
			prefixMatch = code
		}
	}
	if prefixMatch != "" {
		return prefixMatch
	}
	return defLang
}
