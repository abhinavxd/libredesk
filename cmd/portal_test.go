package main

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"github.com/abhinavxd/libredesk/internal/oidc"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"

	authpkg "github.com/abhinavxd/libredesk/internal/auth"
	"github.com/alicebob/miniredis/v2"
	"github.com/go-jose/go-jose/v4"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/helpcenter"
	"github.com/abhinavxd/libredesk/internal/portalform"
	pfmodels "github.com/abhinavxd/libredesk/internal/portalform/models"
	"github.com/zerodha/logf"

	"github.com/abhinavxd/libredesk/internal/testutil"
)

type portalTestSettings struct{ rootURL string }

func (s portalTestSettings) GetAppRootURL() (string, error) { return s.rootURL, nil }

func TestPortalStatusLabel(t *testing.T) {
	lcl := testutil.NewI18n(t)
	tests := []struct {
		category   string
		lastSender string
		wantLabel  string
		wantClass  string
	}{
		{"open", "contact", "In progress", "open"},
		{"open", "", "In progress", "open"},
		{"open", "agent", "Awaiting your reply", "waiting"},
		{"waiting", "agent", "Awaiting your reply", "waiting"},
		{"waiting", "contact", "In progress", "open"},
		{"resolved", "agent", "Resolved", "resolved"},
		{"resolved", "contact", "Resolved", "resolved"},
	}
	for _, tt := range tests {
		label, class := portalStatusLabel(lcl, tt.category, tt.lastSender)
		if label != tt.wantLabel || class != tt.wantClass {
			t.Errorf("portalStatusLabel(%q, %q) = (%q, %q), want (%q, %q)",
				tt.category, tt.lastSender, label, class, tt.wantLabel, tt.wantClass)
		}
	}
}

func TestPortalContactFirstName(t *testing.T) {
	tests := []struct{ email, want string }{
		{"jane@example.com", "jane"},
		{"@example.com", "@example.com"},
	}
	for _, tt := range tests {
		if got := portalContactFirstName(tt.email); got != tt.want {
			t.Errorf("portalContactFirstName(%q) = %q, want %q", tt.email, got, tt.want)
		}
	}
}

