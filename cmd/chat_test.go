package main

import (
	"bytes"
	"math"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/abhinavxd/libredesk/internal/inbox/channel/livechat"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/zerodha/logf"
)

func TestValidateAttributeValuePreservesUTF8(t *testing.T) {
	lo := logf.New(logf.Opts{})
	app := &App{lo: &lo}
	for _, char := range []string{"a", "é", "界", "😀"} {
		for padding := range len(char) {
			prefix := strings.Repeat("a", padding)
			value := prefix + strings.Repeat(char, 1001)
			want := prefix + strings.Repeat(char, (1000-padding)/len(char))
			got := validateAttributeValue("description", value, app).(string)
			if !utf8.ValidString(got) || got != want {
				t.Fatalf("padding %d: got %q, want %q", padding, got, want)
			}
		}
	}
}

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

func TestWidgetSessionStorageFailureIsVisible(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { client.Close() })
	var logs bytes.Buffer
	lo := logf.New(logf.Opts{Writer: &logs})
	app := &App{lo: &lo, redis: client}
	const token = "private-session-token"
	if _, err := loadSession(app, token, livechat.Config{}); err == nil {
		t.Fatal("missing session accepted")
	}
	if logs.Len() != 0 {
		t.Fatal("expired session produced an application error")
	}
	server.SetError("storage unavailable")
	if _, err := loadSession(app, token, livechat.Config{}); err == nil {
		t.Fatal("storage failure ignored")
	}
	if !strings.Contains(logs.String(), "level=error") || strings.Contains(logs.String(), token) {
		t.Fatalf("storage failure logging is missing or exposes the session: %s", logs.String())
	}
}
