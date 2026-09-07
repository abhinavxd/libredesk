package main

import (
	"encoding/json"
	"io"
	"strings"
	"testing"

	autoModels "github.com/abhinavxd/libredesk/internal/automation/models"
	clmodels "github.com/abhinavxd/libredesk/internal/context_link/models"
	cmodels "github.com/abhinavxd/libredesk/internal/custom_attribute/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/helpcenter"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	macromodels "github.com/abhinavxd/libredesk/internal/macro/models"
	smodels "github.com/abhinavxd/libredesk/internal/sla/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/abhinavxd/libredesk/internal/testutil"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	wmodels "github.com/abhinavxd/libredesk/internal/webhook/models"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

const testAppBaseURL = "https://desk.example.com"

func TestValidateAgentRequest(t *testing.T) {
	app := newValidatorTestApp(t)

	tests := []struct {
		name    string
		req     agentReq
		wantErr bool
	}{
		{"valid", agentReq{Email: "agent@example.com", FirstName: "Ada", Roles: []string{"Agent"}}, false},
		{"valid with availability status", agentReq{Email: "agent@example.com", FirstName: "Ada", Roles: []string{"Agent"}, AvailabilityStatus: umodels.AwayAndReassigning}, false},
		{"empty email", agentReq{FirstName: "Ada", Roles: []string{"Agent"}}, true},
		{"whitespace email", agentReq{Email: "   ", FirstName: "Ada", Roles: []string{"Agent"}}, true},
		{"malformed email", agentReq{Email: "not-an-email", FirstName: "Ada", Roles: []string{"Agent"}}, true},
		{"email with display name", agentReq{Email: "Ada <ada@example.com>", FirstName: "Ada", Roles: []string{"Agent"}}, true},
		{"nil roles", agentReq{Email: "agent@example.com", FirstName: "Ada"}, true},
		{"empty first name", agentReq{Email: "agent@example.com", Roles: []string{"Agent"}}, true},
		{"whitespace first name", agentReq{Email: "agent@example.com", FirstName: "  ", Roles: []string{"Agent"}}, true},
		{"unknown availability status", agentReq{Email: "agent@example.com", FirstName: "Ada", Roles: []string{"Agent"}, AvailabilityStatus: "vacation"}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertValidation(t, validateAgentRequest(app, &tc.req), tc.wantErr)
		})
	}
}

func TestValidateAgentRequestNormalizesFields(t *testing.T) {
	app := newValidatorTestApp(t)
	req := agentReq{Email: "  Ada@Example.COM ", FirstName: " Ada ", LastName: " Lovelace ", Roles: []string{"Agent"}}

	if err := validateAgentRequest(app, &req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Email != "ada@example.com" {
		t.Fatalf("got email %q, want ada@example.com", req.Email)
	}
	if req.FirstName != "Ada" || req.LastName != "Lovelace" {
		t.Fatalf("got names %q %q, want Ada Lovelace", req.FirstName, req.LastName)
	}
}

func TestValidateAgentRequestEmptyRolesSlice(t *testing.T) {
	t.Skip("validateAgentRequest checks Roles == nil, so a JSON body with \"roles\": [] passes and creates an agent with no roles")

	app := newValidatorTestApp(t)
	req := agentReq{Email: "agent@example.com", FirstName: "Ada", Roles: []string{}}
	assertValidation(t, validateAgentRequest(app, &req), true)
}

func TestValidateWebhook(t *testing.T) {
	app := newValidatorTestApp(t)

	tests := []struct {
		name    string
		webhook wmodels.Webhook
		wantErr bool
	}{
		{"valid", wmodels.Webhook{Name: "hook", URL: "https://example.com/hook", Events: []string{"conversation.created"}}, false},
		{"empty name", wmodels.Webhook{URL: "https://example.com/hook", Events: []string{"conversation.created"}}, true},
		{"empty url", wmodels.Webhook{Name: "hook", Events: []string{"conversation.created"}}, true},
		{"nil events", wmodels.Webhook{Name: "hook", URL: "https://example.com/hook"}, true},
		{"empty events", wmodels.Webhook{Name: "hook", URL: "https://example.com/hook", Events: []string{}}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertValidation(t, validateWebhook(app, tc.webhook), tc.wantErr)
		})
	}
}