func TestPortalReturnTo(t *testing.T) {
	const base = "https://support.example.com"
	tests := []struct{ raw, want string }{
		{"", "/portal"},
		{"/portal", "/portal"},
		{"/portal/tickets/42", "/portal/tickets/42"},
		{"/portal/tickets/new?article=refunds", "/portal/tickets/new?article=refunds"},
		{"/portal/tickets/new?article=refunds&article_locale=fr", "/portal/tickets/new?article=refunds&article_locale=fr"},
		{"  /portal  ", "/portal"},
		{"https://support.example.com/portal/tickets/42", "/portal/tickets/42"},
		// Anything outside /portal is not a portal page, so it is not a redirect target.
		{"/hc/docs/en", "/portal"},
		{"https://support.example.com/hc/docs/en", "/portal"},
		{"/portalish", "/portal"},
		{"//evil.example.com/portal", "/portal"},
		{"/\\evil.example.com", "/portal"},
		{"https://support.example.com.evil.com/portal", "/portal"},
		{"https://evil.example.com/portal", "/portal"},
		{"javascript:alert(1)", "/portal"},
		{"/portal/x\r\nSet-Cookie: a=b", "/portal"},
	}
	for _, tt := range tests {
		if got := portalReturnTo(base, tt.raw); got != tt.want {
			t.Errorf("portalReturnTo(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
	if got := portalReturnTo("", "https://support.example.com/portal"); got != "/portal" {
		t.Errorf("portalReturnTo with no base URL = %q, want /portal", got)
	}
}

func TestNormalizePortalEmail(t *testing.T) {
	if got := normalizePortalEmail("  Jane@Example.COM "); got != "jane@example.com" {
		t.Errorf("normalizePortalEmail = %q", got)
	}
}

func TestMatchAcceptLanguage(t *testing.T) {
	allowed := []string{"en-US", "fr", "de"}
	tests := []struct {
		header string
		want   string
		wantOK bool
	}{
		{"fr", "fr", true},
		{"FR", "fr", true},
		{"fr-CA", "fr", true},
		{"en-GB,en;q=0.9", "en-US", true},
		{"de;q=0.2,fr;q=0.9", "fr", true},
		{"de;q=0,fr;q=0.1", "fr", true},
		{"ja,ko;q=0.8", "", false},
		{"", "", false},
		{"*", "", false},
	}
	for _, tt := range tests {
		got, ok := matchAcceptLanguage(tt.header, allowed)
		if got != tt.want || ok != tt.wantOK {
			t.Errorf("matchAcceptLanguage(%q) = (%q, %v), want (%q, %v)", tt.header, got, ok, tt.want, tt.wantOK)
		}
	}
}

func TestPortalFormAnswersOptions(t *testing.T) {
	for _, tc := range []struct {
		name, fieldType, value string
		options                []string
		required, wantError    bool
	}{
		{"text with stale options", "text", "new answer", []string{"old"}, false, false},
		{"number with stale options", "number", "0", []string{"old"}, false, false},
		{"negative number", "number", "-12", []string{"old"}, false, false},
		{"select member", "select", "yes", []string{"yes", "no"}, false, false},
		{"select nonmember", "select", "other", []string{"yes", "no"}, false, true},
		{"select empty options", "select", "yes", nil, false, true},
		{"select duplicate options", "select", "yes", []string{"yes", "yes"}, false, false},
		{"optional empty", "select", "", []string{"yes"}, false, false},
		{"required empty", "select", "", []string{"yes"}, true, true},
		{"oversized text", "text", strings.Repeat("x", 501), nil, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := request("/portal/tickets", "")
			r.RequestCtx.Request.Header.SetMethod("POST")
			r.RequestCtx.Request.PostArgs().Set("field_answer", tc.value)
			form := pfmodels.Form{Fields: []pfmodels.Field{{Key: "answer", Label: "Answer", Type: tc.fieldType, Target: pfmodels.TargetAttribute, AttributeKey: "answer", Options: tc.options, Required: tc.required}}}
			_, _, key, _ := portalFormAnswers(r, form)
			if (key != "") != tc.wantError {
				t.Fatalf("error = %q, wantError %v", key, tc.wantError)
			}
		})
	}
}

func TestPortalArticleLocaleAndPublication(t *testing.T) {
	db := testutil.NewDB(t, "portal_article")
	lo := logf.New(logf.Opts{})
	lcl := testutil.NewI18n(t)
	hc, err := helpcenter.New(helpcenter.Opts{DB: db, Lo: &lo, I18n: lcl})
	if err != nil {
		t.Fatal(err)
	}
	forms, err := portalform.New(portalform.Opts{DB: db, Lo: &lo, I18n: lcl})
	if err != nil {
		t.Fatal(err)
	}
	db.MustExec(`INSERT INTO portal_forms (id, name) VALUES (1, 'Default'), (2, 'French')`)
	db.MustExec(`INSERT INTO help_centers (id, name, slug, default_locale, allowed_locales) VALUES (1, 'Help', 'help', 'en', '["en", "fr"]')`)
	db.MustExec(`INSERT INTO article_collections (id, help_center_id, slug, locale, name, is_published) VALUES (1, 1, 'help', 'fr', 'Help', true), (2, 1, 'hidden', 'fr', 'Hidden', false)`)
	db.MustExec(`INSERT INTO help_articles (collection_id, slug, locale, title, status, portal_form_id) VALUES (1, 'refunds', 'fr', 'Remboursements', 'published', 2), (1, 'draft', 'fr', 'Draft', 'draft', 2), (2, 'hidden', 'fr', 'Hidden', 'published', 2)`)
	app := &App{helpcenter: hc, portalForm: forms}
	app.consts.Store(&constants{PortalHelpCenterID: 1, PortalFormID: 1})
	for _, tc := range []struct {
		slug, locale string
		valid        bool
	}{
		{"refunds", "fr", true}, {"refunds", "en", false}, {"refunds", "", false},
		{"refunds", "de", false}, {"draft", "fr", false}, {"hidden", "fr", false},
		{"missing", "fr", false}, {"", "fr", false},
	} {
		t.Run(tc.slug+"/"+tc.locale, func(t *testing.T) {
			article := portalArticle(app, tc.slug, tc.locale)
			if (article.ID > 0) != tc.valid {
				t.Fatalf("article = %+v", article)
			}
			form := portalTicketForm(app, article)
			if tc.valid && (article.Title != "Remboursements" || form.ID != 2) {
				t.Fatalf("article/form mismatch: %+v / %+v", article, form)
			}
		})
	}
	db.MustExec(`UPDATE help_centers SET is_active = false WHERE id = 1`)
	if article := portalArticle(app, "refunds", "fr"); article.ID != 0 {
		t.Fatal("inactive help center accepted")
	}
}

func TestPortalOIDCLoginRedirectsToCallbackOrigin(t *testing.T) {
	db := testutil.NewDB(t, "portal_oidc_origin")
	lo := logf.New(logf.Opts{})
	providers, err := oidc.New(oidc.Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t)}, portalTestSettings{rootURL: "https://app.example.com"})
	if err != nil {
		t.Fatal(err)
	}
	db.MustExec(`INSERT INTO oidc (id, name, provider, provider_url, client_id, client_secret, enabled_for_portal) VALUES (1, 'Provider', 'custom', 'https://id.example.com', '', '', true)`)
	app := &App{oidc: providers, lo: &lo}
	app.consts.Store(&constants{PortalEnabled: true, AppBaseURL: "https://app.example.com"})
	for _, returnTo := range []string{"/portal/tickets/new?article=refunds&article_locale=fr", "https://evil.example.com/portal"} {
		r := request("https://help.example.com/portal/oidc/1/login?return="+url.QueryEscape(returnTo), "")
		r.Context = app
		r.RequestCtx.SetUserValue("id", "1")
		if err := handlePortalOIDCLogin(r); err != nil {
			t.Fatal(err)
		}
		if r.RequestCtx.Response.StatusCode() != 303 {
			t.Fatal("expected redirect")
		}
		target, err := url.Parse(string(r.RequestCtx.Response.Header.Peek("Location")))
		if err != nil {
			t.Fatal(err)
		}
		if target.Scheme != "https" || target.Host != "app.example.com" || target.Path != "/portal/oidc/1/login" {
			t.Fatalf("wrong callback origin: %s", target)
		}
		if target.Query().Get("return") != portalReturnTo("https://app.example.com", returnTo) {
			t.Fatalf("wrong return path: %s", target)
		}
		if len(r.RequestCtx.Response.Header.Peek("Set-Cookie")) != 0 {
			t.Fatal("state cookie was set before reaching callback origin")
		}
	}
}

