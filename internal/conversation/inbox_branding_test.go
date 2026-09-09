package conversation

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/template"
	"github.com/zerodha/logf"
)

func TestProductNameFromInbox(t *testing.T) {
	cases := map[string]string{
		"Drifttt Support":          "Drifttt",
		"Drifttt App":              "Drifttt",
		"Maria App Studio Support": "Maria App Studio",
		"Drifttt":                  "Drifttt",
		"  Studio Helpdesk ":       "Studio",
		"Support":                  "Support",
		"":                         "",
	}
	for in, want := range cases {
		if got := ProductNameFromInbox(in); got != want {
			t.Errorf("ProductNameFromInbox(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestResolveInboxProduct(t *testing.T) {
	cases := []struct {
		name       string
		inboxName  string
		configured string
		want       string
	}{
		{"configured wins over derived", "Drifttt Support", "Drifttt Time Tracker", "Drifttt Time Tracker"},
		{"configured renames the product", "Studio inbox", "Maria App Studio", "Maria App Studio"},
		{"unset falls back to derived", "Drifttt Support", "", "Drifttt"},
		{"blank falls back to derived", "Drifttt App", "   ", "Drifttt"},
		{"configured is trimmed", "Drifttt Support", "  Drifttt  ", "Drifttt"},
		{"no inbox name and no config", "", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveInboxProduct(tc.inboxName, tc.configured); got != tc.want {
				t.Errorf("resolveInboxProduct(%q, %q) = %q, want %q", tc.inboxName, tc.configured, got, tc.want)
			}
		})
	}
}

func TestRenderInboxSignature(t *testing.T) {
	lo := logf.New(logf.Opts{})
	m := &Manager{lo: &lo, template: &template.Manager{}}

	data := map[string]any{
		"Agent": map[string]any{"FirstName": "Slava", "FullName": "Slava P"},
		"Inbox": map[string]any{"Name": "Drifttt Support", "Product": "Drifttt"},
	}

	cases := []struct {
		name      string
		signature string
		want      string
	}{
		{"unset", "", ""},
		{"plain text", "The Drifttt team", "The Drifttt team"},
		{"agent and product", "{{ .Agent.FirstName }} from {{ .Inbox.Product }}", "Slava from Drifttt"},
		{"simple html", "<b>{{ .Agent.FullName }}</b><br>{{ .Inbox.Name }}", "<b>Slava P</b><br>Drifttt Support"},
		// A signature the admin typed wrong must be dropped, never half-rendered into the email.
		{"unparseable template", "{{ .Agent.FirstName from {{ .Inbox.Product }}", ""},
		{"unclosed action", "{{ .Agent.FirstName ", ""},
		{"execution error", "{{ .Agent.FirstName.Nope }}", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := m.renderInboxSignature(tc.signature, data); got != tc.want {
				t.Errorf("renderInboxSignature(%q) = %q, want %q", tc.signature, got, tc.want)
			}
		})
	}
}
