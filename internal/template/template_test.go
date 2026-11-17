package template

import (
	"errors"
	"fmt"
	"html/template"
	"os"
	"regexp"
	"strings"
	"testing"

	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/template/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

var notificationSeedPattern = regexp.MustCompile(`(?s)VALUES\s*\(\s*'email_notification'::template_type,\s*'((?:''|[^'])*)'\s*,\s*false,\s*'([^']+)'`)

func TestDefaultTemplateConflict(t *testing.T) {
	db := testutil.NewDB(t, "default_template_conflict")
	lo := logf.New(logf.Opts{})
	m, err := New(&lo, db, nil /** webTpls **/, nil /** tpls **/, nil /** funcMap **/, testutil.NewI18n(t))
	if err != nil {
		t.Fatal(err)
	}
	first, err := m.Create(models.Template{Name: "First", Type: TypeEmailOutgoing, IsDefault: true, Body: `{{ template "content" . }}`})
	if err != nil {
		t.Fatal(err)
	}
	second, err := m.Create(models.Template{Name: "Second", Type: TypeEmailOutgoing, Body: `{{ template "content" . }}`})
	if err != nil {
		t.Fatal(err)
	}
	second.IsDefault = true
	for _, save := range []func() (models.Template, error){
		func() (models.Template, error) { return m.Create(second) },
		func() (models.Template, error) { return m.Update(second.ID, second) },
	} {
		_, err := save()
		var conflict envelope.Error
		if !errors.As(err, &conflict) || conflict.ErrorType != envelope.ConflictError {
			t.Fatalf("expected an actionable conflict, got %v", err)
		}
		current, err := m.getDefaultOutgoingEmailTemplate()
		if err != nil || current.ID != first.ID {
			t.Fatalf("default changed after rejected save: %+v, %v", current, err)
		}
	}
	first.IsDefault = false
	if _, err := m.Update(first.ID, first); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Update(second.ID, second); err != nil {
		t.Fatal(err)
	}
	current, err := m.getDefaultOutgoingEmailTemplate()
	if err != nil || current.ID != second.ID {
		t.Fatalf("default did not switch: %+v, %v", current, err)
	}
}

func TestUpdateDefaultTemplateType(t *testing.T) {
	db := testutil.NewDB(t, "default_template_type")
	lo := logf.New(logf.Opts{})
	m, err := New(&lo, db, nil /** webTpls **/, nil /** tpls **/, nil /** funcMap **/, testutil.NewI18n(t))
	if err != nil {
		t.Fatal(err)
	}
	saved, err := m.Create(models.Template{Name: "Layout", Type: TypeEmailOutgoing, Body: `{{ template "content" . }}`})
	if err != nil {
		t.Fatal(err)
	}
	for _, typ := range []string{TypeEmailNotification, "", "invalid", TypeEmailOutgoing} {
		candidate := saved
		candidate.Type = typ
		candidate.IsDefault = true
		updated, err := m.Update(saved.ID, candidate)
		if err != nil || updated.Type != TypeEmailOutgoing || !updated.IsDefault {
			t.Fatalf("default type %q was not normalized: %+v, %v", typ, updated, err)
		}
		stored, err := m.Get(saved.ID)
		if err != nil || stored.Type != TypeEmailOutgoing || !stored.IsDefault {
			t.Fatalf("incorrect persisted default: %+v, %v", stored, err)
		}
		candidate.Body = "<p>No reply placeholder</p>"
		if _, err := m.Update(saved.ID, candidate); err == nil {
			t.Fatalf("accepted default type %q without a reply", typ)
		}
	}
	saved.Type = TypeEmailNotification
	saved.Body = "<p>Notification</p>"
	updated, err := m.Update(saved.ID, saved)
	if err != nil || updated.Type != TypeEmailNotification || updated.IsDefault {
		t.Fatalf("changed nondefault notification type: %+v, %v", updated, err)
	}
}

func TestValidateOutgoingTemplate(t *testing.T) {
	m := &Manager{i18n: testutil.NewI18n(t)}
	for _, tc := range []struct {
		name string
		body string
		ok   bool
	}{
		{"empty", "", false},
		{"missing", "<p>Header</p>", false},
		{"single", `{{ template "content" . }}`, true},
		{"duplicate", `{{ template "content" . }}{{ template "content" . }}`, true},
		{"triple", `{{ template "content" . }}{{ template "content" . }}{{ template "content" . }}`, true},
		{"trimmed duplicate", `{{- template "content" . -}}<p>Footer</p>{{- template "content" . -}}`, true},
		{"comment only", `{{/* template "content" . */}}`, false},
		{"comment and single", `{{/* template "content" . */}}{{ template "content" . }}`, true},
		{"malformed", `{{ template "content" .`, false},
		{"wrong name", `{{ template "contents" . }}`, false},
		{"literal template text", `{{ printf "%s" "{{ template \"content\" . }}" }}`, false},
		{"nested wrappers", `{{ define "inner" }}{{ template "content" . }}{{ end }}{{ define "outer" }}{{ template "inner" . }}{{ end }}{{ template "outer" . }}`, true},
		{"nested duplicate", `{{ define "inner" }}{{ template "content" . }}{{ end }}{{ define "outer" }}{{ template "inner" . }}{{ template "inner" . }}{{ end }}{{ template "outer" . }}`, true},
		{"unused duplicate", `{{ define "unused" }}{{ template "content" . }}{{ template "content" . }}{{ end }}{{ template "content" . }}`, true},
		{"duplicate alternatives", `{{ if .Show }}{{ template "content" . }}{{ template "content" . }}{{ else }}{{ template "content" . }}{{ template "content" . }}{{ end }}`, true},
		{"nested alternatives", `{{ if .Show }}{{ with .Author }}{{ template "content" . }}{{ else }}{{ template "content" . }}{{ end }}{{ else }}{{ template "content" . }}{{ end }}`, true},
		{"dynamic conditional", `{{ template "content" . }}{{ if .Show }}{{ template "content" . }}{{ end }}`, true},
		{"alternatives", `{{ if .Show }}{{ template "content" . }}{{ else }}{{ template "content" . }}{{ end }}`, true},
		{"with alternatives", `{{ with .Author }}{{ template "content" . }}{{ else }}{{ template "content" . }}{{ end }}`, true},
		{"duplicate with alternatives", `{{ template "content" . }}{{ with .Author }}{{ template "content" . }}{{ else }}{{ template "content" . }}{{ end }}`, true},
		{"unused definition", `{{ define "unused" }}{{ template "content" . }}{{ end }}{{ template "content" . }}`, true},
		{"single wrapper", `{{ define "wrapper" }}{{ template "content" . }}{{ end }}{{ template "wrapper" . }}`, true},
		{"repeated wrapper", `{{ define "wrapper" }}{{ template "content" . }}{{ end }}{{ template "wrapper" . }}{{ template "wrapper" . }}`, true},
		{"previous messages", `{{ template "content" . }}{{ range .PreviousMessages }}{{ .Content }}{{ end }}`, true},
		{"dynamic loop", `{{ range . }}{{ template "content" . }}{{ end }}`, true},
		{"recursive definition", `{{ define "wrapper" }}{{ template "wrapper" . }}{{ end }}{{ template "wrapper" . }}{{ template "content" . }}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := m.validateOutgoingTemplate(models.Template{Type: TypeEmailOutgoing, Body: tc.body})
			if (err == nil) != tc.ok {
				t.Fatalf("validation error = %v, want valid = %v", err, tc.ok)
			}
		})
	}
}

func TestOutgoingTemplatePreservesReply(t *testing.T) {
	db := testutil.NewDB(t, "outgoing_template_content")
	lo := logf.New(logf.Opts{})
	m, err := New(&lo, db, nil /** webTpls **/, nil /** tpls **/, nil /** funcMap **/, testutil.NewI18n(t))
	if err != nil {
		t.Fatal(err)
	}
	valid := models.Template{Name: "Customer layout", Type: TypeEmailOutgoing, IsDefault: true, Body: `<header>Support</header>{{ template "content" . }}<footer>Thank you</footer>`}
	saved, err := m.Create(valid)
	if err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{
		"",
		"<p>Support header</p>",
		`{{/* template "content" . */}}`,
		`{{ template "content" .`,
	} {
		t.Run("reject "+body, func(t *testing.T) {
			candidate := valid
			candidate.Body = body
			candidate.IsDefault = false
			if _, err := m.Create(candidate); err == nil {
				t.Fatal("created an invalid outgoing template")
			}
			if _, err := m.Update(saved.ID, candidate); err == nil {
				t.Fatal("updated to an invalid outgoing template")
			}
			stored, err := m.Get(saved.ID)
			if err != nil || stored.Body != valid.Body || !stored.IsDefault {
				t.Fatalf("rejected edit changed the saved template: %+v, %v", stored, err)
			}
			var count int
			if err := db.Get(&count, `SELECT COUNT(*) FROM templates WHERE type = $1`, TypeEmailOutgoing); err != nil || count != 1 {
				t.Fatalf("rejected create changed template count: %d, %v", count, err)
			}
		})
	}
	stored, err := m.Get(saved.ID)
	if err != nil || stored.Body != valid.Body {
		t.Fatalf("rejected edit changed template: %+v, %v", stored, err)
	}
	for _, body := range []string{
		`{{ template "content" . }}{{ template "content" . }}`,
		`{{ define "wrapper" }}{{ template "content" . }}{{ end }}{{ template "wrapper" . }}{{ template "wrapper" . }}`,
		`{{ range . }}{{ template "content" . }}{{ end }}`,
	} {
		candidate := valid
		candidate.Body = body
		candidate.IsDefault = false
		if _, err := m.Create(candidate); err != nil {
			t.Fatalf("could not create repeated reply template: %v", err)
		}
		candidate.IsDefault = true
		if _, err := m.Update(saved.ID, candidate); err != nil {
			t.Fatalf("could not update repeated reply template: %v", err)
		}
	}
	const conditional = `{{ if .Show }}<p>Hello</p>{{ template "content" . }}{{ else }}{{ template "content" . }}{{ end }}`
	if _, err := m.Update(saved.ID, models.Template{Name: valid.Name, Type: valid.Type, IsDefault: true, Body: conditional}); err != nil {
		t.Fatalf("rejected a conditional layout that renders the reply once: %v", err)
	}
	for _, tc := range []struct {
		body  string
		data  any
		count int
	}{
		{valid.Body, nil, 1},
		{`{{template "content" .}}`, nil, 1},
		{`{{- template "content" . -}}`, nil, 1},
		{`{{ define "wrapper" }}{{ template "content" . }}{{ end }}{{ template "wrapper" . }}`, nil, 1},
		{`{{ if .Show }}{{ template "content" . }}{{ end }}`, map[string]any{"Show": true}, 1},
		{`{{ if .Show }}{{ template "content" . }}{{ end }}`, map[string]any{"Show": false}, 0},
		{conditional, map[string]any{"Show": true}, 1},
		{conditional, map[string]any{"Show": false}, 1},
		{`{{ template "content" . }}{{ template "content" . }}`, nil, 2},
		{`{{ range . }}{{ template "content" . }}{{ end }}`, []string{}, 0},
		{`{{ range . }}{{ template "content" . }}{{ end }}`, []string{"first", "second"}, 2},
		{`{{ define "wrapper" }}{{ template "content" . }}{{ end }}{{ template "wrapper" . }}{{ template "wrapper" . }}`, nil, 2},
		{`<p>Previously saved layout without a reply</p>`, nil, 0},
	} {
		t.Run("render "+tc.body+fmt.Sprint(tc.data), func(t *testing.T) {
			if _, err := db.Exec(`UPDATE templates SET body = $1 WHERE id = $2`, tc.body, saved.ID); err != nil {
				t.Fatal(err)
			}
			const reply = "<p>Your receipt is ready. Café 日本語.</p>"
			got, err := m.RenderEmailWithTemplate(tc.data, reply)
			if tc.count > 0 {
				if err != nil || strings.Count(got, reply) != tc.count {
					t.Fatalf("unexpected rendered reply: %q, %v", got, err)
				}
			} else if err == nil || got != "" {
				t.Fatalf("unsafe outgoing content accepted: %q, %v", got, err)
			}
		})
	}
	if _, err := db.Exec(`UPDATE templates SET body = $1 WHERE id = $2`, valid.Body, saved.ID); err != nil {
		t.Fatal(err)
	}
	const contactText = `<p>Hi {{range(99999999999)}}A{{end}} {{ .Show }}</p>`
	got, err := m.RenderEmailWithTemplate(map[string]any{"Show": true}, contactText)
	if err != nil || !strings.Contains(got, contactText) {
		t.Fatalf("reply text was run as a template: %q, %v", got, err)
	}
}

func TestRenderNotificationEmailUsesSystemLayout(t *testing.T) {
	m := emailTemplateManager(t)
	var header strings.Builder
	if err := m.tpls.ExecuteTemplate(&header, "header", nil /** data **/); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Customer A", "Customer B"} {
		t.Run(name, func(t *testing.T) {
			data := map[string]any{
				"Author":  map[string]any{"FirstName": name, "FullName": name},
				"Message": map[string]any{"Content": `<script>alert("test")</script> {{ .Author.FirstName }}`},
			}
			got, err := m.RenderNotificationEmail(`<p>{{ .Author.FullName }} replied.</p><blockquote>{{ .Message.Content }}</blockquote>`, data)
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
			if !strings.Contains(got, "<!doctype html>") || !strings.Contains(got, "</html>") || !strings.Contains(got, "support.example.com/images/logo.png") {
				t.Fatal("notification lost the system layout")
			}
		})
	}
}

func TestRenderNotificationEmailUsesCurrentFunctions(t *testing.T) {
	layout := template.Must(template.New("emails").Parse(`{{ define "header" }}<html><body>{{ end }}{{ define "footer" }}</body></html>{{ end }}`))
	m := &Manager{}
	for _, rootURL := range []string{"https://support.example.com", "https://updated.example.com"} {
		funcs := template.FuncMap{"RootURL": func() string { return rootURL }}
		if err := m.Reload(nil /** webTpls **/, layout, funcs); err != nil {
			t.Fatal(err)
		}
		got, err := m.RenderNotificationEmail(`<a href="{{ RootURL }}">View conversation</a>`, nil /** data **/)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(got, `href="`+rootURL+`"`) {
			t.Fatal("notification did not use the current template functions")
		}
	}
}

func TestBuildConversationData(t *testing.T) {
	conversation := cmodels.Conversation{
		ReferenceNumber: "C-42",
		Subject:         null.StringFrom("Account <question>"),
		Priority:        null.StringFrom("High"),
		UUID:            "conversation-a",
		InboxName:       "Support",
		InboxChannel:    "email",
		Contact: cmodels.ConversationContact{
			FirstName: "Customer",
			LastName:  "A",
			Email:     null.StringFrom("customer@example.com"),
		},
	}
	customer := umodels.User{FirstName: "Customer", LastName: "A", Email: null.StringFrom("customer@example.com")}
	agent := umodels.User{FirstName: "Agent", LastName: "B", Email: null.StringFrom("agent@example.com")}
	content := template.Must(template.New("content").Option("missingkey=error").Parse(
		`{{ .Conversation.ReferenceNumber }}|{{ .Conversation.Subject }}|{{ .Conversation.Priority }}|{{ .Conversation.UUID }}|{{ .Inbox.Name }}|{{ .Inbox.Channel }}|{{ .Contact.FirstName }}|{{ .Contact.LastName }}|{{ .Contact.FullName }}|{{ .Contact.Email }}|{{ .Recipient.FirstName }}|{{ .Recipient.LastName }}|{{ .Recipient.FullName }}|{{ .Recipient.Email }}|{{ .Author.FirstName }}|{{ .Author.LastName }}|{{ .Author.FullName }}|{{ .Author.Email }}|{{ .IsAIComposed }}`))
	for _, tt := range []struct {
		name      string
		recipient umodels.User
		author    umodels.User
		want      string
	}{
		{
			name:      "customer reply",
			recipient: customer,
			author:    agent,
			want:      "Customer|A|Customer A|customer@example.com|Agent|B|Agent B|agent@example.com|false",
		},
		{
			name:      "incoming reply alert",
			recipient: agent,
			author:    customer,
			want:      "Agent|B|Agent B|agent@example.com|Customer|A|Customer A|customer@example.com|false",
		},
		{
			name:      "automatic alert",
			recipient: agent,
			want:      "Agent|B|Agent B|agent@example.com||| ||false",
		},
		{
			name:      "automated customer reply",
			recipient: customer,
			author:    umodels.User{FirstName: "System", Email: null.StringFrom(umodels.SystemUserEmail)},
			want:      "Customer|A|Customer A|customer@example.com|||||false",
		},
		{
			name:      "AI reply",
			recipient: customer,
			author:    umodels.User{FirstName: "Assistant", Type: umodels.UserTypeAIAssistant},
			want:      "Customer|A|Customer A|customer@example.com|Assistant||Assistant ||true",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			data := BuildConversationData(conversation, tt.recipient, tt.author)
			var rendered strings.Builder
			if err := content.Execute(&rendered, data); err != nil {
				t.Fatal(err)
			}
			want := "C-42|Account &lt;question&gt;|High|conversation-a|Support|email|Customer|A|Customer A|customer@example.com|" + tt.want
			if rendered.String() != want {
				t.Fatalf("rendered = %q, want %q", rendered.String(), want)
			}
		})
	}
}

func TestDefaultNotificationTemplatesRenderForRecipient(t *testing.T) {
	schema, err := os.ReadFile("../../schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	bodies := make(map[string]string)
	for _, match := range notificationSeedPattern.FindAllStringSubmatch(string(schema), -1) {
		bodies[match[2]] = strings.ReplaceAll(match[1], "''", "'")
	}
	m := emailTemplateManager(t)
	conversation := cmodels.Conversation{
		UUID: "conversation-a", ReferenceNumber: "C-42", Subject: null.StringFrom("Account <question>"),
		InboxName: "Support", InboxChannel: "email",
		Contact: cmodels.ConversationContact{FirstName: "Customer", LastName: "A"},
	}
	recipient := umodels.User{FirstName: "Agent", LastName: "B"}
	for _, name := range []string{
		TmplConversationAssigned, TmplNewReply, TmplNewReplyParticipating, TmplConversationReopened,
		TmplMentioned, TmplSLABreachWarning, TmplSLABreached,
	} {
		t.Run(name, func(t *testing.T) {
			body, ok := bodies[name]
			if !ok {
				t.Fatalf("missing default template %q", name)
			}
			data := BuildConversationData(conversation, recipient, umodels.User{FirstName: "Customer", LastName: "A"})
			data["MentionedBy"] = data["Author"]
			data["Message"] = map[string]any{"UUID": "message-a", "Content": "Latest <reply>"}
			data["SLA"] = map[string]any{"Metric": "First response", "DueIn": "5 minutes", "OverdueBy": "2 minutes"}
			got, err := m.RenderNotificationEmail(body, data)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Count(got, "Hi Agent,") != 1 || strings.Contains(got, "Hi Customer,") {
				t.Fatalf("notification greeting used the wrong person: %s", got)
			}
			if !strings.Contains(got, "C-42") || !strings.Contains(got, "/conversation/conversation-a") {
				t.Fatal("notification lost its conversation reference or link")
			}
			if strings.Contains(body, ".Message.Content") && !strings.Contains(got, "Latest &lt;reply&gt;") {
				t.Fatal("notification lost or interpreted the reply content")
			}
		})
	}
}

func TestNotificationRenderingKeepsRecipientsAndSystemEmailsSeparate(t *testing.T) {
	m := emailTemplateManager(t)
	welcomeData := map[string]any{"Email": "agent@example.com", "ResetToken": "test-token"}
	welcome, err := m.RenderInMemoryTemplate(TmplWelcome, welcomeData)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("parallel notifications", func(t *testing.T) {
		for index := range 8 {
			t.Run(fmt.Sprintf("recipient %d", index), func(t *testing.T) {
				t.Parallel()
				recipient := fmt.Sprintf("Agent %d", index)
				body := fmt.Sprintf(`<p>Notice %d: {{ .Recipient }}</p>`, index)
				want := fmt.Sprintf("Notice %d: %s", index, recipient)
				for range 3 {
					got, err := m.RenderNotificationEmail(body, map[string]any{"Recipient": recipient})
					if err != nil {
						t.Fatal(err)
					}
					if !strings.Contains(got, want) || strings.Contains(got, "agent@example.com") || strings.Count(got, "Notice ") != 1 {
						t.Fatalf("notification contained another template or recipient: %s", got)
					}
				}
			})
		}
	})
	after, err := m.RenderInMemoryTemplate(TmplWelcome, welcomeData)
	if err != nil {
		t.Fatal(err)
	}
	if after != welcome || !strings.Contains(after, "test-token") {
		t.Fatal("notification rendering changed the welcome email")
	}
}

func TestBrokenNotificationTemplateDoesNotAffectLaterEmails(t *testing.T) {
	m := emailTemplateManager(t)
	data := map[string]any{"Recipient": map[string]any{"FirstName": "Agent"}}
	for _, body := range []string{"{{", "{{ .Recipient.FirstName.Unknown }}"} {
		t.Run(body, func(t *testing.T) {
			if _, err := m.RenderNotificationEmail(body, data); err == nil {
				t.Fatal("broken notification template rendered without an error")
			}
			got, err := m.RenderNotificationEmail("<p>Hi {{ .Recipient.FirstName }},</p>", data)
			if err != nil || !strings.Contains(got, "Hi Agent,") {
				t.Fatalf("valid notification failed after a broken template: %v", err)
			}
		})
	}
}

func emailTemplateManager(t *testing.T) *Manager {
	t.Helper()
	funcs := template.FuncMap{
		"RootURL":  func() string { return "https://support.example.com" },
		"SiteName": func() string { return "Support" },
		"LogoURL":  func() string { return "" },
	}
	layout, err := template.New("emails").Option("missingkey=error").Funcs(funcs).ParseFiles(
		"../../static/email-templates/base.html", "../../static/email-templates/welcome.html", "../../static/email-templates/reset-password.html")
	if err != nil {
		t.Fatal(err)
	}
	m := &Manager{}
	if err := m.Reload(nil /** webTpls **/, layout, funcs); err != nil {
		t.Fatal(err)
	}
	return m
}