func TestPortalOIDCRejectsUnverifiedEmail(t *testing.T) {
	db := testutil.NewDB(t, "portal_oidc_email")
	lo := logf.New(logf.Opts{})
	lcl := testutil.NewI18n(t)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	var issuer string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			json.NewEncoder(w).Encode(map[string]any{"issuer": issuer, "authorization_endpoint": issuer + "/auth", "token_endpoint": issuer + "/token", "jwks_uri": issuer + "/keys", "id_token_signing_alg_values_supported": []string{"RS256"}})
		case "/keys":
			json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &key.PublicKey, KeyID: "test", Algorithm: "RS256", Use: "sig"}}})
		case "/token":
			claims := jwt.MapClaims{"iss": issuer, "aud": "portal-test", "sub": "customer", "exp": time.Now().Add(time.Minute).Unix(), "nonce": "test-nonce", "email": "customer@example.com"}
			if r.FormValue("code") == "unverified" {
				claims["email_verified"] = false
			}
			token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
			token.Header["kid"] = "test"
			signed, err := token.SignedString(key)
			if err != nil {
				t.Error(err)
				w.WriteHeader(500)
				return
			}
			json.NewEncoder(w).Encode(map[string]any{"access_token": "access", "token_type": "Bearer", "id_token": signed})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	issuer = server.URL
	mr := miniredis.RunT(t)
	rd := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { rd.Close() })
	auth, err := authpkg.New(authpkg.Config{Providers: []authpkg.Provider{{ID: 1, ProviderURL: issuer, ClientID: "portal-test", RedirectURL: "https://app.example.com/api/v1/oidc/1/finish"}}}, lcl, rd, &lo, nil)
	if err != nil {
		t.Fatal(err)
	}
	providers, err := oidc.New(oidc.Opts{DB: db, Lo: &lo, I18n: lcl}, portalTestSettings{rootURL: "https://app.example.com"})
	if err != nil {
		t.Fatal(err)
	}
	db.MustExec(`INSERT INTO oidc (id, name, provider, provider_url, client_id, client_secret, enabled_for_portal) VALUES (1, 'Provider', 'custom', $1, '', '', true)`, issuer)
	app := &App{oidc: providers, auth: auth, redis: rd, lo: &lo}
	app.consts.Store(&constants{PortalEnabled: true, AppBaseURL: "https://app.example.com"})
	for _, code := range []string{"unverified", "missing"} {
		t.Run(code, func(t *testing.T) {
			_, claims, err := auth.ExchangeOIDCToken(t.Context(), 1, code, "test-verifier", "test-nonce")
			if err != nil || claims.Email != "customer@example.com" {
				t.Fatalf("test token did not validate: %+v, %v", claims, err)
			}
			state := portalOIDCStatePrefix + code
			if err := rd.HSet(t.Context(), portalOIDCFlowPrefix+state, map[string]any{"nonce": "test-nonce", "provider_id": "1", "code_verifier": "test-verifier"}).Err(); err != nil {
				t.Fatal(err)
			}
			r := request("https://app.example.com/api/v1/oidc/1/finish?code="+code+"&state="+state, "")
			r.RequestCtx.Init2(nil, nil, false)
			r.Context = app
			r.RequestCtx.SetUserValue("id", "1")
			r.RequestCtx.Request.Header.SetCookie(portalOIDCStateCookie, state)
			if err := handlePortalOIDCCallback(r); err != nil {
				t.Fatal(err)
			}
			if got := string(r.RequestCtx.Response.Header.Peek("Location")); !strings.HasSuffix(got, "/portal/login?error=oidc") {
				t.Fatalf("unverified email accepted: %s", got)
			}
			if keys := mr.Keys(); len(keys) != 0 {
				t.Fatalf("unexpected session state: %v", keys)
			}
		})
	}
}
