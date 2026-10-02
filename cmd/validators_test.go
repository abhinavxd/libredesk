package main

import (
	"encoding/json"
	"io"

	"testing"

	"github.com/jakedolan443/fernmail/internal/envelope"

	imodels "github.com/jakedolan443/fernmail/internal/inbox/models"

	"github.com/jakedolan443/fernmail/internal/testutil"
	umodels "github.com/jakedolan443/fernmail/internal/user/models"
	wmodels "github.com/jakedolan443/fernmail/internal/webhook/models"
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
		{"retired reassigning availability status", agentReq{Email: "agent@example.com", FirstName: "Ada", Roles: []string{"Agent"}, AvailabilityStatus: umodels.AwayAndReassigning}, true},
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
		{"unknown event", wmodels.Webhook{Name: "hook", URL: "https://example.com/hook", Events: []string{"not.an.event"}}, true},
		{"retired tag event", wmodels.Webhook{Name: "hook", URL: "https://example.com/hook", Events: []string{"conversation.tags_changed"}}, true},
		{"retired assignment event", wmodels.Webhook{Name: "hook", URL: "https://example.com/hook", Events: []string{"conversation.assigned"}}, true},
		{"mixed supported and unsupported events", wmodels.Webhook{Name: "hook", URL: "https://example.com/hook", Events: []string{"message.created", "not.an.event"}}, true},
		{"all retained events", wmodels.Webhook{Name: "hook", URL: "https://example.com/hook", Events: []string{"conversation.created", "conversation.status_changed", "message.created", "message.updated"}}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertValidation(t, validateWebhook(app, tc.webhook), tc.wantErr)
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
