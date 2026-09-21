package main

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/abhinavxd/libredesk/internal/inbox/channel/livechat"
)

func TestIsFormFieldValuePresent(t *testing.T) {
	tests := []struct {
		name  string
		field livechat.PreChatFormField
		value any
		want  bool
	}{
		{name: "text", field: livechat.PreChatFormField{Type: "text"}, value: "answer", want: true},
		{name: "blank text", field: livechat.PreChatFormField{Type: "text"}, value: "  "},
		{name: "wrong text type", field: livechat.PreChatFormField{Type: "text"}, value: []string{"answer"}},
		{name: "number", field: livechat.PreChatFormField{Type: "number"}, value: float64(0), want: true},
		{name: "invalid number", field: livechat.PreChatFormField{Type: "number"}, value: math.NaN()},
		{name: "checkbox", field: livechat.PreChatFormField{Type: "checkbox"}, value: false, want: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isFormFieldValuePresent(tc.field, tc.value); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestInitialChatConfigDisablesHandoffOnlyForm(t *testing.T) {
	var config livechat.Config
	config.PreChatForm.Enabled = true
	config.PreChatForm.HandoffOnly = true

	initial := resolveInitialChatConfig(config, true)
	if initial.PreChatForm.Enabled {
		t.Fatal("handoff-only form remained enabled for initial chat")
	}
	if !config.PreChatForm.Enabled {
		t.Fatal("input config was mutated")
	}
}

func TestInitialChatConfigUsesAudienceSetting(t *testing.T) {
	var config livechat.Config
	config.PreChatForm.Enabled = true
	showVisitors := true
	showUsers := false
	config.PreChatForm.Visitors = &livechat.AudiencePreChatFormConfig{Enabled: &showVisitors}
	config.PreChatForm.Users = &livechat.AudiencePreChatFormConfig{Enabled: &showUsers}

	if !resolveInitialChatConfig(config, true).PreChatForm.Enabled {
		t.Fatal("visitor pre-chat form was disabled")
	}
	if resolveInitialChatConfig(config, false).PreChatForm.Enabled {
		t.Fatal("user pre-chat form remained enabled")
	}
}

func TestValidateHandoffFormUsesSubmittedContactDetails(t *testing.T) {
	var config livechat.Config
	config.PreChatForm.Enabled = true
	config.PreChatForm.Fields = []livechat.PreChatFormField{{
		Key:       "name",
		Type:      "text",
		Enabled:   true,
		Required:  true,
		IsDefault: true,
	}}

	name, _, _, _, err := validateFormData(nil, map[string]any{"name": "QA Visitor"}, config, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "QA Visitor" {
		t.Fatalf("got name %q", name)
	}
}

func TestChatLauncherSettingsKeepsLegacyFields(t *testing.T) {
	config := livechat.Config{
		Theme: livechat.ThemeDark,
		Launcher: livechat.LauncherLayout{
			Position:  "right",
			IconScale: 80,
			Spacing:   livechat.LauncherSpacing{Side: 20, Bottom: 24},
		},
		Branding: livechat.BrandingSet{
			Light: livechat.Branding{Colors: livechat.Colors{Primary: "#ffffff"}},
			Dark: livechat.Branding{
				Colors:   livechat.Colors{Primary: "#111111"},
				Launcher: livechat.BrandingLauncher{LogoURL: "https://example.com/logo.png", Color: "#222222"},
			},
		},
	}

	encoded, err := json.Marshal(chatLauncherSettings(config))
	if err != nil {
		t.Fatal(err)
	}
	var response struct {
		Colors   livechat.Colors `json:"colors"`
		Launcher struct {
			Position  string `json:"position"`
			IconScale int    `json:"icon_scale"`
			LogoURL   string `json:"logo_url"`
			Color     string `json:"color"`
		} `json:"launcher"`
		Branding map[string]json.RawMessage `json:"branding"`
	}
	if err := json.Unmarshal(encoded, &response); err != nil {
		t.Fatal(err)
	}
	if response.Colors.Primary != "#111111" {
		t.Fatalf("legacy primary = %q", response.Colors.Primary)
	}
	if response.Launcher.Position != "right" || response.Launcher.IconScale != 80 {
		t.Fatalf("legacy launcher layout = %+v", response.Launcher)
	}
	if response.Launcher.LogoURL != "https://example.com/logo.png" || response.Launcher.Color != "#222222" {
		t.Fatalf("legacy launcher branding = %+v", response.Launcher)
	}
	if response.Branding["light"] == nil || response.Branding["dark"] == nil {
		t.Fatalf("branding = %+v", response.Branding)
	}

	settingsResponse, err := json.Marshal(chatSettingsResponse{
		Config:     config,
		DarkMode:   true,
		Colors:     config.Branding.Dark.Colors,
		HomeScreen: config.Branding.Dark.HomeScreen,
		Launcher:   buildWidgetLauncherSettings(config, config.Branding.Dark),
		LogoURL:    config.Branding.Dark.LogoURL,
	})
	if err != nil {
		t.Fatal(err)
	}
	var fullSettings struct {
		DarkMode   bool                   `json:"dark_mode"`
		Colors     livechat.Colors        `json:"colors"`
		HomeScreen livechat.HomeScreen    `json:"home_screen"`
		Launcher   widgetLauncherSettings `json:"launcher"`
		LogoURL    string                 `json:"logo_url"`
		Branding   livechat.BrandingSet   `json:"branding"`
	}
	if err := json.Unmarshal(settingsResponse, &fullSettings); err != nil {
		t.Fatal(err)
	}
	if !fullSettings.DarkMode || fullSettings.Colors.Primary != "#111111" {
		t.Fatalf("legacy theme = %+v", fullSettings)
	}
	if fullSettings.Launcher.LogoURL != "https://example.com/logo.png" {
		t.Fatalf("legacy launcher = %+v", fullSettings.Launcher)
	}
	if fullSettings.Branding.Dark.Colors.Primary != "#111111" {
		t.Fatalf("new branding = %+v", fullSettings.Branding)
	}
}
