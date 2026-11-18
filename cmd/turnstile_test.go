package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/livechat"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/abhinavxd/libredesk/internal/setting"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/zerodha/logf"
)

func TestChatInitNeedsTurnstile(t *testing.T) {
	tests := []struct {
		name      string
		site      string
		secret    string
		contactID int
		visitor   bool
		want      bool
	}{
		{name: "off when both keys are empty", site: " ", secret: "  ", contactID: 0, visitor: true, want: false},
		{name: "site key with an unusable secret still checks", site: "site", secret: "", contactID: 0, visitor: true, want: true},
		{name: "signed-in user skips an unusable secret", site: "site", secret: "", contactID: 4, visitor: false, want: false},
		{name: "new visitor", site: "site", secret: "sek", contactID: 0, visitor: true, want: true},
		{name: "returning visitor", site: "site", secret: "sek", contactID: 4, visitor: true, want: true},
		{name: "signed-in user", site: "site", secret: "sek", contactID: 4, visitor: false, want: false},
		{name: "no session is a visitor even if the flag is false", site: "site", secret: "sek", contactID: 0, visitor: false, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := chatInitNeedsTurnstile(tt.site, tt.secret, tt.contactID, tt.visitor); got != tt.want {
				t.Fatalf("chatInitNeedsTurnstile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEnforceChatInitTurnstileReturnsEnvelope(t *testing.T) {
	lo := logf.New(logf.Opts{Level: logf.FatalLevel})
	app := &App{lo: &lo, i18n: testutil.NewI18n(t)}
	original := turnstileHTTPClient
	turnstileHTTPClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Error("siteverify was called")
		return nil, errors.New("siteverify was called")
	})}
	t.Cleanup(func() { turnstileHTTPClient = original })

	// A nil request panics if the rejection is written with SendErrorEnvelope.
	err := enforceChatInitTurnstile(nil, app, livechat.Config{TurnstileSiteKey: "site", TurnstileSecretKey: "sek"}, "", "203.0.113.8", 0, true, 7)
	env, ok := err.(envelope.Error)
	if !ok {
		t.Fatalf("error = %T %v, want envelope.Error", err, err)
	}
	if env.ErrorType != envelope.InputError || env.Code != http.StatusBadRequest {
		t.Fatalf("envelope = %#v", env)
	}
	data, _ := env.Data.(map[string]any)
	if data["code"] != turnstileRequiredCode {
		t.Fatalf("data = %#v", env.Data)
	}

	err = enforceChatInitTurnstile(nil, app, livechat.Config{TurnstileSiteKey: "site"}, "", "", 0, true, 7)
	if _, ok := err.(envelope.Error); !ok {
		t.Fatalf("unusable secret error = %T %v", err, err)
	}
	if err := enforceChatInitTurnstile(nil, app, livechat.Config{TurnstileSiteKey: "site"}, "", "", 4, false, 7); err != nil {
		t.Fatalf("signed-in user error = %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestValidateInboxRejectsTurnstileKeyAlias(t *testing.T) {
	db := testutil.NewDB(t, "turnstile_alias")
	lo := logf.New(logf.Opts{Level: logf.FatalLevel})
	settings, err := setting.New(setting.Opts{DB: db, Lo: &lo})
	if err != nil {
		t.Fatal(err)
	}
	app := &App{i18n: testutil.NewI18n(t), lo: &lo, setting: settings}
	valid := json.RawMessage(`{"theme":"light","campaign_cooldown":"0s","launcher":{"position":"right","icon_scale":100,"spacing":{"side":0,"bottom":0}},"branding":{"light":{"colors":{"primary":"#111111"}},"dark":{"colors":{"primary":"#222222"}}},"turnstile_site_key":"site","turnstile_secret_key":"sek"}`)
	inbox := imodels.Inbox{Name: "Chat", Channel: livechat.ChannelLiveChat, Config: valid}
	if err := validateInbox(app, inbox, false); err != nil {
		t.Fatal(err)
	}
	inbox.Config = bytes.ReplaceAll(valid, []byte(`"turnstile_secret_key"`), []byte(`"Turnstile_Secret_Key"`))
	err = validateInbox(app, inbox, false)
	env, ok := err.(envelope.Error)
	if !ok || env.ErrorType != envelope.InputError {
		t.Fatalf("alias error = %T %v", err, err)
	}
}

func TestDuplicateTurnstileSecretDecodersDisagree(t *testing.T) {
	raw := []byte(`{"turnstile_site_key":"site","turnstile_secret_key":"sek","turnstile_secret_key":null}`)
	var cfg livechat.Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	// The struct keeps the first string, so validation of that struct still sees a secret.
	// The map keeps null. encryptInboxConfig rejects that map before it is stored.
	if cfg.TurnstileSecretKey != "sek" || decoded["turnstile_secret_key"] != nil {
		t.Fatalf("struct secret %q, map secret %#v", cfg.TurnstileSecretKey, decoded["turnstile_secret_key"])
	}
}

func TestTurnstileKeyAlias(t *testing.T) {
	alias, err := turnstileKeyAlias(json.RawMessage(`{"turnstile_site_key":"site","Turnstile_Secret_Key":"sek"}`))
	if err != nil || alias != "Turnstile_Secret_Key" {
		t.Fatalf("alias = %q, err = %v", alias, err)
	}
	alias, err = turnstileKeyAlias(json.RawMessage(`{"turnstile_site_key":"site","turnstile_secret_key":"sek"}`))
	if err != nil || alias != "" {
		t.Fatalf("canonical alias = %q, err = %v", alias, err)
	}
}

func TestChatSettingsHidesTurnstileSecret(t *testing.T) {
	raw, err := json.Marshal(chatSettingsResponse{
		Config: livechat.Config{
			TurnstileSiteKey:   "site-key",
			TurnstileSecretKey: "secret-key",
			TrustedDomains:     []string{"example.com"},
			Theme:              "light",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	if body["turnstile_site_key"] != "site-key" {
		t.Fatalf("site key = %#v", body["turnstile_site_key"])
	}
	if body["theme"] != "light" {
		t.Fatalf("theme = %#v", body["theme"])
	}
	if _, ok := body["turnstile_secret_key"]; ok {
		t.Fatal("turnstile secret leaked in widget settings")
	}
	if _, ok := body["trusted_domains"]; ok {
		t.Fatal("trusted domains leaked in widget settings")
	}
}

func TestVerifyTurnstileToken(t *testing.T) {
	var mu sync.Mutex
	var gotSecret, gotToken, gotIP string
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		mu.Lock()
		calls++
		gotSecret = r.PostForm.Get("secret")
		gotToken = r.PostForm.Get("response")
		gotIP = r.PostForm.Get("remoteip")
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
	}))
	t.Cleanup(srv.Close)

	err := verifyTurnstileToken(context.Background(), srv.Client(), srv.URL, "sek", "tok", "203.0.113.8")
	if err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	if calls != 1 || gotSecret != "sek" || gotToken != "tok" || gotIP != "203.0.113.8" {
		t.Fatalf("calls=%d secret=%q token=%q ip=%q", calls, gotSecret, gotToken, gotIP)
	}
	mu.Unlock()

	if err := verifyTurnstileToken(context.Background(), srv.Client(), srv.URL, "sek", "  ", "203.0.113.8"); !errors.Is(err, errTurnstileRejected) {
		t.Fatalf("empty token error = %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if calls != 1 {
		t.Fatalf("empty token made %d siteverify calls", calls)
	}
}

func TestVerifyTurnstileTokenRejects(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"success": false, "error-codes": []string{"invalid-input-response"}})
	}))
	t.Cleanup(srv.Close)

	err := verifyTurnstileToken(context.Background(), srv.Client(), srv.URL, "sek", "tok", "")
	if !errors.Is(err, errTurnstileRejected) {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(err.Error(), "invalid-input-response") {
		t.Fatalf("error = %v, want cloudflare error code", err)
	}
}

func TestVerifyTurnstileTokenUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	t.Cleanup(srv.Close)

	err := verifyTurnstileToken(context.Background(), srv.Client(), srv.URL, "sek", "tok", "")
	if !errors.Is(err, errTurnstileUnavailable) {
		t.Fatalf("error = %v", err)
	}
}
