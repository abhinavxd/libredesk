package main

import (
	"encoding/json"
	"html/template"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	hcmodels "github.com/abhinavxd/libredesk/internal/helpcenter/models"
	"github.com/alicebob/miniredis/v2"
	"github.com/knadh/stuffbin"
	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp"
	goredis "github.com/zerodha/fastcache/stores/goredis/v9"
	"github.com/zerodha/fastcache/v4"
	"github.com/zerodha/fastglue"
	"golang.org/x/text/language"
)

func TestHelpCenterCustomCodeAllowedHosts(t *testing.T) {
	app := newValidatorTestApp(t)
	tests := []struct {
		name           string
		uri            string
		customDomain   string
		allowAppDomain bool
		wantCode       bool
	}{
		{"app host without custom domain", "https://desk.example.com/hc/docs/en", "", false, false},
		{"app host with custom domain", "https://desk.example.com/hc/docs/en", "https://help.example.com", false, false},
		{"app host configured as custom domain", "https://desk.example.com/en", "https://desk.example.com", false, false},
		{"app host with different scheme", "http://desk.example.com/en", "http://desk.example.com", false, false},
		{"app host with different port", "https://desk.example.com:8443/en", "https://desk.example.com:8443", false, false},
		{"app host case", "https://DESK.EXAMPLE.COM/en", "https://desk.example.com", false, false},
		{"configured custom domain", "https://help.example.com/en", "https://help.example.com", false, true},
		{"custom domain case", "https://HELP.EXAMPLE.COM/en", "https://help.example.com", false, true},
		{"embedded article on custom domain", "https://help.example.com/en/articles/example?embed=1", "https://help.example.com", false, true},
		{"embedded article on app host", "https://desk.example.com/hc/docs/en/articles/example?embed=1", "https://help.example.com", false, false},
		{"other host", "https://other.example.com/en", "https://help.example.com", false, false},
		{"unconfigured host", "https://help.example.com/en", "", false, false},
		{"invalid custom domain", "https://help.example.com/en", "help.example.com", false, false},
		{"app host allowed by config", "https://desk.example.com/hc/docs/en", "", true, true},
		{"app host with custom domain allowed by config", "https://desk.example.com/hc/docs/en", "https://help.example.com", true, true},
		{"custom domain allowed by config", "https://help.example.com/en", "https://help.example.com", true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ko.Delete(hcAppDomainCustomCodeKey)
			if tt.allowAppDomain {
				if err := ko.Set(hcAppDomainCustomCodeKey, true); err != nil {
					t.Fatalf("set config: %v", err)
				}
			}
			t.Cleanup(func() { ko.Delete(hcAppDomainCustomCodeKey) })

			r := request(tt.uri, "" /** ifNoneMatch **/)
			hc := hcmodels.HelpCenter{
				Slug:         "docs",
				CustomDomain: tt.customDomain,
				CustomCSS:    `</style><script>window.cssExecuted = true</script>`,
				CustomJS:     `window.jsExecuted = true`,
			}
			data := helpCenterTemplateData(app, r, hc, "en")
			wantCSS, wantJS := "", ""
			if tt.wantCode {
				wantCSS, wantJS = hc.CustomCSS, hc.CustomJS
			}
			if got := string(data["CustomCSS"].(template.CSS)); got != wantCSS {
				t.Errorf("CustomCSS = %q, want %q", got, wantCSS)
			}
			if got := string(data["CustomJS"].(template.JS)); got != wantJS {
				t.Errorf("CustomJS = %q, want %q", got, wantJS)
			}
		})
	}
}

func TestHelpCenterPreviewShowsCSSWithoutJS(t *testing.T) {
	app := newValidatorTestApp(t)
	for _, allowAppDomain := range []bool{false, true} {
		ko.Delete(hcAppDomainCustomCodeKey)
		if allowAppDomain {
			if err := ko.Set(hcAppDomainCustomCodeKey, true); err != nil {
				t.Fatalf("set config: %v", err)
			}
		}
		for _, uri := range []string{"https://desk.example.com/api/v1/help-centers/1/preview", "https://help.example.com/api/v1/help-centers/1/preview"} {
			hc := hcmodels.HelpCenter{
				Slug:         "docs",
				CustomDomain: "https://help.example.com",
				CustomCSS:    `body { color: red }`,
				CustomJS:     `window.jsExecuted = true`,
			}
			data := helpCenterPreviewData(app, request(uri, "" /** ifNoneMatch **/), hc, "en")
			if got := string(data["CustomCSS"].(template.CSS)); got != hc.CustomCSS {
				t.Errorf("%s (allowAppDomain=%v): CustomCSS = %q, want %q", uri, allowAppDomain, got, hc.CustomCSS)
			}
			if got := string(data["CustomJS"].(template.JS)); got != "" {
				t.Errorf("%s (allowAppDomain=%v): CustomJS = %q, want empty", uri, allowAppDomain, got)
			}
		}
	}
	ko.Delete(hcAppDomainCustomCodeKey)
}

