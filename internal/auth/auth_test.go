package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/alicebob/miniredis/v2"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
	"github.com/zerodha/simplesessions/v3"
	"golang.org/x/oauth2"
)

type testOIDCKeys struct {
	payload []byte
}

type sessionRedisHook struct {
	process  func(redis.Cmder) error
	pipeline func([]redis.Cmder) error
}

func (k testOIDCKeys) VerifySignature(context.Context, string) ([]byte, error) {
	return k.payload, nil
}

func (h sessionRedisHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h sessionRedisHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if h.process != nil {
			if err := h.process(cmd); err != nil {
				return err
			}
		}
		return next(ctx, cmd)
	}
}

func (h sessionRedisHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		if h.pipeline != nil {
			if err := h.pipeline(cmds); err != nil {
				return err
			}
		}
		return next(ctx, cmds)
	}
}

func TestOIDCRejectsBlankEmail(t *testing.T) {
	for _, tt := range []struct {
		name     string
		email    string
		verified any
		want     bool
	}{
		{"verified", "agent@example.com", true, true},
		{"unverified", "agent@example.com", false, true},
		{"missing verification", "agent@example.com", nil, true},
		{"empty email", "", true, false},
		{"blank email", " \t", true, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			claims := map[string]any{"iss": "https://id.example.com", "aud": "desk", "sub": "test-user", "exp": time.Now().Add(time.Hour).Unix(), "email": tt.email}
			if tt.verified != nil {
				claims["email_verified"] = tt.verified
			}
			payload, err := json.Marshal(claims)
			if err != nil {
				t.Fatal(err)
			}
			token := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256"}`)) + "." + base64.RawURLEncoding.EncodeToString(payload) + ".c2ln"
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]any{"access_token": "test-token", "token_type": "Bearer", "id_token": token})
			}))
			defer server.Close()
			lo := logf.New(logf.Opts{})
			a := &Auth{
				logger:       &lo,
				oidcClient:   server.Client(),
				oauthCfgs:    map[int]oauth2.Config{1: {ClientID: "desk", Endpoint: oauth2.Endpoint{TokenURL: server.URL}}},
				verifiers:    map[int]*oidc.IDTokenVerifier{1: oidc.NewVerifier("https://id.example.com", testOIDCKeys{payload}, &oidc.Config{ClientID: "desk"})},
				redirectURLs: map[int]func() (string, error){1: func() (string, error) { return "https://app.example.com/finish", nil }},
			}
			_, _, err = a.ExchangeOIDCToken(t.Context(), 1, "test-code")
			if (err == nil) != tt.want {
				t.Fatalf("exchange err = %v, want success %v", err, tt.want)
			}
		})
	}
}

func TestRemovedProviderStaysDisabledWhenReloadFails(t *testing.T) {
	lo := logf.New(logf.Opts{})
	a := &Auth{
		logger:       &lo,
		i18n:         testutil.NewI18n(t),
		oidcClient:   http.DefaultClient,
		oauthCfgs:    map[int]oauth2.Config{1: {ClientID: "desk", Endpoint: oauth2.Endpoint{AuthURL: "https://id.example.com/auth"}}},
		verifiers:    map[int]*oidc.IDTokenVerifier{1: nil},
		redirectURLs: map[int]func() (string, error){1: func() (string, error) { return "https://app.example.com/finish", nil }},
	}
	if _, err := a.LoginURL(1, "state"); err != nil {
		t.Fatal(err)
	}
	a.RemoveProvider(1)
	if err := a.Reload(Config{Providers: []Provider{{ID: 2, ProviderURL: ":invalid"}}}); err == nil {
		t.Fatal("expected failed discovery")
	}
	if _, err := a.LoginURL(1, "state"); err == nil {
		t.Fatal("removed provider still accepts logins")
	}
	if _, _, err := a.ExchangeOIDCToken(t.Context(), 1, "code"); err == nil {
		t.Fatal("removed provider still accepts callbacks")
	}
}