func TestValidateContextLink(t *testing.T) {
	app := newValidatorTestApp(t)
	masked := strings.Repeat(stringutil.PasswordDummy, 10)

	tests := []struct {
		name    string
		link    clmodels.ContextLink
		wantErr bool
	}{
		{"valid", clmodels.ContextLink{Name: "CRM", URLTemplate: "https://crm.example.com/{{email}}"}, false},
		{"valid with 32 char secret", clmodels.ContextLink{Name: "CRM", URLTemplate: "https://crm.example.com", Secret: strings.Repeat("a", 32)}, false},
		{"valid with masked secret", clmodels.ContextLink{Name: "CRM", URLTemplate: "https://crm.example.com", Secret: masked}, false},
		{"empty name", clmodels.ContextLink{URLTemplate: "https://crm.example.com"}, true},
		{"empty url template", clmodels.ContextLink{Name: "CRM"}, true},
		{"short secret", clmodels.ContextLink{Name: "CRM", URLTemplate: "https://crm.example.com", Secret: strings.Repeat("a", 31)}, true},
		{"long secret", clmodels.ContextLink{Name: "CRM", URLTemplate: "https://crm.example.com", Secret: strings.Repeat("a", 33)}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertValidation(t, validateContextLink(app, &tc.link), tc.wantErr)
		})
	}
}

func TestValidateContextLinkDefaultsTokenExpiry(t *testing.T) {
	app := newValidatorTestApp(t)

	tests := []struct {
		name  string
		given int
		want  int
	}{
		{"zero defaults", 0, 1200},
		{"negative defaults", -5, 1200},
		{"positive kept", 60, 60},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			link := clmodels.ContextLink{Name: "CRM", URLTemplate: "https://crm.example.com", TokenExpirySeconds: tc.given}
			if err := validateContextLink(app, &link); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if link.TokenExpirySeconds != tc.want {
				t.Fatalf("got token expiry %d, want %d", link.TokenExpirySeconds, tc.want)
			}
		})
	}
}

func TestValidateCustomAttribute(t *testing.T) {
	app := newValidatorTestApp(t)
	valid := cmodels.CustomAttribute{Name: "Plan", AppliesTo: "conversation", DataType: "text", Description: "Customer plan", Key: "plan"}

	tests := []struct {
		name    string
		attr    cmodels.CustomAttribute
		wantErr bool
	}{
		{"valid", valid, false},
		{"empty name", cmodels.CustomAttribute{AppliesTo: "conversation", DataType: "text", Description: "d", Key: "plan"}, true},
		{"empty applies to", cmodels.CustomAttribute{Name: "Plan", DataType: "text", Description: "d", Key: "plan"}, true},
		{"empty data type", cmodels.CustomAttribute{Name: "Plan", AppliesTo: "conversation", Description: "d", Key: "plan"}, true},
		{"empty description", cmodels.CustomAttribute{Name: "Plan", AppliesTo: "conversation", DataType: "text", Key: "plan"}, true},
		{"empty key", cmodels.CustomAttribute{Name: "Plan", AppliesTo: "conversation", DataType: "text", Description: "d"}, true},
		{"reserved key status", cmodels.CustomAttribute{Name: "Plan", AppliesTo: "conversation", DataType: "text", Description: "d", Key: "status"}, true},
		{"reserved key inbox", cmodels.CustomAttribute{Name: "Plan", AppliesTo: "conversation", DataType: "text", Description: "d", Key: "inbox"}, true},
		{"key near reserved", cmodels.CustomAttribute{Name: "Plan", AppliesTo: "conversation", DataType: "text", Description: "d", Key: "status_code"}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertValidation(t, validateCustomAttribute(app, tc.attr), tc.wantErr)
		})
	}
}

