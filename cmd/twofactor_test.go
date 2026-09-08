package main

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	activitylog "github.com/abhinavxd/libredesk/internal/activity_log"
	auth_ "github.com/abhinavxd/libredesk/internal/auth"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/abhinavxd/libredesk/internal/twofactor"
	"github.com/abhinavxd/libredesk/internal/user"
	"github.com/alicebob/miniredis/v2"
	"github.com/jmoiron/sqlx"
	"github.com/pquerna/otp/totp"
	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
	"golang.org/x/crypto/bcrypt"
)

func TestTwoFactorLoginRequiresSecondFactor(t *testing.T) {
	app, db, id := newTwoFactorTestApp(t, "twofactor_login")
	setup, err := app.twoFactor.BeginSetup(id, "agent@example.com")
	if err != nil {
		t.Fatal(err)
	}
	code, _ := totp.GenerateCode(setup.Secret, time.Now())
	codes, err := app.twoFactor.ConfirmSetup(id, code)
	if err != nil {
		t.Fatal(err)
	}
	login := twoFactorRequestForTest(app, map[string]string{"email": "agent@example.com", "password": "TestPassword123!"})
	if err := handleLogin(login); err != nil {
		t.Fatal(err)
	}
	var response struct {
		Data struct {
			Required bool `json:"two_factor_required"`
		}
	}
	if err := json.Unmarshal(login.RequestCtx.Response.Body(), &response); err != nil {
		t.Fatal(err)
	}
	if !response.Data.Required {
		t.Fatalf("no second-factor challenge: %s", login.RequestCtx.Response.Body())
	}
	var fullCookie fasthttp.Cookie
	fullCookie.SetKey("libredesk_session")
	if login.RequestCtx.Response.Header.Cookie(&fullCookie) {
		t.Fatal("password alone created a session")
	}
	var pending fasthttp.Cookie
	pending.SetKey("libredesk_two_factor")
	if !login.RequestCtx.Response.Header.Cookie(&pending) {
		t.Fatal("missing challenge cookie")
	}
	verify := twoFactorRequestForTest(app, map[string]string{"code": codes[0]})
	verify.RequestCtx.Request.Header.SetCookie("libredesk_two_factor", string(pending.Value()))
	verify.RequestCtx.Request.Header.SetCookie("csrf_token", "test")
	verify.RequestCtx.Request.Header.Set("X-CSRFTOKEN", "test")
	if _, err := authenticateUser(verify, app); err == nil {
		t.Fatal("pending challenge grants API access")
	}
	if err := handleVerifyTwoFactorLogin(verify); err != nil {
		t.Fatal(err)
	}
	if verify.RequestCtx.Response.StatusCode() != 200 || !verify.RequestCtx.Response.Header.Cookie(&fullCookie) {
		t.Fatalf("recovery login failed: %s", verify.RequestCtx.Response.Body())
	}
	repeat := twoFactorRequestForTest(app, map[string]string{"code": codes[1]})
	repeat.RequestCtx.Request.Header.SetCookie("libredesk_two_factor", string(pending.Value()))
	handleVerifyTwoFactorLogin(repeat)
	if repeat.RequestCtx.Response.StatusCode() != 401 {
		t.Fatal("challenge reused")
	}
	db.MustExec(`UPDATE user_two_factor SET recovery_hashes='{}' WHERE user_id=$1`, id)
	regenerate := twoFactorRequestForTest(app, map[string]string{"password": "TestPassword123!"})
	regenerate.RequestCtx.Request.Header.SetCookie("libredesk_session", string(fullCookie.Value()))
	regenerate.RequestCtx.Request.Header.SetCookie("csrf_token", "test")
	regenerate.RequestCtx.Request.Header.Set("X-CSRFTOKEN", "test")
	if err := auth(handleRegenerateTwoFactorCodes)(regenerate); err != nil {
		t.Fatal(err)
	}
	var recovery struct {
		Data struct {
			Codes []string `json:"recovery_codes"`
		}
	}
	if err := json.Unmarshal(regenerate.RequestCtx.Response.Body(), &recovery); err != nil || len(recovery.Data.Codes) != 10 {
		t.Fatalf("last recovery code did not allow recovery: %s %v", regenerate.RequestCtx.Response.Body(), err)
	}
	if err := app.auth.SetSessionValues(regenerate, map[string]any{"two_factor_verified_at": time.Now().Add(-6 * time.Minute).UTC().Format(time.RFC3339)}); err != nil {
		t.Fatal(err)
	}
	regenerate.RequestCtx.Response.Reset()
	if err := auth(handleRegenerateTwoFactorCodes)(regenerate); err != nil {
		t.Fatal(err)
	}
	if regenerate.RequestCtx.Response.StatusCode() == 200 {
		t.Fatal("expired verification allowed changing recovery codes without a factor")
	}
	var lastLogin bool
	if err := db.Get(&lastLogin, `SELECT last_login_at IS NOT NULL FROM users WHERE id=$1`, id); err != nil || !lastLogin {
		t.Fatal("completed login not recorded")
	}
}