func TestResolveQueryLocale(t *testing.T) {
	tests := []struct {
		name          string
		requested     string
		allowed       string
		defaultLocale string
		want          string
	}{
		{"exact language", "de", `["en","de"]`, "en", "de"},
		{"widget region matches base", "de-DE", `["en","de"]`, "en", "de"},
		{"exact region wins", "de-AT", `["en","de","de-AT"]`, "en", "de-AT"},
		{"region falls back to base", "de-CH", `["en","de-AT","de"]`, "en", "de"},
		{"other region matches", "de-DE", `["en","de-AT"]`, "en", "de-AT"},
		{"base matches regional English", "en", `["de","en-GB"]`, "de", "en-GB"},
		{"base matches regional Portuguese", "pt", `["en","pt-BR"]`, "en", "pt-BR"},
		{"regional English matches another region", "en-GB", `["de","en-US"]`, "de", "en-US"},
		{"exact script wins", "zh-Hant", `["en","zh","zh-Hant"]`, "en", "zh-Hant"},
		{"simplified Chinese region", "zh-CN", `["en","zh-Hans","zh-Hant"]`, "en", "zh-Hans"},
		{"traditional Chinese region", "zh-TW", `["en","zh-Hans","zh-Hant"]`, "en", "zh-Hant"},
		{"Hong Kong Chinese", "zh-HK", `["en","zh-Hans","zh-Hant"]`, "en", "zh-Hant"},
		{"Singapore Chinese", "zh-SG", `["en","zh-Hans","zh-Hant"]`, "en", "zh-Hans"},
		{"explicit Chinese script", "zh-Hant-TW", `["en","zh-Hans","zh-Hant"]`, "en", "zh-Hant"},
		{"wrong Chinese script is not selected", "zh-TW", `["en","zh-Hans"]`, "en", "en"},
		{"wrong Serbian script is not selected", "sr-Latn-RS", `["en","sr"]`, "en", "en"},
		{"canonical case", "DE-de", `["en","de"]`, "en", "de"},
		{"legacy separator", "pt_PT", `["en","pt"]`, "en", "pt"},
		{"Norwegian browser language", "nb-NO", `["en","no"]`, "en", "no"},
		{"UK region alias", "en-UK", `["en","en-GB"]`, "en", "en-GB"},
		{"legacy Hebrew language", "iw-IL", `["en","he"]`, "en", "he"},
		{"explicit script overrides region", "zh-Hans-TW", `["en","zh-Hant","zh-Hans"]`, "en", "zh-Hans"},
		{"redundant script retains exact region", "fr-Latn-CA", `["en","fr","fr-CA"]`, "en", "fr-CA"},
		{"numeric region", "es-419", `["en","es"]`, "en", "es"},
		{"unspecified language", "und", `["en","fr"]`, "fr", "fr"},
		{"unsupported language", "xx", `["en","de"]`, "en", "en"},
		{"empty request", "", `["en","de"]`, "de", "de"},
		{"whitespace request", "  ", `["en","de"]`, "de", "de"},
		{"trimmed region", " de-DE ", `["en","de"]`, "en", "de"},
		{"duplicate locales", "de-DE", `["en","de","de"]`, "en", "de"},
		{"empty locales", "de-DE", `[]`, "en", "en"},
		{"missing locales", "de-DE", "", "en", "en"},
		{"invalid locales", "de-DE", `{`, "en", "en"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &fastglue.Request{RequestCtx: &fasthttp.RequestCtx{}}
			r.RequestCtx.QueryArgs().Set("locale", tt.requested)
			hc := hcmodels.HelpCenter{
				DefaultLocale:  tt.defaultLocale,
				AllowedLocales: json.RawMessage(tt.allowed),
			}
			if got := resolveQueryLocale(r, hc); got != tt.want {
				t.Errorf("resolveQueryLocale(%q) = %q, want %q", tt.requested, got, tt.want)
			}
		})
	}
}

