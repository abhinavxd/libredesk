package template

import (
	htmltemplate "html/template"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/zerodha/logf"
)

func TestRenderStringPreservesTemplateMarkup(t *testing.T) {
	m := &Manager{funcMap: htmltemplate.FuncMap{}}
	content := m.RenderString(map[string]any{
		"Contact": map[string]any{"FullName": "Jane"},
	}, `<!--[if mso]><table><![endif]--><p>Hi {{ .Contact.FullName }}<b unclosed`)

	if !strings.Contains(content, "<!--[if mso]><table><![endif]-->") {
		t.Fatalf("HTML comment was stripped: %s", content)
	}
	if !strings.Contains(content, "Hi Jane") {
		t.Fatalf("malformed markup blocked rendering: %s", content)
	}
}

func TestRenderEmailWithTemplatePreservesTemplateMarkup(t *testing.T) {
	db := testutil.NewDB(t, "outgoing_email_template_markup")
	lo := logf.New(logf.Opts{})
	m, err := New(&lo, db, nil, nil, htmltemplate.FuncMap{}, testutil.NewI18n(t))
	if err != nil {
		t.Fatal(err)
	}

	content, err := m.RenderEmailWithTemplate(map[string]any{
		"Contact": map[string]any{"FullName": "Jane"},
	}, `<!--[if mso]><table><![endif]--><p>Hi {{ .Contact.FullName }}<b unclosed`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(content, "<!--[if mso]><table><![endif]-->") {
		t.Fatalf("HTML comment was stripped: %s", content)
	}
	if !strings.Contains(content, "Hi Jane") {
		t.Fatalf("malformed markup blocked rendering: %s", content)
	}
}