func TestPendingLoginRejectsChangedPasswordAndDisabledAgent(t *testing.T) {
	app, db, id := newTwoFactorTestApp(t, "twofactor_login_change")
	setup, err := app.twoFactor.BeginSetup(id, "agent@example.com")
	if err != nil {
		t.Fatal(err)
	}
	code, _ := totp.GenerateCode(setup.Secret, time.Now())
	codes, err := app.twoFactor.ConfirmSetup(id, code)
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []string{"password", "enabled"} {
		db.MustExec(`UPDATE users SET enabled=true WHERE id=$1`, id)
		agent, err := app.user.GetAgent(id, "")
		if err != nil {
			t.Fatal(err)
		}
		begin := twoFactorRequestForTest(app, nil)
		if err := app.auth.BeginPendingLogin(begin, auth_.PendingLogin{UserID: id, CredentialHash: credentialHash(agent.Password.String)}); err != nil {
			t.Fatal(err)
		}
		var pending fasthttp.Cookie
		pending.SetKey("libredesk_two_factor")
		begin.RequestCtx.Response.Header.Cookie(&pending)
		if change == "password" {
			db.MustExec(`UPDATE users SET password='changed' WHERE id=$1`, id)
		} else {
			db.MustExec(`UPDATE users SET enabled=false WHERE id=$1`, id)
		}
		verify := twoFactorRequestForTest(app, map[string]string{"code": codes[0]})
		verify.RequestCtx.Request.Header.SetCookie("libredesk_two_factor", string(pending.Value()))
		handleVerifyTwoFactorLogin(verify)
		if verify.RequestCtx.Response.StatusCode() != 401 {
			t.Fatalf("%s change did not invalidate pending login: %s", change, verify.RequestCtx.Response.Body())
		}
	}
}