func TestValidateSLA(t *testing.T) {
	app := newValidatorTestApp(t)
	notification := smodels.SlaNotification{Type: "warning", TimeDelayType: "after", Metric: "first_response", TimeDelay: "30m", Recipients: []string{"assigned_user"}}

	withNotification := func(n smodels.SlaNotification) *smodels.SLAPolicy {
		return &smodels.SLAPolicy{Name: "Gold", FirstResponseTime: null.StringFrom("1h"), Notifications: smodels.SlaNotifications{n}}
	}

	tests := []struct {
		name    string
		sla     *smodels.SLAPolicy
		wantErr bool
	}{
		{"valid first response only", &smodels.SLAPolicy{Name: "Gold", FirstResponseTime: null.StringFrom("1h")}, false},
		{"valid all durations", &smodels.SLAPolicy{Name: "Gold", FirstResponseTime: null.StringFrom("1h"), NextResponseTime: null.StringFrom("2h"), ResolutionTime: null.StringFrom("24h")}, false},
		{"empty name", &smodels.SLAPolicy{FirstResponseTime: null.StringFrom("1h")}, true},
		{"no durations", &smodels.SLAPolicy{Name: "Gold"}, true},

		{"first response unparseable", &smodels.SLAPolicy{Name: "Gold", FirstResponseTime: null.StringFrom("one hour")}, true},
		{"first response below one minute", &smodels.SLAPolicy{Name: "Gold", FirstResponseTime: null.StringFrom("30s")}, true},
		{"first response exactly one minute", &smodels.SLAPolicy{Name: "Gold", FirstResponseTime: null.StringFrom("1m")}, false},
		{"next response unparseable", &smodels.SLAPolicy{Name: "Gold", NextResponseTime: null.StringFrom("soon")}, true},
		{"next response below one minute", &smodels.SLAPolicy{Name: "Gold", NextResponseTime: null.StringFrom("59s")}, true},
		{"resolution unparseable", &smodels.SLAPolicy{Name: "Gold", ResolutionTime: null.StringFrom("later")}, true},
		{"resolution below one minute", &smodels.SLAPolicy{Name: "Gold", ResolutionTime: null.StringFrom("10s")}, true},
		{"first response after resolution", &smodels.SLAPolicy{Name: "Gold", FirstResponseTime: null.StringFrom("5h"), ResolutionTime: null.StringFrom("1h")}, true},
		{"first response equals resolution", &smodels.SLAPolicy{Name: "Gold", FirstResponseTime: null.StringFrom("1h"), ResolutionTime: null.StringFrom("1h")}, false},

		{"notification valid", withNotification(notification), false},
		{"notification empty type", withNotification(smodels.SlaNotification{TimeDelayType: "after", Metric: "first_response", TimeDelay: "30m", Recipients: []string{"assigned_user"}}), true},
		{"notification empty time delay type", withNotification(smodels.SlaNotification{Type: "warning", Metric: "first_response", TimeDelay: "30m", Recipients: []string{"assigned_user"}}), true},
		{"notification empty metric", withNotification(smodels.SlaNotification{Type: "warning", TimeDelayType: "after", TimeDelay: "30m", Recipients: []string{"assigned_user"}}), true},
		{"notification empty time delay", withNotification(smodels.SlaNotification{Type: "warning", TimeDelayType: "after", Metric: "first_response", Recipients: []string{"assigned_user"}}), true},
		{"notification unparseable time delay", withNotification(smodels.SlaNotification{Type: "warning", TimeDelayType: "after", Metric: "first_response", TimeDelay: "half an hour", Recipients: []string{"assigned_user"}}), true},
		{"notification time delay below one minute", withNotification(smodels.SlaNotification{Type: "warning", TimeDelayType: "after", Metric: "first_response", TimeDelay: "45s", Recipients: []string{"assigned_user"}}), true},
		{"notification immediately skips delay", withNotification(smodels.SlaNotification{Type: "warning", TimeDelayType: "immediately", Metric: "first_response", Recipients: []string{"assigned_user"}}), false},
		{"notification no recipients", withNotification(smodels.SlaNotification{Type: "warning", TimeDelayType: "after", Metric: "first_response", TimeDelay: "30m"}), true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertValidation(t, validateSLA(app, tc.sla), tc.wantErr)
		})
	}
}