func TestResolveQueryLocaleFallback(t *testing.T) {
	for _, tt := range []struct {
		name      string
		requested string
		fallback  string
		allowed   string
		want      string
	}{
		{"unsupported primary uses widget language", "ko-KR", "de-DE", `["en","de"]`, "de"},
		{"exact primary wins", "fr-CA", "de-DE", `["en","fr-CA","de"]`, "fr-CA"},
		{"base primary wins", "fr-CA", "de-DE", `["en","fr","de"]`, "fr"},
		{"regional primary wins", "fr-CA", "de-DE", `["en","fr-BE","de"]`, "fr-BE"},
		{"empty primary uses widget language", "", "de-DE", `["en","de"]`, "de"},
		{"invalid primary uses widget language", "en--GB", "de-DE", `["en","de"]`, "de"},
		{"unsupported preferences use default", "ko-KR", "ja-JP", `["en","de"]`, "en"},
		{"wrong writing system uses default", "zh-TW", "sr-Latn-RS", `["en","zh-Hans","sr"]`, "en"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := &fastglue.Request{RequestCtx: &fasthttp.RequestCtx{}}
			r.RequestCtx.QueryArgs().Set("locale", tt.requested)
			r.RequestCtx.QueryArgs().Set("fallback_locale", tt.fallback)
			hc := hcmodels.HelpCenter{DefaultLocale: "en", AllowedLocales: json.RawMessage(tt.allowed)}
			if got := resolveQueryLocale(r, hc); got != tt.want {
				t.Errorf("resolveQueryLocale(%q, %q) = %q, want %q", tt.requested, tt.fallback, got, tt.want)
			}
		})
	}
}

func FuzzResolveQueryLocale(f *testing.F) {
	f.Add("zh-Hant-HK", `["en","zh-Hant"]`, "en")
	f.Add("fr-CH", `["en","fr"]`, "en")
	f.Add("EN-us", `["en","en-US"]`, "en")
	f.Add("sr-Latn", `["en","sr"]`, "en")
	f.Fuzz(func(t *testing.T, requested, allowed, defaultLocale string) {
		r := &fastglue.Request{RequestCtx: &fasthttp.RequestCtx{}}
		r.RequestCtx.QueryArgs().Set("locale", requested)
		hc := hcmodels.HelpCenter{
			DefaultLocale:  defaultLocale,
			AllowedLocales: json.RawMessage(allowed),
		}
		got := resolveQueryLocale(r, hc)
		if got == hc.DefaultLocale {
			return
		}
		if !slices.Contains(helpCenterLocales(hc), got) {
			t.Fatalf("returned unconfigured locale %q", got)
		}
		wanted, err := language.All.Parse(strings.TrimSpace(requested))
		if err != nil {
			t.Fatalf("matched invalid locale %q to %q", requested, got)
		}
		matched, err := language.All.Parse(got)
		if err != nil {
			t.Fatalf("returned invalid locale %q", got)
		}
		wantedBase, _ := wanted.Base()
		matchedBase, _ := matched.Base()
		wantedScript, _ := wanted.Script()
		matchedScript, _ := matched.Script()
		if wantedBase != matchedBase || wantedScript != matchedScript {
			t.Fatalf("matched %q to a different language or script %q", requested, got)
		}
	})
}

