package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/zerodha/logf"
	"golang.org/x/oauth2"
)

type testOIDCKeys struct {
	payload []byte
}

func (k testOIDCKeys) VerifySignature(context.Context, string) ([]byte, error) {
	return k.payload, nil
}

func TestOIDCRequiresVerifiedEmail(t *testing.T) {
	for _, tt := range []struct {
		name     string
		email    string
		verified any
		want     bool
	}{
		{"verified", "agent@example.com", true, true},
		{"unverified", "agent@example.com", false, false},
		{"missing verification", "agent@example.com", nil, false},
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
