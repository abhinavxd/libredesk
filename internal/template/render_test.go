package template

import (
	htmltemplate "html/template"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/zerodha/logf"
)

func TestRenderStringEscapesDataAndPreservesTemplateMarkup(t *testing.T) {
	m := &Manager{funcMap: htmltemplate.FuncMap{}}
	content := m.RenderString(map[string]any{
		"Contact": map[string]any{"FullName": "<b>attacker</b>"},
	}, `<p>Hi {{ .Contact.FullName }}</p>`)

	if !strings.Contains(content, "<p>Hi &lt;b&gt;attacker&lt;/b&gt;</p>") {
		t.Fatalf("unexpected content: %s", content)
	}
}

func TestRenderEmailWithTemplateEscapesDataAndPreservesTemplateMarkup(t *testing.T) {
	db := testutil.NewDB(t, "outgoing_email_template_escape")
	lo := logf.New(logf.Opts{})
	m, err := New(&lo, db, nil, nil, htmltemplate.FuncMap{}, testutil.NewI18n(t))
	if err != nil {
		t.Fatal(err)
	}

	content, err := m.RenderEmailWithTemplate(map[string]any{
		"Contact": map[string]any{"FullName": "<b>attacker</b>"},
	}, `<p>Hi {{ .Contact.FullName }}</p>`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(content, "<p>Hi &lt;b&gt;attacker&lt;/b&gt;</p>") {
		t.Fatalf("unexpected content: %s", content)
	}
}

func TestRenderStoredEmailTemplateEscapesUntrustedHTML(t *testing.T) {
	db := testutil.NewDB(t, "stored_email_template_escape")
	lo := logf.New(logf.Opts{})
	m, err := New(&lo, db, nil, nil, htmltemplate.FuncMap{"RootURL": func() string { return "http://localhost" }}, testutil.NewI18n(t))
	if err != nil {
		t.Fatal(err)
	}

	content, _, err := m.RenderStoredEmailTemplate(TmplNewReply, map[string]any{
		"Author": map[string]any{"FullName": "<b>attacker</b>"},
		"Conversation": map[string]any{
			"ReferenceNumber": 42,
			"Subject":         "<i>subject</i>",
			"UUID":            "conversation-uuid",
		},
		"Message": map[string]any{"Content": htmltemplate.HTML("<strong>safe message</strong>")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(content, "&lt;b&gt;attacker&lt;/b&gt;") || !strings.Contains(content, "&lt;i&gt;subject&lt;/i&gt;") {
		t.Fatalf("untrusted HTML was not escaped: %s", content)
	}
	if !strings.Contains(content, "<strong>safe message</strong>") {
		t.Fatalf("trusted message HTML was escaped: %s", content)
	}
}
