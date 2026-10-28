package main

import (
	"math"
	"strings"
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

func TestNormalizeChatSubject(t *testing.T) {
	ascii := strings.Repeat("a", maxChatSubjectLength)
	cyrillic := strings.Repeat("ж", maxChatSubjectLength) // 2 bytes per character: byte-counting would reject it

	tests := []struct {
		name string
		in   string
		want string
		ok   bool
	}{
		{"empty stays empty", "", "", true},
		{"whitespace collapsed and trimmed", "  Login \t fails\n after  update  ", "Login fails after update", true},
		{"ascii at the limit", ascii, ascii, true},
		{"ascii one over the limit", ascii + "a", "", false},
		{"non-ascii at the limit counts characters, not bytes", cyrillic, cyrillic, true},
		{"non-ascii one over the limit", cyrillic + "ж", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := normalizeChatSubject(tt.in)
			if ok != tt.ok || got != tt.want {
				t.Fatalf("normalizeChatSubject(%q) = (%q, %v), want (%q, %v)", tt.in, got, ok, tt.want, tt.ok)
			}
		})
	}
}
