package template

import (
	"html/template"
	"os"
	"strings"
	"testing"
)

func TestRenderNotificationEmailUsesSystemLayout(t *testing.T) {
	body, err := os.ReadFile("../../static/email-templates/base.html")
	if err != nil {
		t.Fatal(err)
	}
	funcs := template.FuncMap{
		"RootURL":  func() string { return "https://support.example.com" },
		"SiteName": func() string { return "Support" },
		"LogoURL":  func() string { return "" },
	}
	layout := template.Must(template.New("emails").Funcs(funcs).Parse(string(body)))
	m := &Manager{}
	if err := m.Reload(nil /** webTpls **/, layout, funcs); err != nil {
		t.Fatal(err)
	}
	var header strings.Builder
	if err := layout.ExecuteTemplate(&header, "header", nil /** data **/); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Customer A", "Customer B"} {
		t.Run(name, func(t *testing.T) {
			data := map[string]any{
				"Author":  map[string]any{"FirstName": name, "FullName": name},
				"Message": map[string]any{"Content": `<script>alert("test")</script> {{ .Author.FirstName }}`},
			}
			got, err := m.renderNotificationEmail(`<p>{{ .Author.FullName }} replied.</p><blockquote>{{ .Message.Content }}</blockquote>`, data)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(got, name+" replied.") {
				t.Fatal("notification lost the reply author")
			}
			if strings.Count(got, name) != 1 || strings.Contains(got, "Best regards") {
				t.Fatal("notification added a customer signature")
			}
			if strings.Contains(got, "<script>") || !strings.Contains(got, "{{ .Author.FirstName }}") {
				t.Fatal("message content was interpreted as HTML or template code")
			}
			if !strings.Contains(got, "<html>") || !strings.Contains(got, "</html>") || !strings.Contains(got, "support.example.com/images/logo.png") {
				t.Fatal("notification lost the system layout")
			}
		})
	}
}

func TestRenderNotificationEmailUsesCurrentFunctions(t *testing.T) {
	layout := template.Must(template.New("emails").Parse(`{{ define "header" }}<html><body>{{ end }}{{ define "footer" }}</body></html>{{ end }}`))
	funcs := template.FuncMap{"RootURL": func() string { return "https://support.example.com" }}
	m := &Manager{}
	if err := m.Reload(nil /** webTpls **/, layout, funcs); err != nil {
		t.Fatal(err)
	}
	got, err := m.renderNotificationEmail(`<a href="{{ RootURL }}">View conversation</a>`, nil /** data **/)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `href="https://support.example.com"`) {
		t.Fatal("notification did not use the current template functions")
	}
}
