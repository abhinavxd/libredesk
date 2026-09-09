package template

import (
	"strings"
	"testing"
	"text/template"
)

func TestRenderStringWithError(t *testing.T) {
	m := &Manager{}
	data := map[string]any{"Inbox": map[string]any{"Product": "Drifttt"}}

	got, err := m.RenderStringWithError(data, "Hello from {{ .Inbox.Product }}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := "Hello from Drifttt"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	if _, err := m.RenderStringWithError(data, "{{ .Inbox.Product "); err == nil {
		t.Error("expected a parse error for an unclosed action")
	}
	if _, err := m.RenderStringWithError(data, "{{ .Inbox.Product.Nope }}"); err == nil {
		t.Error("expected an execution error for an unknown field")
	}

	// RenderString keeps swallowing errors and returning the content unchanged.
	if got := m.RenderString(data, "{{ .Inbox.Product "); got != "{{ .Inbox.Product " {
		t.Errorf("RenderString on a broken template returned %q", got)
	}
}

// renderDefaultOutgoing mirrors how RenderEmailWithTemplate composes the default outgoing template
// around a message body, without needing a database-backed Manager.
func renderDefaultOutgoing(t *testing.T, content string, data any) string {
	t.Helper()
	base, err := template.New(TmplBase).Parse(DefaultOutgoingEmailTemplate)
	if err != nil {
		t.Fatalf("parsing default outgoing template: %v", err)
	}
	body, err := template.New(TmplContent).Parse(content)
	if err != nil {
		t.Fatalf("parsing content: %v", err)
	}
	if base, err = base.AddParseTree(TmplContent, body.Tree); err != nil {
		t.Fatalf("adding content template: %v", err)
	}
	var out strings.Builder
	if err := base.ExecuteTemplate(&out, TmplBase, data); err != nil {
		t.Fatalf("executing default outgoing template: %v", err)
	}
	return out.String()
}

func TestDefaultOutgoingEmailTemplate(t *testing.T) {
	withSignature := map[string]any{"Inbox": map[string]any{"Signature": "Slava from Drifttt"}}
	if got, want := renderDefaultOutgoing(t, "Hi", withSignature), "Hi<div>Slava from Drifttt</div>"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	noSignature := map[string]any{"Inbox": map[string]any{"Signature": ""}}
	if got, want := renderDefaultOutgoing(t, "Hi", noSignature), "Hi"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	// Notification templates are wrapped in the same default template with data that has no Inbox key;
	// the signature block must stay silent there instead of failing the send.
	notification := map[string]any{"Conversation": map[string]any{"ReferenceNumber": "42"}}
	if got, want := renderDefaultOutgoing(t, "Hi", notification), "Hi"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
