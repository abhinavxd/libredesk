package livechat

import (
	"encoding/json"
	"testing"
)

func TestResolvePreChatFormFallsBackToLegacyValues(t *testing.T) {
	var config Config
	if err := json.Unmarshal([]byte(`{
		"prechat_form":{
			"enabled":true,
			"title":"Before we start",
			"fields":[{"key":"email","label":"Email"}]
		}
	}`), &config); err != nil {
		t.Fatal(err)
	}

	for _, isVisitor := range []bool{true, false} {
		resolved := config.ResolvePreChatForm(isVisitor)
		if !resolved.PreChatForm.Enabled || resolved.PreChatForm.Title != "Before we start" {
			t.Fatalf("resolved pre-chat form = %+v", resolved.PreChatForm)
		}
		if len(resolved.PreChatForm.Fields) != 1 || resolved.PreChatForm.Fields[0].Key != "email" {
			t.Fatalf("resolved fields = %+v", resolved.PreChatForm.Fields)
		}
	}
}

func TestResolvePreChatFormUsesAudienceValues(t *testing.T) {
	var config Config
	if err := json.Unmarshal([]byte(`{
		"prechat_form":{
			"enabled":true,
			"title":"Legacy title",
			"fields":[{"key":"email"}],
			"visitors":{"enabled":true,"title":"Choose a plan","fields":[{"key":"plan"}]},
			"users":{"enabled":false,"title":"What is the issue?","fields":[]}
		}
	}`), &config); err != nil {
		t.Fatal(err)
	}

	visitor := config.ResolvePreChatForm(true)
	if !visitor.PreChatForm.Enabled || visitor.PreChatForm.Title != "Choose a plan" {
		t.Fatalf("visitor pre-chat form = %+v", visitor.PreChatForm)
	}
	if len(visitor.PreChatForm.Fields) != 1 || visitor.PreChatForm.Fields[0].Key != "plan" {
		t.Fatalf("visitor fields = %+v", visitor.PreChatForm.Fields)
	}
	user := config.ResolvePreChatForm(false)
	if user.PreChatForm.Enabled || user.PreChatForm.Title != "What is the issue?" {
		t.Fatalf("user pre-chat form = %+v", user.PreChatForm)
	}
	if user.PreChatForm.Fields == nil || len(user.PreChatForm.Fields) != 0 {
		t.Fatalf("user fields = %+v", user.PreChatForm.Fields)
	}
}

func TestUnmarshalFillsBrandingFromLegacyConfig(t *testing.T) {
	legacy := []byte(`{
		"dark_mode":true,
		"colors":{"primary":"#2563eb"},
		"logo_url":"https://example.com/a.png",
		"launcher":{"position":"right","logo_url":"https://example.com/l.png","color":"#000000","spacing":{"side":20,"bottom":20}},
		"home_screen":{"header_text_color":"black","background":{"type":"solid","color":"#ffffff"},"fade_background":true}
	}`)

	var config Config
	if err := json.Unmarshal(legacy, &config); err != nil {
		t.Fatal(err)
	}

	if config.Theme != ThemeDark {
		t.Fatalf("theme = %q, want %q", config.Theme, ThemeDark)
	}
	if config.Launcher.Position != "right" || config.Launcher.Spacing.Side != 20 {
		t.Fatalf("launcher layout = %+v", config.Launcher)
	}
	for name, branding := range map[string]Branding{"light": config.Branding.Light, "dark": config.Branding.Dark} {
		if branding.Colors.Primary != "#2563eb" {
			t.Fatalf("%s primary = %q", name, branding.Colors.Primary)
		}
		if branding.LogoURL != "https://example.com/a.png" {
			t.Fatalf("%s logo = %q", name, branding.LogoURL)
		}
		if branding.Launcher.LogoURL != "https://example.com/l.png" || branding.Launcher.Color != "#000000" {
			t.Fatalf("%s launcher branding = %+v", name, branding.Launcher)
		}
		if branding.HomeScreen.Background.Color != "#ffffff" || !branding.HomeScreen.FadeBackground {
			t.Fatalf("%s home screen = %+v", name, branding.HomeScreen)
		}
	}

	normalized, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	var response struct {
		Branding BrandingSet `json:"branding"`
	}
	if err := json.Unmarshal(normalized, &response); err != nil {
		t.Fatal(err)
	}
	if response.Branding.Light.LogoURL != "https://example.com/a.png" || response.Branding.Dark.LogoURL != "https://example.com/a.png" {
		t.Fatalf("response branding = %+v", response.Branding)
	}
}

func TestUnmarshalLegacyLightModeSetsLightTheme(t *testing.T) {
	var config Config
	if err := json.Unmarshal([]byte(`{"dark_mode":false,"colors":{"primary":"#2563eb"}}`), &config); err != nil {
		t.Fatal(err)
	}
	if config.Theme != ThemeLight {
		t.Fatalf("theme = %q, want %q", config.Theme, ThemeLight)
	}
}

func TestUnmarshalKeepsExplicitBranding(t *testing.T) {
	var config Config
	if err := json.Unmarshal([]byte(`{
		"theme":"system",
		"launcher":{"position":"left","spacing":{"side":10,"bottom":10}},
		"colors":{"primary":"#111111"},
		"branding":{
			"light":{"colors":{"primary":"#2563eb"}},
			"dark":{"colors":{"primary":"#60a5fa"}}
		}
	}`), &config); err != nil {
		t.Fatal(err)
	}
	if config.Theme != ThemeSystem {
		t.Fatalf("theme = %q, want %q", config.Theme, ThemeSystem)
	}
	if config.Branding.Light.Colors.Primary != "#2563eb" || config.Branding.Dark.Colors.Primary != "#60a5fa" {
		t.Fatalf("branding = %+v", config.Branding)
	}
}
