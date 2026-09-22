package helpcenter

import (
	"encoding/json"
	"testing"

	"github.com/abhinavxd/libredesk/internal/helpcenter/models"
)

func TestSanitizeAssetURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"absolute https", "https://example.com/a.png", "https://example.com/a.png"},
		{"absolute http", "http://example.com/a.png", "http://example.com/a.png"},
		{"root relative", "/uploads/a.png", "/uploads/a.png"},
		{"site root", "/", "/"},
		{"scheme only", "https://", ""},
		{"trims space", "  /uploads/a.png  ", "/uploads/a.png"},
		{"protocol relative", "//evil.com/a.png", ""},
		{"javascript", "javascript:alert(1)", ""},
		{"data uri", "data:image/png;base64,AAA", ""},
		{"bare host", "example.com/a.png", ""},
		{"css escape", "/a.png\") ; body{display:none", ""},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeAssetURL(tt.input); got != tt.want {
				t.Errorf("sanitizeAssetURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitizeNavLinks(t *testing.T) {
	got := sanitizeNavLinks([]models.NavLink{
		{Label: "Home", URL: "/"},
		{Label: "", URL: "https://example.com"},
		{Label: "  ", URL: "https://example.com"},
		{Label: "Bad", URL: "javascript:alert(1)"},
		{Label: "Offsite", URL: "//evil.com"},
		{Label: " Docs ", URL: "https://example.com/docs"},
	})
	want := []models.NavLink{
		{Label: "Home", URL: "/"},
		{Label: "Docs", URL: "https://example.com/docs"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d links, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("link %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestSanitizeSocialLinks(t *testing.T) {
	got := sanitizeSocialLinks([]models.SocialLink{
		{Platform: "github", URL: "https://github.com/x"},
		{Platform: "myspace", URL: "https://myspace.com/x"},
		{Platform: "twitter", URL: "javascript:alert(1)"},
	})
	want := []models.SocialLink{
		{Platform: "github", URL: "https://github.com/x"},
		{Platform: "website", URL: "https://myspace.com/x"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d links, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("link %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestNormalizeThemeRejectsUnreadableTheme(t *testing.T) {
	if _, err := normalizeTheme(json.RawMessage(`{"layout":{"columns":"3"}}`)); err == nil {
		t.Error("a theme with a mistyped field must be rejected, not silently blanked")
	}
	if _, err := normalizeTheme(json.RawMessage(`not json`)); err == nil {
		t.Error("invalid JSON must be rejected")
	}
}

func TestNormalizeThemeDropsUnsafeValues(t *testing.T) {
	raw, err := normalizeTheme(json.RawMessage(`{
		"color": "red",
		"logo_url": "javascript:alert(1)",
		"header": {"background_type": "wat", "background_color": "notacolor"},
		"layout": {"collections": "wat", "columns": 5},
		"cards": {"icon_position": "wat"},
		"announcement": {"text": "", "link_url": "https://example.com", "link_label": "Go"}
	}`))
	if err != nil {
		t.Fatalf("normalizeTheme: %v", err)
	}
	var got models.Theme
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Color != defaultAccentColor {
		t.Errorf("color = %q, want %q", got.Color, defaultAccentColor)
	}
	if got.LogoURL != "" {
		t.Errorf("logo_url = %q, want empty", got.LogoURL)
	}
	if got.Header.BackgroundType != "" || got.Header.BackgroundColor != "" {
		t.Errorf("header = %+v, want blanked", got.Header)
	}
	if got.Layout.Collections != "" || got.Layout.Columns != 0 {
		t.Errorf("layout = %+v, want blanked", got.Layout)
	}
	if got.Cards.IconPosition != cardIconPositions[0] {
		t.Errorf("icon_position = %q, want %q", got.Cards.IconPosition, cardIconPositions[0])
	}
	if got.Announcement != (models.AnnouncementTheme{}) {
		t.Errorf("announcement = %+v, want cleared when the text is empty", got.Announcement)
	}
}

func TestNormalizeThemeColorScheme(t *testing.T) {
	tests := []struct {
		name          string
		raw           string
		wantScheme    string
		wantColorDark string
		wantLogoDark  string
	}{
		{"no theme saved", ``, models.ColorSchemeLight, defaultAccentColor, ""},
		{"null theme", `null`, models.ColorSchemeLight, defaultAccentColor, ""},
		{"saved before dark mode", `{"color": "#112233", "logo_url": "/uploads/a.png"}`, models.ColorSchemeLight, "#112233", ""},
		{"unknown scheme", `{"color_scheme": "auto"}`, models.ColorSchemeLight, defaultAccentColor, ""},
		{"system kept", `{"color_scheme": "system", "color_dark": "#445566"}`, models.ColorSchemeSystem, "#445566", ""},
		{"dark kept", `{"color_scheme": "dark"}`, models.ColorSchemeDark, defaultAccentColor, ""},
		{"bad dark color", `{"color": "#112233", "color_dark": "blue"}`, models.ColorSchemeLight, "#112233", ""},
		{"dark logo kept", `{"logo_url_dark": "/uploads/dark.png"}`, models.ColorSchemeLight, defaultAccentColor, "/uploads/dark.png"},
		{"unsafe dark logo", `{"logo_url_dark": "javascript:alert(1)"}`, models.ColorSchemeLight, defaultAccentColor, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, err := normalizeTheme(json.RawMessage(tt.raw))
			if err != nil {
				t.Fatalf("normalizeTheme: %v", err)
			}
			var got models.Theme
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if got.ColorScheme != tt.wantScheme || got.ColorDark != tt.wantColorDark || got.LogoURLDark != tt.wantLogoDark {
				t.Errorf("got scheme %q, dark color %q, dark logo %q; want %q, %q, %q", got.ColorScheme, got.ColorDark, got.LogoURLDark, tt.wantScheme, tt.wantColorDark, tt.wantLogoDark)
			}
		})
	}
}

func TestNormalizeLocales(t *testing.T) {
	tests := []struct {
		name          string
		locales       []string
		defaultLocale string
		want          []string
	}{
		{"default first", []string{"es", "en"}, "en", []string{"en", "es"}},
		{"dedupes", []string{"en", "en", "es"}, "en", []string{"en", "es"}},
		{"trims", []string{" es "}, "en", []string{"en", "es"}},
		{"drops blanks", []string{"", "es"}, "en", []string{"en", "es"}},
		{"adds missing default", []string{"es"}, "fr", []string{"fr", "es"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeLocales(tt.locales, tt.defaultLocale)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("got %v, want %v", got, tt.want)
				}
			}
		})
	}
}