func TestSessionsRejectRevokedAndLegacyCookies(t *testing.T) {
	a, rd, _ := sessionAuthForTest(t, time.Hour)
	first := saveTestSession(t, a, 1)
	second := saveTestSession(t, a, 1)
	other := saveTestSession(t, a, 2)
	requireSession(t, a, first, 1)
	requireSession(t, a, second, 1)
	if err := rd.SAdd(t.Context(), userSessionKeyPrefix+"1", first).Err(); err != nil {
		t.Fatal(err)
	}
	if err := a.DestroyUserSessions(t.Context(), 1); err != nil {
		t.Fatal(err)
	}
	requireSession(t, a, first, 0 /** wantID **/)
	requireSession(t, a, second, 0 /** wantID **/)
	requireSession(t, a, other, 2)
	if n, err := rd.Exists(t.Context(), userSessionKeyPrefix+"1", sessionKeyPrefix+first, sessionKeyPrefix+second).Result(); err != nil || n != 0 {
		t.Fatalf("revoked sessions remain: count=%d err=%v", n, err)
	}
	if err := a.DestroyUserSessions(t.Context(), 1); err != nil {
		t.Fatalf("repeated revocation: %v", err)
	}
	requireSession(t, a, saveTestSession(t, a, 1), 1)

	r := sessionRequest("" /** cookie **/)
	sess, err := a.sess.NewSession(r, r)
	if err != nil {
		t.Fatal(err)
	}
	if err := sess.SetMulti(map[string]any{"id": 1}); err != nil {
		t.Fatal(err)
	}
	requireSession(t, a, sess.ID(), 0 /** wantID **/)
}

func TestSessionLogoutPreservesOtherLogins(t *testing.T) {
	a, rd, _ := sessionAuthForTest(t, time.Hour)
	first := saveTestSession(t, a, 1)
	second := saveTestSession(t, a, 1)
	if err := a.DestroySession(sessionRequest(first)); err != nil {
		t.Fatal(err)
	}
	requireSession(t, a, first, 0 /** wantID **/)
	requireSession(t, a, second, 1)
	ids, err := rd.SMembers(t.Context(), userSessionKeyPrefix+"1").Result()
	if err != nil || len(ids) != 1 || ids[0] != second {
		t.Fatalf("remaining sessions = %v, err = %v", ids, err)
	}
	if err := a.DestroySession(sessionRequest(first)); err != nil {
		t.Fatalf("repeated logout: %v", err)
	}
}

func TestSessionIndexExpiresAfterLastLogin(t *testing.T) {
	a, rd, mr := sessionAuthForTest(t, time.Hour)
	first := saveTestSession(t, a, 1)
	mr.FastForward(30 * time.Minute)
	second := saveTestSession(t, a, 1)
	if ttl := mr.TTL(userSessionKeyPrefix + "1"); ttl != time.Hour {
		t.Fatalf("index TTL = %v, want %v", ttl, time.Hour)
	}
	mr.FastForward(31 * time.Minute)
	requireSession(t, a, first, 0 /** wantID **/)
	requireSession(t, a, second, 1)
	if err := a.DestroyUserSessions(t.Context(), 1); err != nil {
		t.Fatal(err)
	}
	requireSession(t, a, second, 0 /** wantID **/)

	last := saveTestSession(t, a, 1)
	mr.FastForward(time.Hour)
	requireSession(t, a, last, 0 /** wantID **/)
	if n, err := rd.Exists(t.Context(), userSessionKeyPrefix+"1").Result(); err != nil || n != 0 {
		t.Fatalf("expired index remains: count=%d err=%v", n, err)
	}
}

func TestSessionIndexPreservesLongerSessionLifetime(t *testing.T) {
	a, rd, mr := sessionAuthForTest(t, time.Hour)
	first := saveTestSession(t, a, 1)
	mr.FastForward(10 * time.Minute)
	shorter, err := New(Config{SessionLifetime: time.Minute}, a.i18n, rd, a.logger, nil /** dialControl **/)
	if err != nil {
		t.Fatal(err)
	}
	second := saveTestSession(t, shorter, 1)
	if ttl := mr.TTL(userSessionKeyPrefix + "1"); ttl != 50*time.Minute {
		t.Fatalf("index expired before older sessions: TTL=%v", ttl)
	}
	if err := shorter.DestroyUserSessions(t.Context(), 1); err != nil {
		t.Fatal(err)
	}
	requireSession(t, shorter, first, 0 /** wantID **/)
	requireSession(t, shorter, second, 0 /** wantID **/)
}

func TestSessionIndexFailureDoesNotAuthenticate(t *testing.T) {
	a, rd, _ := sessionAuthForTest(t, time.Hour)
	if err := rd.Set(t.Context(), userSessionKeyPrefix+"1", "wrong type", 0 /** expiration **/).Err(); err != nil {
		t.Fatal(err)
	}
	r := sessionRequest("" /** cookie **/)
	if err := a.SaveSession(amodels.User{ID: 1}, r); err == nil {
		t.Fatal("login succeeded without indexing the session")
	}
	var cookie fasthttp.Cookie
	cookie.SetKey("libredesk_session")
	if !r.RequestCtx.Response.Header.Cookie(&cookie) {
		t.Fatal("missing session cookie")
	}
	user, err := a.ValidateSession(sessionRequest(string(cookie.Value())))
	if err != nil || user.ID != 0 {
		t.Fatalf("failed login acquired a user: id=%d err=%v", user.ID, err)
	}
}

