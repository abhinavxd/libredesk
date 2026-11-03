package template

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"text/template"
	"text/template/parse"

	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/valyala/fasthttp"
)

const (
	// Built-in templates names stored in the database.
	TmplConversationAssigned  = "Conversation assigned"
	TmplSLABreachWarning      = "SLA breach warning"
	TmplSLABreached           = "SLA breached"
	TmplMentioned             = "Mentioned in conversation"
	TmplCSATRequest           = "CSAT request"
	TmplNewReply              = "New reply from contact"
	TmplNewReplyParticipating = "New reply on participating conversation"
	TmplConversationReopened  = "Conversation reopened"

	// Built-in templates fetched from memory stored in `static` directory.
	TmplResetPassword = "reset-password"
	TmplWelcome       = "welcome"

	// Template names for rendering.
	TmplBase    = "base"
	TmplContent = "content"
)

// RenderString renders Go template variables in the given content string
// without wrapping it in the base email template. Returns original content on any error.
func (m *Manager) RenderString(data any, content string) string {
	rendered, _ := renderString(data, content, m.funcMap)
	return rendered
}

// RenderStoredTemplate fetches a template by name and renders its body with the provided data
// without wrapping it in the base email template.
func (m *Manager) RenderStoredTemplate(name string, data any) (string, error) {
	tmpl, err := m.getByName(name)
	if err != nil {
		return "", err
	}
	rendered, err := renderString(data, tmpl.Body, m.funcMap)
	if err != nil {
		m.lo.Warn("saved template interpolation failed, using original content", "template", name, "error", err)
	}
	return rendered, nil
}

// RenderEmailWithTemplate places content verbatim inside the default outgoing email template.
func (m *Manager) RenderEmailWithTemplate(data any, content string) (string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	defaultTmpl, err := m.getDefaultOutgoingEmailTemplate()
	if err != nil && !errors.Is(err, ErrTemplateNotFound) {
		m.lo.Error("error fetching default outgoing email template", "error", err)
	}

	if defaultTmpl.Body == "" {
		defaultTmpl.Body = `{{ template "content" . }}`
	}

	baseTemplate, err := template.New(TmplBase).Funcs(m.funcMap).Parse(defaultTmpl.Body)
	if err != nil {
		return "", fmt.Errorf("parsing base template: %w", err)
	}

	// The reply is already rendered and can hold contact-supplied text, so it is inserted as a value and never parsed as a template.
	contentCount := 0
	contentFuncs := template.FuncMap{
		"_libredeskTrackContent": func() string {
			contentCount++
			return ""
		},
		"_libredeskReplyContent": func() string { return content },
	}
	baseTemplate.Funcs(contentFuncs)
	contentTemplate, err := template.New(TmplContent).Funcs(m.funcMap).Funcs(contentFuncs).Parse(`{{_libredeskTrackContent}}{{_libredeskReplyContent}}`)
	if err != nil {
		return "", fmt.Errorf("parsing content template: %w", err)
	}

	baseTemplate, err = baseTemplate.AddParseTree(TmplContent, contentTemplate.Tree)
	if err != nil {
		return "", fmt.Errorf("adding content template: %w", err)
	}

	var rendered strings.Builder
	if err := baseTemplate.ExecuteTemplate(&rendered, TmplBase, data); err != nil {
		return "", fmt.Errorf("executing base template: %w", err)
	}

	if contentCount == 0 {
		return "", fmt.Errorf("outgoing template did not render the reply")
	}
	return rendered.String(), nil
}

func (m *Manager) RenderStoredEmailTemplate(name string, data any) (string, string, error) {
	tmpl, err := m.getByName(name)
	if err != nil {
		if err == ErrTemplateNotFound {
			return "", "", fmt.Errorf("template %s not found", name)
		}
		return "", "", err
	}

	executeSubjectTemplate := func(subject string) (string, error) {
		var sb strings.Builder
		subjectTmpl, err := template.New("subject").Funcs(m.funcMap).Parse(subject)
		if err != nil {
			return "", fmt.Errorf("parsing subject template: %w", err)
		}
		if err := subjectTmpl.Execute(&sb, data); err != nil {
			return "", fmt.Errorf("executing subject template: %w", err)
		}
		return sb.String(), nil
	}

	content, err := m.RenderNotificationEmail(tmpl.Body, data)
	if err != nil {
		return "", "", err
	}

	subject, err := executeSubjectTemplate(tmpl.Subject.String)
	if err != nil {
		return "", "", err
	}

	return content, subject, nil
}

