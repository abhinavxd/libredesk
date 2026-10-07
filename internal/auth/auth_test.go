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

func (k testOIDCKeys) VerifySignature(context.Context, string) ([]byte, error) {
	return k.payload, nil
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

type sessionUsers struct {
	versions map[int]int
	err      error
}

func (u *sessionUsers) GetSessionVersion(id int) (int, error) {
	return u.versions[id], u.err
}

func TestSessionsRejectRevokedAndLegacyCookies(t *testing.T) {
	rd := redis.NewClient(&redis.Options{Addr: miniredis.RunT(t).Addr()})
	t.Cleanup(func() { rd.Close() })
	users := &sessionUsers{versions: map[int]int{1: 1, 2: 1}}
	lo := logf.New(logf.Opts{})
	a, err := New(Config{}, testutil.NewI18n(t), rd, &lo, nil /** dialControl **/, users)
	if err != nil {
		t.Fatal(err)
	}

	request := func(cookie string) *fastglue.Request {
		r := &fastglue.Request{RequestCtx: &fasthttp.RequestCtx{}}
		r.RequestCtx.Request.SetRequestURI("https://app.example.com/")
		if cookie != "" {
			r.RequestCtx.Request.Header.SetCookie("libredesk_session", cookie)
		}
		return r
	}
	login := func(id, version int) string {
		t.Helper()
		r := request("")
		if err := a.SaveSession(amodels.User{ID: id, SessionVersion: version}, r); err != nil {
			t.Fatal(err)
		}
		var cookie fasthttp.Cookie
		cookie.SetKey("libredesk_session")
		if !r.RequestCtx.Response.Header.Cookie(&cookie) {
			t.Fatal("missing session cookie")
		}
		return string(cookie.Value())
	}
	validate := func(cookie string, wantValid bool) {
		t.Helper()
		user, err := a.ValidateSession(request(cookie))
		if wantValid {
			if err != nil || user.ID <= 0 {
				t.Fatalf("valid session rejected: %v", err)
			}
		} else if !errors.Is(err, simplesessions.ErrInvalidSession) {
			t.Fatalf("revoked session accepted: user=%d err=%v", user.ID, err)
		}
	}

	first, second, other := login(1, 1), login(1, 1), login(2, 1)
	validate(first, true)
	validate(second, true)
	users.versions[1]++
	validate(first, false)
	validate(second, false)
	validate(other, true)
	validate(login(1, 2), true)
	validate(login(1, 0), false)
	delete(users.versions, 2)
	validate(other, false)
	users.err = errors.New("database unavailable")
	if user, err := a.ValidateSession(request(other)); err == nil || errors.Is(err, simplesessions.ErrInvalidSession) || user.ID != 0 {
		t.Fatalf("lookup failure must reject without revoking: user=%d err=%v", user.ID, err)
	}
}