func TestOIDCLoginRequiresSecondFactor(t *testing.T) {
	app, _, id := newTwoFactorTestApp(t, "twofactor_oidc")
	setup, err := app.twoFactor.BeginSetup(id, "agent@example.com")
	if err != nil {
		t.Fatal(err)
	}
	code, _ := totp.GenerateCode(setup.Secret, time.Now())
	if _, err := app.twoFactor.ConfirmSetup(id, code); err != nil {
		t.Fatal(err)
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	var issuer string
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			json.NewEncoder(w).Encode(map[string]any{"issuer": issuer, "authorization_endpoint": issuer + "/authorize", "token_endpoint": issuer + "/token", "jwks_uri": issuer + "/keys", "id_token_signing_alg_values_supported": []string{"RS256"}})
		case "/keys":
			json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{"kty": "RSA", "kid": "test", "alg": "RS256", "use": "sig", "n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": "AQAB"}}})
		case "/token":
			token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{"iss": issuer, "aud": "libredesk", "sub": "agent", "exp": time.Now().Add(time.Minute).Unix(), "email": "agent@example.com", "email_verified": true})
			token.Header["kid"] = "test"
			signed, err := token.SignedString(key)
			if err != nil {
				t.Error(err)
				http.Error(w, "token failed", 500)
				return
			}
			json.NewEncoder(w).Encode(map[string]any{"access_token": "test", "token_type": "Bearer", "id_token": signed})
		}
	}))
	defer provider.Close()
	issuer = provider.URL
	if err := app.auth.Reload(auth_.Config{Providers: []auth_.Provider{{ID: 1, ProviderURL: issuer, ClientID: "libredesk", RedirectURL: func() (string, error) { return "http://localhost/callback", nil }}}}); err != nil {
		t.Fatal(err)
	}
	start := twoFactorRequestForTest(app, nil)
	if err := app.auth.SetSessionValues(start, map[string]any{oidcStateSessKey: "test-state", oidcNextSessKey: "/account/security"}); err != nil {
		t.Fatal(err)
	}
	var session fasthttp.Cookie
	session.SetKey("libredesk_session")
	start.RequestCtx.Response.Header.Cookie(&session)
	callback := twoFactorRequestForTest(app, nil)
	callback.RequestCtx.Request.Header.SetMethod("GET")
	callback.RequestCtx.Request.SetRequestURI("http://localhost/api/v1/oidc/1/finish?code=test&state=test-state")
	callback.RequestCtx.SetUserValue("id", "1")
	callback.RequestCtx.Request.Header.SetCookie("libredesk_session", string(session.Value()))
	if err := handleOIDCCallback(callback); err != nil {
		t.Fatal(err)
	}
	var challenge fasthttp.Cookie
	challenge.SetKey("libredesk_two_factor")
	if !callback.RequestCtx.Response.Header.Cookie(&challenge) {
		t.Fatalf("SSO did not challenge: %s", callback.RequestCtx.Response.Header.String())
	}
	callback.RequestCtx.Request.Header.SetCookie("libredesk_two_factor", string(challenge.Value()))
	pending, err := app.auth.PendingLogin(callback)
	if err != nil || pending.UserID != id || pending.Next != "/account/security" {
		t.Fatalf("SSO challenge incorrect: %+v %v", pending, err)
	}
	sessionUser, _ := app.auth.ValidateSession(callback)
	if sessionUser.ID != 0 {
		t.Fatal("SSO created full session before verification")
	}
}

func newTwoFactorTestApp(t *testing.T, name string) (*App, *sqlx.DB, int) {
	t.Helper()
	db := testutil.NewDB(t, name)
	translations := testutil.NewI18n(t)
	lo := logf.New(logf.Opts{})
	users, err := user.New(translations, user.Opts{DB: db, Lo: &lo})
	if err != nil {
		t.Fatal(err)
	}
	mfa, err := twofactor.New(db, "01234567890123456789012345678901", translations, &lo)
	if err != nil {
		t.Fatal(err)
	}
	server := miniredis.RunT(t)
	rd := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { rd.Close() })
	a, err := auth_.New(auth_.Config{}, translations, rd, &lo, nil)
	if err != nil {
		t.Fatal(err)
	}
	logs, err := activitylog.New(activitylog.Opts{DB: db, Lo: &lo, I18n: translations})
	if err != nil {
		t.Fatal(err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("TestPassword123!"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	var id int
	if err := db.Get(&id, `INSERT INTO users(type,email,first_name,last_name,password,enabled) VALUES('agent','agent@example.com','Test','Agent',$1,true) RETURNING id`, string(hash)); err != nil {
		t.Fatal(err)
	}
	return &App{user: users, twoFactor: mfa, auth: a, lo: &lo, i18n: translations, activityLog: logs}, db, id
}

func twoFactorRequestForTest(app *App, data any) *fastglue.Request {
	ctx := &fasthttp.RequestCtx{}
	ctx.Init(&fasthttp.Request{}, nil, nil)
	ctx.Request.Header.SetMethod("POST")
	ctx.Request.Header.SetContentType("application/json")
	raw, _ := json.Marshal(data)
	ctx.Request.SetBody(raw)
	return &fastglue.Request{RequestCtx: ctx, Context: app}
}