func TestMatchLangFile(t *testing.T) {
	dir := t.TempDir()
	for _, code := range []string{"en-US", "de-DE", "pt-BR", "pt-PT", "es-ES", "zh-CN"} {
		if err := os.WriteFile(filepath.Join(dir, code+".json"), []byte("{}"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	fs, err := stuffbin.NewLocalFS("/", dir+":i18n")
	if err != nil {
		t.Fatalf("creating test filesystem: %v", err)
	}
	for _, tt := range []struct {
		requested string
		want      string
	}{
		{"de-AT", "de-DE"},
		{"DE-de", "de-DE"},
		{"pt-PT", "pt-PT"},
		{"pt_PT", "pt-PT"},
		{"zh-Hans", "zh-CN"},
		{"zh-SG", "zh-CN"},
		{"zh-TW", "en-US"},
		{"zh-Hant", "en-US"},
		{"es-419", "es-ES"},
		{"zh-Hans-TW", "zh-CN"},
		{"und", "en-US"},
		{"xx", "en-US"},
	} {
		t.Run(tt.requested, func(t *testing.T) {
			if got := matchLangFile(fs, tt.requested); got != tt.want {
				t.Errorf("matchLangFile(%q) = %q, want %q", tt.requested, got, tt.want)
			}
		})
	}
}

func TestResolveColorScheme(t *testing.T) {
	const (
		system = hcmodels.ColorSchemeSystem
		light  = hcmodels.ColorSchemeLight
		dark   = hcmodels.ColorSchemeDark
	)
	tests := []struct {
		name      string
		scheme    string
		requested string
		embed     bool
		want      colorSchemeResult
	}{
		{"light site", light, "", false, colorSchemeResult{}},
		{"dark site", dark, "", false, colorSchemeResult{dark: true}},
		{"system site", system, "", false, colorSchemeResult{showToggle: true, followSystem: true}},
		{"preview dark of a light site", light, dark, false, colorSchemeResult{dark: true}},
		{"preview light of a dark site", dark, light, false, colorSchemeResult{}},
		{"preview of a system site keeps the toggle", system, dark, false, colorSchemeResult{dark: true, showToggle: true}},
		{"embed in a light widget on a dark site", dark, "", true, colorSchemeResult{}},
		{"embed in a dark widget", light, dark, true, colorSchemeResult{dark: true}},
		{"embed on a system site", system, dark, true, colorSchemeResult{dark: true}},
		{"unknown requested value", dark, "purple", false, colorSchemeResult{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveColorScheme(tt.scheme, tt.requested, tt.embed); got != tt.want {
				t.Errorf("resolveColorScheme(%q, %q, %v) = %+v, want %+v", tt.scheme, tt.requested, tt.embed, got, tt.want)
			}
		})
	}
}

func TestBuildDarkThemeCSSVars(t *testing.T) {
	tests := []struct {
		name   string
		header hcmodels.HeaderTheme
		footer hcmodels.FooterTheme
		want   string
	}{
		{"nothing set", hcmodels.HeaderTheme{}, hcmodels.FooterTheme{}, ""},
		{"header text on default header", hcmodels.HeaderTheme{TextColor: "#000"}, hcmodels.FooterTheme{}, "--hc-header-text:initial;"},
		{"header text on solid header", hcmodels.HeaderTheme{BackgroundType: "solid", BackgroundColor: "#fff", TextColor: "#000"}, hcmodels.FooterTheme{}, ""},
		{"header text on image header", hcmodels.HeaderTheme{BackgroundType: "image", BackgroundImage: "/a.png", TextColor: "#000"}, hcmodels.FooterTheme{}, ""},
		{"header text on half-set gradient", hcmodels.HeaderTheme{BackgroundType: "gradient", GradientFrom: "#fff", TextColor: "#000"}, hcmodels.FooterTheme{}, "--hc-header-text:initial;"},
		{"footer text on default footer", hcmodels.HeaderTheme{}, hcmodels.FooterTheme{TextColor: "#000"}, "--hc-footer-text:initial;"},
		{"light footer colors only", hcmodels.HeaderTheme{}, hcmodels.FooterTheme{BackgroundColor: "#fff", TextColor: "#000"}, "--hc-footer-bg:initial;--hc-footer-text:initial;"},
		{"dark footer colors", hcmodels.HeaderTheme{}, hcmodels.FooterTheme{BackgroundColor: "#fff", TextColor: "#000", BackgroundColorDark: "#111", TextColorDark: "#eee"}, "--hc-footer-bg:#111;--hc-footer-text:#eee;"},
		{"dark footer text only", hcmodels.HeaderTheme{}, hcmodels.FooterTheme{TextColorDark: "#eee"}, "--hc-footer-text:#eee;"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(buildDarkThemeCSSVars(hcmodels.Theme{Header: tt.header, Footer: tt.footer}))
			if got != tt.want {
				t.Errorf("buildDarkThemeCSSVars() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHelpCenterPageCacheServesAndInvalidates(t *testing.T) {
	fc := newTestFastCache(t)

	renders := 0
	body := "<h1>original</h1>"
	handler := func(r *fastglue.Request) error {
		renders++
		r.RequestCtx.SetContentType("text/html; charset=utf-8")
		r.RequestCtx.Response.Header.Set("Cache-Control", helpCenterCacheControl)
		r.RequestCtx.Response.SetBodyString(body)
		return nil
	}
	cached := fc.Cached(handler, helpCenterCacheOpts, helpCenterCacheGroup)

	const uri = "/hc/support/en/articles/refunds"

	first := request(uri, "")
	if err := cached(first); err != nil {
		t.Fatalf("first request: %v", err)
	}
	if renders != 1 {
		t.Fatalf("first request should render, renders = %d", renders)
	}
	etag := string(first.RequestCtx.Response.Header.Peek("ETag"))
	if etag == "" {
		t.Fatal("expected an ETag on the rendered page")
	}

	second := request(uri, "")
	if err := cached(second); err != nil {
		t.Fatalf("second request: %v", err)
	}
	if renders != 1 {
		t.Errorf("second request re-rendered instead of using the cache, renders = %d", renders)
	}
	if got := string(second.RequestCtx.Response.Body()); got != body {
		t.Errorf("cached body = %q, want %q", got, body)
	}

	revalidate := request(uri, etag)
	if err := cached(revalidate); err != nil {
		t.Fatalf("revalidation: %v", err)
	}
	if got := revalidate.RequestCtx.Response.StatusCode(); got != fasthttp.StatusNotModified {
		t.Errorf("matching ETag returned %d, want 304", got)
	}
	if got := revalidate.RequestCtx.Response.Body(); len(got) != 0 {
		t.Errorf("304 carried a body of %d bytes", len(got))
	}

	if err := fc.DelGroup(helpCenterCacheNamespace, helpCenterCacheGroup); err != nil {
		t.Fatalf("clearing the group: %v", err)
	}
	body = "<h1>edited</h1>"
	afterEdit := request(uri, "")
	if err := cached(afterEdit); err != nil {
		t.Fatalf("request after edit: %v", err)
	}
	if renders != 2 {
		t.Errorf("cleared cache did not re-render, renders = %d", renders)
	}
	if got := string(afterEdit.RequestCtx.Response.Body()); got != body {
		t.Errorf("stale body served after an edit: got %q, want %q", got, body)
	}
	if newETag := string(afterEdit.RequestCtx.Response.Header.Peek("ETag")); newETag == etag {
		t.Error("ETag survived an edit, readers holding the old copy would never refetch")
	}
}

func TestHelpCenterPageCacheKeysPerQueryAndSurvivesRename(t *testing.T) {
	fc := newTestFastCache(t)

	served := map[string]int{}
	handler := func(r *fastglue.Request) error {
		uri := string(r.RequestCtx.URI().RequestURI())
		served[uri]++
		r.RequestCtx.Response.SetBodyString(uri)
		return nil
	}
	cached := fc.Cached(handler, helpCenterCacheOpts, helpCenterCacheGroup)

	for _, uri := range []string{
		"/hc/support/en/search?q=refund",
		"/hc/support/en/search?q=billing",
		"/hc/docs/en/search?q=refund",
	} {
		if err := cached(request(uri, "")); err != nil {
			t.Fatalf("request: %v", err)
		}
	}
	if len(served) != 3 {
		t.Fatalf("expected 3 distinct cache entries, got %d: %v", len(served), served)
	}

	clearAll(t, fc)
	for uri, before := range served {
		if err := cached(request(uri, "")); err != nil {
			t.Fatalf("request: %v", err)
		}
		if served[uri] == before {
			t.Errorf("%s still served from cache after a clear", uri)
		}
	}
}

func TestHelpCenterPageCacheReportsHitOrMiss(t *testing.T) {
	app := &App{fc: newTestFastCache(t)}
	handler := cachedHCPage(func(r *fastglue.Request) error {
		r.RequestCtx.Response.SetBodyString("<h1>page</h1>")
		return nil
	})

	const uri = "/hc/support/en/articles/refunds"
	run := func(ifNoneMatch string) *fasthttp.RequestCtx {
		req := request(uri, ifNoneMatch)
		req.Context = app
		if err := handler(req); err != nil {
			t.Fatalf("request: %v", err)
		}
		return req.RequestCtx
	}

	first := run("")
	if got := string(first.Response.Header.Peek("X-Cache")); got != "MISS" {
		t.Errorf("first request X-Cache = %q, want MISS", got)
	}

	second := run("")
	if got := string(second.Response.Header.Peek("X-Cache")); got != "HIT" {
		t.Errorf("second request X-Cache = %q, want HIT", got)
	}

	revalidated := run(string(first.Response.Header.Peek("ETag")))
	if got := revalidated.Response.StatusCode(); got != fasthttp.StatusNotModified {
		t.Fatalf("revalidation returned %d, want 304", got)
	}
	if got := string(revalidated.Response.Header.Peek("X-Cache")); got != "HIT" {
		t.Errorf("304 X-Cache = %q, want HIT", got)
	}
}

func TestHelpCenterPageCacheKeepsNoIndexOnHits(t *testing.T) {
	app := &App{fc: newTestFastCache(t)}
	body := func(r *fastglue.Request) error {
		r.RequestCtx.Response.SetBodyString("raw markdown")
		return nil
	}

	cases := []struct {
		name    string
		handler fastglue.FastRequestHandler
		uri     string
		slug    string
	}{
		{"search results", cachedHCNoIndexPage(body), "/hc/support/en/search?q=refund", ""},
		{"markdown article", cachedHCPage(body), "/hc/support/en/articles/refunds.md", "refunds.md"},
		{"html article", cachedHCPage(body), "/hc/support/en/articles/refunds", "refunds"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			want := tc.slug != "refunds"
			for _, pass := range []string{"miss", "hit"} {
				req := request(tc.uri, "")
				req.Context = app
				if tc.slug != "" {
					req.RequestCtx.SetUserValue("article_slug", tc.slug)
				}
				if err := tc.handler(req); err != nil {
					t.Fatalf("%s: %v", pass, err)
				}
				got := string(req.RequestCtx.Response.Header.Peek("X-Robots-Tag")) == noIndexHeader
				if got != want {
					t.Errorf("%s: noindex = %v, want %v", pass, got, want)
				}
			}
		})
	}
}

func newTestFastCache(t *testing.T) *fastcache.FastCache {
	t.Helper()
	mr := miniredis.RunT(t)
	return fastcache.New(goredis.New(goredis.Config{Prefix: fastCachePrefix}, redis.NewClient(&redis.Options{Addr: mr.Addr()})))
}

func request(uri, ifNoneMatch string) *fastglue.Request {
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI(uri)
	ctx.SetUserValue(helpCenterCacheNamespaceKey, helpCenterCacheNamespace)
	if ifNoneMatch != "" {
		ctx.Request.Header.Set("If-None-Match", ifNoneMatch)
	}
	return &fastglue.Request{RequestCtx: ctx}
}

func clearAll(t *testing.T, fc *fastcache.FastCache) {
	t.Helper()
	if err := fc.DelGroup(helpCenterCacheNamespace, helpCenterCacheGroup); err != nil {
		t.Fatalf("clearing the group: %v", err)
	}
}

func TestCollectionTrail(t *testing.T) {
	tree := []hcmodels.TreeCollection{
		{Collection: hcmodels.Collection{Slug: "other"}},
		{
			Collection: hcmodels.Collection{Slug: "root"},
			Children: []hcmodels.TreeCollection{{
				Collection: hcmodels.Collection{Slug: "child"},
				Children:   []hcmodels.TreeCollection{{Collection: hcmodels.Collection{Slug: "leaf"}}},
			}},
		},
	}
	for _, tt := range []struct {
		slug string
		want []string
	}{
		{"root", []string{"root"}},
		{"child", []string{"root", "child"}},
		{"leaf", []string{"root", "child", "leaf"}},
		{"other", []string{"other"}},
		{"missing", nil},
		{"", nil},
	} {
		t.Run(tt.slug, func(t *testing.T) {
			var got []string
			for _, c := range collectionTrail(tree, tt.slug) {
				got = append(got, c.Slug)
			}
			if !slices.Equal(got, tt.want) {
				t.Fatalf("collectionTrail(%q) = %v, want %v", tt.slug, got, tt.want)
			}
		})
	}
}