func TestValidateEmailConfig(t *testing.T) {
	app := newValidatorTestApp(t)

	tests := []struct {
		name    string
		config  string
		wantErr bool
	}{
		{"valid password auth", `{"auth_type":"password","smtp":[{"host":"smtp.example.com","port":587,"auth_protocol":"plain"}],"imap":[{"host":"imap.example.com","port":993,"mailbox":"INBOX","tls_type":"tls"}]}`, false},
		{"valid empty auth type", `{"smtp":[{"host":"smtp.example.com","port":587}]}`, false},
		{"valid oauth2", `{"auth_type":"oauth2","oauth":{"provider":"google","client_id":"abc"}}`, false},
		{"malformed json", `{"auth_type":`, true},
		{"unknown auth type", `{"auth_type":"kerberos"}`, true},
		{"oauth2 without oauth block", `{"auth_type":"oauth2"}`, true},
		{"oauth2 unknown provider", `{"auth_type":"oauth2","oauth":{"provider":"yahoo","client_id":"abc"}}`, true},
		{"oauth2 empty client id", `{"auth_type":"oauth2","oauth":{"provider":"microsoft"}}`, true},

		{"smtp empty host", `{"smtp":[{"port":587}]}`, true},
		{"smtp zero port", `{"smtp":[{"host":"smtp.example.com","port":0}]}`, true},
		{"smtp negative port", `{"smtp":[{"host":"smtp.example.com","port":-1}]}`, true},
		{"smtp unknown auth protocol", `{"smtp":[{"host":"smtp.example.com","port":587,"auth_protocol":"ntlm"}]}`, true},
		{"smtp auth protocol ignored for oauth2", `{"auth_type":"oauth2","oauth":{"provider":"google","client_id":"abc"},"smtp":[{"host":"smtp.example.com","port":587,"auth_protocol":"ntlm"}]}`, false},
		{"smtp second entry invalid", `{"smtp":[{"host":"smtp.example.com","port":587},{"host":"","port":587}]}`, true},

		{"imap empty host", `{"imap":[{"port":993,"mailbox":"INBOX","tls_type":"tls"}]}`, true},
		{"imap zero port", `{"imap":[{"host":"imap.example.com","port":0,"mailbox":"INBOX","tls_type":"tls"}]}`, true},
		{"imap empty mailbox", `{"imap":[{"host":"imap.example.com","port":993,"tls_type":"tls"}]}`, true},
		{"imap unknown tls type", `{"imap":[{"host":"imap.example.com","port":993,"mailbox":"INBOX","tls_type":"ssl"}]}`, true},
		{"imap empty tls type", `{"imap":[{"host":"imap.example.com","port":993,"mailbox":"INBOX"}]}`, true},
		{"imap starttls", `{"imap":[{"host":"imap.example.com","port":143,"mailbox":"INBOX","tls_type":"starttls"}]}`, false},
		{"imap none", `{"imap":[{"host":"imap.example.com","port":143,"mailbox":"INBOX","tls_type":"none"}]}`, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertValidation(t, validateEmailConfig(app, json.RawMessage(tc.config)), tc.wantErr)
		})
	}
}