func TestSessionRevocationFailurePreservesIndex(t *testing.T) {
	a, rd, _ := sessionAuthForTest(t, time.Hour)
	cookie := saveTestSession(t, a, 1)
	rd.AddHook(sessionRedisHook{process: func(cmd redis.Cmder) error {
		if cmd.Name() == "del" {
			return errors.New("redis unavailable")
		}
		return nil
	}})
	if err := a.DestroyUserSessions(t.Context(), 1); err == nil {
		t.Fatal("revocation failure reported success")
	}
	requireSession(t, a, cookie, 1)
	if n, err := rd.SCard(t.Context(), userSessionKeyPrefix+"1").Result(); err != nil || n != 1 {
		t.Fatalf("failed revocation lost its index: count=%d err=%v", n, err)
	}
}

func TestSessionRevocationWaitsForRegistration(t *testing.T) {
	a, rd, _ := sessionAuthForTest(t, time.Hour)
	entered := make(chan struct{})
	release := make(chan struct{})
	defer close(release)
	rd.AddHook(sessionRedisHook{pipeline: func(cmds []redis.Cmder) error {
		for _, cmd := range cmds {
			if cmd.Name() == "hmset" {
				close(entered)
				<-release
			}
		}
		return nil
	}})
	r := sessionRequest("" /** cookie **/)
	loginDone := make(chan error, 1)
	go func() { loginDone <- a.SaveSession(amodels.User{ID: 1}, r) }()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("login did not reach session registration")
	}
	revokeDone := make(chan error, 1)
	go func() { revokeDone <- a.DestroyUserSessions(t.Context(), 1) }()
	release <- struct{}{}
	if err := <-loginDone; err != nil {
		t.Fatal(err)
	}
	if err := <-revokeDone; err != nil {
		t.Fatal(err)
	}
	var cookie fasthttp.Cookie
	cookie.SetKey("libredesk_session")
	if !r.RequestCtx.Response.Header.Cookie(&cookie) {
		t.Fatal("missing session cookie")
	}
	requireSession(t, a, string(cookie.Value()), 0 /** wantID **/)
}

func TestSessionValidationUsesOneRedisRead(t *testing.T) {
	a, _, mr := sessionAuthForTest(t, 0 /** lifetime **/)
	cookie := saveTestSession(t, a, 1)
	before := mr.CommandCount()
	requireSession(t, a, cookie, 1)
	if count := mr.CommandCount() - before; count != 1 {
		t.Fatalf("validation used %d Redis commands, want 1", count)
	}
	if ttl := mr.TTL(userSessionKeyPrefix + "1"); ttl != defaultSessionLifetime {
		t.Fatalf("default index TTL = %v", ttl)
	}
}

func sessionAuthForTest(t *testing.T, lifetime time.Duration) (*Auth, *redis.Client, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rd := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: -1})
	t.Cleanup(func() { rd.Close() })
	lo := logf.New(logf.Opts{})
	a, err := New(Config{SessionLifetime: lifetime}, testutil.NewI18n(t), rd, &lo, nil /** dialControl **/)
	if err != nil {
		t.Fatal(err)
	}
	return a, rd, mr
}

func sessionRequest(cookie string) *fastglue.Request {
	r := &fastglue.Request{RequestCtx: &fasthttp.RequestCtx{}}
	var req fasthttp.Request
	r.RequestCtx.Init(&req, nil /** remoteAddr **/, nil /** logger **/)
	r.RequestCtx.Request.SetRequestURI("https://app.example.com/")
	if cookie != "" {
		r.RequestCtx.Request.Header.SetCookie("libredesk_session", cookie)
	}
	return r
}

func saveTestSession(t *testing.T, a *Auth, id int) string {
	t.Helper()
	r := sessionRequest("" /** cookie **/)
	if err := a.SaveSession(amodels.User{ID: id}, r); err != nil {
		t.Fatal(err)
	}
	var cookie fasthttp.Cookie
	cookie.SetKey("libredesk_session")
	if !r.RequestCtx.Response.Header.Cookie(&cookie) {
		t.Fatal("missing session cookie")
	}
	return string(cookie.Value())
}

func requireSession(t *testing.T, a *Auth, cookie string, wantID int) {
	t.Helper()
	user, err := a.ValidateSession(sessionRequest(cookie))
	if wantID > 0 {
		if err != nil || user.ID != wantID {
			t.Fatalf("session user=%d err=%v, want user=%d", user.ID, err, wantID)
		}
	} else if !errors.Is(err, simplesessions.ErrInvalidSession) {
		t.Fatalf("revoked session accepted: user=%d err=%v", user.ID, err)
	}
}