// RenderInMemoryTemplate executes an in-memory template with data and returns the rendered content.
// This is for system emails like reset password and welcome email etc.
func (m *Manager) RenderInMemoryTemplate(name string, data interface{}) (string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	var buf bytes.Buffer
	if err := m.tpls.ExecuteTemplate(&buf, name, data); err != nil {
		return "", fmt.Errorf("executing in-memory template %q: %w", name, err)
	}
	return buf.String(), nil
}

// RenderWebPage renders a template to the http.ResponseWriter with data.
func (m *Manager) RenderWebPage(ctx *fasthttp.RequestCtx, tmplFile string, data map[string]interface{}) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	ctx.SetContentType("text/html; charset=utf-8")
	ctx.SetStatusCode(fasthttp.StatusOK)
	// Add no-cache headers
	ctx.Response.Header.Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	ctx.Response.Header.Set("Pragma", "no-cache")
	ctx.Response.Header.Set("Expires", "0")
	return m.webTpls.ExecuteTemplate(ctx, tmplFile, data)
}

func (m *Manager) RenderNotificationEmail(content string, data any) (string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.notificationTpls == nil {
		return "", fmt.Errorf("notification templates are not loaded")
	}
	tmpl, err := m.notificationTpls.Clone()
	if err != nil {
		return "", fmt.Errorf("cloning notification templates: %w", err)
	}
	tmpl.Funcs(m.funcMap)
	if _, err := tmpl.New(TmplContent).Parse(content); err != nil {
		return "", fmt.Errorf("parsing notification content: %w", err)
	}
	tmpl, err = tmpl.New(TmplBase).Parse(`{{ template "header" . }}{{ template "content" . }}{{ template "footer" . }}`)
	if err != nil {
		return "", fmt.Errorf("parsing notification layout: %w", err)
	}
	var rendered strings.Builder
	if err := tmpl.ExecuteTemplate(&rendered, TmplBase, data); err != nil {
		return "", fmt.Errorf("rendering notification email: %w", err)
	}
	return rendered.String(), nil
}

func BuildConversationData(conversation cmodels.Conversation, recipient, author umodels.User) map[string]any {
	isAIComposed := author.Type == umodels.UserTypeAIAssistant
	authorData := personData(author)
	if author.IsSystemUser() {
		authorData = personData(umodels.User{})
		authorData["FullName"] = ""
	}
	return map[string]any{
		"Conversation": map[string]any{
			"ReferenceNumber": conversation.ReferenceNumber,
			"Subject":         conversation.Subject.String,
			"Priority":        conversation.Priority.String,
			"UUID":            conversation.UUID,
		},
		"Inbox": map[string]any{
			"Name":    conversation.InboxName,
			"Channel": conversation.InboxChannel,
		},
		"Contact": map[string]any{
			"FirstName": conversation.Contact.FirstName,
			"LastName":  conversation.Contact.LastName,
			"FullName":  conversation.Contact.FullName(),
			"Email":     conversation.Contact.Email.String,
		},
		"Recipient":    personData(recipient),
		"Author":       authorData,
		"IsAIComposed": isAIComposed,
	}
}

func personData(person umodels.User) map[string]any {
	return map[string]any{
		"FirstName": person.FirstName,
		"LastName":  person.LastName,
		"FullName":  person.FullName(),
		"Email":     person.Email.String,
	}
}

// parseOutgoingTemplate checks syntax and requires a reply placeholder. Rendering checks that the reply actually appears.
func parseOutgoingTemplate(body string, funcs template.FuncMap) (*template.Template, error) {
	tmpl, err := template.New(TmplBase).Funcs(funcs).Parse(body)
	if err != nil {
		return nil, err
	}
	count := 0
	for _, part := range tmpl.Templates() {
		count += countContentTemplates(part.Tree.Root)
	}
	if count == 0 {
		return nil, fmt.Errorf("outgoing template contains no reply template")
	}
	return tmpl, nil
}

func countContentTemplates(node parse.Node) int {
	switch node := node.(type) {
	case *parse.TemplateNode:
		if node.Name == TmplContent {
			return 1
		}
	case *parse.ListNode:
		if node == nil {
			return 0
		}
		count := 0
		for _, child := range node.Nodes {
			count += countContentTemplates(child)
		}
		return count
	case *parse.IfNode:
		return countContentTemplates(node.List) + countContentTemplates(node.ElseList)
	case *parse.RangeNode:
		return countContentTemplates(node.List) + countContentTemplates(node.ElseList)
	case *parse.WithNode:
		return countContentTemplates(node.List) + countContentTemplates(node.ElseList)
	}
	return 0
}

func renderString(data any, content string, funcs template.FuncMap) (string, error) {
	t, err := template.New("content").Funcs(funcs).Parse(content)
	if err != nil {
		return content, err
	}
	var buf strings.Builder
	if err := t.Execute(&buf, data); err != nil {
		return content, err
	}
	return buf.String(), nil
}