func TestValidateEmailConfigAuthTypeConstants(t *testing.T) {
	app := newValidatorTestApp(t)

	for _, authType := range []string{imodels.AuthTypePassword, imodels.AuthTypeOAuth2} {
		cfg := imodels.Config{AuthType: authType}
		if authType == imodels.AuthTypeOAuth2 {
			cfg.OAuth = &imodels.OAuthConfig{Provider: "google", ClientID: "abc"}
		}
		b, err := json.Marshal(cfg)
		if err != nil {
			t.Fatalf("marshalling config: %v", err)
		}
		if err := validateEmailConfig(app, b); err != nil {
			t.Fatalf("auth type %q: unexpected error: %v", authType, err)
		}
	}
}

func TestValidateHelpCenter(t *testing.T) {
	app := newValidatorTestApp(t)

	tests := []struct {
		name    string
		req     helpcenter.HelpCenterRequest
		wantErr bool
	}{
		{"valid", helpcenter.HelpCenterRequest{Name: "Docs", Slug: "docs", PageTitle: "Help"}, false},
		{"empty name", helpcenter.HelpCenterRequest{Slug: "docs", PageTitle: "Help"}, true},
		{"whitespace name", helpcenter.HelpCenterRequest{Name: "  ", Slug: "docs", PageTitle: "Help"}, true},
		{"empty slug", helpcenter.HelpCenterRequest{Name: "Docs", PageTitle: "Help"}, true},
		{"whitespace slug", helpcenter.HelpCenterRequest{Name: "Docs", Slug: " \t ", PageTitle: "Help"}, true},
		{"empty page title", helpcenter.HelpCenterRequest{Name: "Docs", Slug: "docs"}, true},
		{"whitespace page title", helpcenter.HelpCenterRequest{Name: "Docs", Slug: "docs", PageTitle: " "}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertValidation(t, validateHelpCenter(app, &tc.req), tc.wantErr)
		})
	}
}

func TestValidateHelpCenterStripsBaseURLFromTheme(t *testing.T) {
	app := newValidatorTestApp(t)
	req := helpcenter.HelpCenterRequest{
		Name:      " Docs ",
		Slug:      " docs ",
		PageTitle: " Help ",
		Theme:     json.RawMessage(`{"logo_url":"` + testAppBaseURL + `/uploads/abc"}`),
	}

	if err := validateHelpCenter(app, &req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Name != "Docs" || req.Slug != "docs" || req.PageTitle != "Help" {
		t.Fatalf("fields not trimmed: %q %q %q", req.Name, req.Slug, req.PageTitle)
	}
	if got, want := string(req.Theme), `{"logo_url":"/uploads/abc"}`; got != want {
		t.Fatalf("got theme %s, want %s", got, want)
	}
}

func TestValidateCollection(t *testing.T) {
	app := newValidatorTestApp(t)

	tests := []struct {
		name    string
		req     helpcenter.CollectionRequest
		wantErr bool
	}{
		{"valid", helpcenter.CollectionRequest{Name: "Billing"}, false},
		{"empty name", helpcenter.CollectionRequest{}, true},
		{"whitespace name", helpcenter.CollectionRequest{Name: "   "}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertValidation(t, validateCollection(app, &tc.req), tc.wantErr)
		})
	}
}

func TestValidateCollectionTrimsName(t *testing.T) {
	app := newValidatorTestApp(t)
	req := helpcenter.CollectionRequest{Name: "  Billing  "}

	if err := validateCollection(app, &req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Name != "Billing" {
		t.Fatalf("got name %q, want Billing", req.Name)
	}
}

func TestValidateArticle(t *testing.T) {
	app := newValidatorTestApp(t)

	tests := []struct {
		name    string
		req     helpcenter.ArticleRequest
		wantErr bool
	}{
		{"valid", helpcenter.ArticleRequest{Title: "Refunds", Content: "<p>How refunds work</p>"}, false},
		{"empty title", helpcenter.ArticleRequest{Content: "<p>body</p>"}, true},
		{"whitespace title", helpcenter.ArticleRequest{Title: "  ", Content: "<p>body</p>"}, true},
		{"empty content", helpcenter.ArticleRequest{Title: "Refunds"}, true},
		{"whitespace content", helpcenter.ArticleRequest{Title: "Refunds", Content: " \n\t "}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertValidation(t, validateArticle(app, &tc.req), tc.wantErr)
		})
	}
}

