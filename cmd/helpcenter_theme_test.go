package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	hcmodels "github.com/abhinavxd/libredesk/internal/helpcenter/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"golang.org/x/text/language"
)

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
		catalogDir := filepath.Join(dir, code, "LC_MESSAGES")
		if err := os.MkdirAll(catalogDir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(catalogDir, "frontend.json"), []byte("{}"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	fs := os.DirFS(dir)
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