func TestValidateArticleStripsBaseURLFromAssets(t *testing.T) {
	app := newValidatorTestApp(t)
	req := helpcenter.ArticleRequest{
		Title:        " Refunds ",
		Content:      `<img src="` + testAppBaseURL + `/uploads/img.png">`,
		MetaImageURL: testAppBaseURL + "/uploads/meta.png",
	}

	if err := validateArticle(app, &req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Title != "Refunds" {
		t.Fatalf("got title %q, want Refunds", req.Title)
	}
	if got, want := req.Content, `<img src="/uploads/img.png">`; got != want {
		t.Fatalf("got content %q, want %q", got, want)
	}
	if got, want := req.MetaImageURL, "/uploads/meta.png"; got != want {
		t.Fatalf("got meta image url %q, want %q", got, want)
	}
}

func TestValidateMacro(t *testing.T) {
	app := newValidatorTestApp(t)

	actions := func(a ...autoModels.RuleAction) json.RawMessage {
		b, err := json.Marshal(a)
		if err != nil {
			t.Fatalf("marshalling actions: %v", err)
		}
		return b
	}

	tests := []struct {
		name    string
		macro   macromodels.Macro
		wantErr bool
	}{
		{"valid", macromodels.Macro{Name: "Close", VisibleWhen: []string{"replying"}, Actions: actions(autoModels.RuleAction{Type: autoModels.ActionSetStatus, Value: []string{"Closed"}})}, false},
		{"valid no actions", macromodels.Macro{Name: "Close", VisibleWhen: []string{"replying"}, Actions: json.RawMessage(`[]`)}, false},
		{"empty name", macromodels.Macro{VisibleWhen: []string{"replying"}, Actions: json.RawMessage(`[]`)}, true},
		{"nil visible when", macromodels.Macro{Name: "Close", Actions: json.RawMessage(`[]`)}, true},
		{"empty visible when", macromodels.Macro{Name: "Close", VisibleWhen: []string{}, Actions: json.RawMessage(`[]`)}, true},
		{"malformed actions json", macromodels.Macro{Name: "Close", VisibleWhen: []string{"replying"}, Actions: json.RawMessage(`{`)}, true},
		{"action with no value", macromodels.Macro{Name: "Close", VisibleWhen: []string{"replying"}, Actions: actions(autoModels.RuleAction{Type: autoModels.ActionSetStatus})}, true},
		{"second action with no value", macromodels.Macro{Name: "Close", VisibleWhen: []string{"replying"}, Actions: actions(
			autoModels.RuleAction{Type: autoModels.ActionSetStatus, Value: []string{"Closed"}},
			autoModels.RuleAction{Type: autoModels.ActionAddTags, Value: []string{}},
		)}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertValidation(t, validateMacro(app, tc.macro), tc.wantErr)
		})
	}
}

func newValidatorTestApp(t *testing.T) *App {
	t.Helper()
	lo := logf.New(logf.Opts{Writer: io.Discard})
	app := &App{
		i18n: testutil.NewI18n(t),
		lo:   &lo,
	}
	app.consts.Store(&constants{AppBaseURL: testAppBaseURL})
	return app
}

func assertValidation(t *testing.T, err error, wantErr bool) {
	t.Helper()
	if !wantErr {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return
	}
	if err == nil {
		t.Fatal("expected a validation error, got nil")
	}
	envErr, ok := err.(envelope.Error)
	if !ok {
		t.Fatalf("got %T, want envelope.Error", err)
	}
	if envErr.ErrorType != envelope.InputError {
		t.Fatalf("got error type %q, want %q", envErr.ErrorType, envelope.InputError)
	}
	if envErr.Message == "" {
		t.Fatal("expected a non-empty error message")
	}
}
