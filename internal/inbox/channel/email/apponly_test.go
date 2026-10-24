package email

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/stretchr/testify/require"
	"github.com/zerodha/logf"
	"golang.org/x/oauth2"
)

// stubAppOnlyTokens replaces the client credentials request for one test.
type stubAppOnlyTokens struct {
	mu     sync.Mutex
	calls  int
	token  string
	expiry time.Duration
	err    error
}

func (s *stubAppOnlyTokens) install(t *testing.T) {
	t.Helper()
	original := clientCredentialsToken
	clientCredentialsToken = func(_ context.Context, clientID, clientSecret, tenantID string) (*oauth2.Token, error) {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.calls++
		if clientID != "app-id" || clientSecret != "s3cret" || tenantID != "contoso.onmicrosoft.com" {
			return nil, errors.New("unexpected credentials")
		}
		if s.err != nil {
			return nil, s.err
		}
		return &oauth2.Token{AccessToken: s.token, Expiry: time.Now().Add(s.expiry)}, nil
	}
	t.Cleanup(func() { clientCredentialsToken = original })
}

func (s *stubAppOnlyTokens) set(token string, expiry time.Duration, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.token, s.expiry, s.err = token, expiry, err
}

func (s *stubAppOnlyTokens) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

type authStatusRecorder struct {
	mu       sync.Mutex
	statuses []bool
}

func (r *authStatusRecorder) record(_ int, ok bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.statuses = append(r.statuses, ok)
}

func (r *authStatusRecorder) last() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.statuses[len(r.statuses)-1]
}

func newAppOnlyEmail(t *testing.T, smtpCfg []imodels.SMTPConfig, status *authStatusRecorder) *Email {
	t.Helper()
	logger := logf.New(logf.Opts{Writer: io.Discard})
	e, err := New(nil, nil, Opts{
		ID: 7,
		Lo: &logger,
		Config: imodels.Config{
			AuthType: imodels.AuthTypeOAuth2App,
			From:     "shared@contoso.com",
			OAuth: &imodels.OAuthConfig{
				Provider: "microsoft", TenantID: "contoso.onmicrosoft.com", ClientID: "app-id", ClientSecret: "s3cret",
				AccessToken: "another-mailbox-token", RefreshToken: "stale-delegated-refresh-token", ExpiresAt: time.Now().Add(time.Hour),
			},
			SMTP: smtpCfg,
		},
		TokenRefreshCallback: func(int, imodels.Config) error {
			t.Error("app-only tokens must not be persisted")
			return nil
		},
		AuthStatusCallback: status.record,
	})
	require.NoError(t, err)
	t.Cleanup(func() { e.Close() })
	return e
}

func TestAppOnlyTokenLifecycle(t *testing.T) {
	tokens := &stubAppOnlyTokens{}
	tokens.install(t)
	tokens.set("first", time.Hour, nil)
	status := &authStatusRecorder{}
	smtpCfg := []imodels.SMTPConfig{{Host: "smtp.office365.com", Port: 587, TLSType: "starttls", Username: "shared@contoso.com", MaxConns: 1}}

	e := newAppOnlyEmail(t, smtpCfg, status)
	require.Equal(t, 1, tokens.count(), "token fetched at start")
	require.Len(t, e.smtpPools, 1)
	require.Equal(t, "first", e.smtpPoolsToken)
	require.True(t, status.last())

	cfg, refreshed, err := e.refreshOAuthIfNeeded()
	require.NoError(t, err)
	require.False(t, refreshed)
	require.Equal(t, "first", cfg.AccessToken)
	require.Equal(t, 1, tokens.count(), "valid token reused")

	e.oauthMu.Lock()
	e.oauth.ExpiresAt = time.Now().Add(2 * time.Minute)
	e.oauthMu.Unlock()
	tokens.set("second", time.Hour, nil)
	cfg, refreshed, err = e.refreshOAuthIfNeeded()
	require.NoError(t, err)
	require.True(t, refreshed)
	require.Equal(t, "second", cfg.AccessToken)
	require.Empty(t, cfg.RefreshToken)
	require.Equal(t, 2, tokens.count(), "token renewed inside the expiry window")

	e.oauthMu.Lock()
	e.oauth.ExpiresAt = time.Now()
	e.oauthMu.Unlock()
	tokens.set("", 0, &oauth2.RetrieveError{ErrorCode: "invalid_client", ErrorDescription: "AADSTS7000215: Invalid client secret provided."})
	_, _, err = e.refreshOAuthIfNeeded()
	require.ErrorContains(t, err, "AADSTS7000215")
	require.False(t, status.last())

	tokens.set("third", time.Hour, nil)
	cfg, _, err = e.refreshOAuthIfNeeded()
	require.NoError(t, err)
	require.Equal(t, "third", cfg.AccessToken)
	require.True(t, status.last())
}

func TestAppOnlyStartsWithoutToken(t *testing.T) {
	tokens := &stubAppOnlyTokens{}
	tokens.install(t)
	tokens.set("", 0, errors.New("network unreachable"))
	status := &authStatusRecorder{}
	smtpCfg := []imodels.SMTPConfig{{Host: "smtp.office365.com", Port: 587, TLSType: "starttls", Username: "shared@contoso.com", MaxConns: 1}}

	e := newAppOnlyEmail(t, smtpCfg, status)
	require.Equal(t, 1, tokens.count())
	require.Empty(t, e.smtpPools, "no pool without a token")
	require.False(t, status.last())
}

func TestAppOnlyRuntimeErrorsAreSafe(t *testing.T) {
	tokens := &stubAppOnlyTokens{}
	tokens.install(t)
	tokens.set("first", time.Hour, nil)
	e := newAppOnlyEmail(t, nil, &authStatusRecorder{})
	var logs bytes.Buffer
	logger := logf.New(logf.Opts{Writer: &logs})
	e.lo = &logger
	tokens.set("", 0, &oauth2.RetrieveError{
		ErrorCode: "invalid_client", ErrorDescription: "AADSTS7000215: reflected-secret",
		Body: []byte(`{"access_token":"reflected-token"}`),
	})
	e.oauthMu.Lock()
	e.oauth.ExpiresAt = time.Time{}
	e.oauthMu.Unlock()
	_, _, err := e.refreshOAuthIfNeeded()
	require.ErrorContains(t, err, "AADSTS7000215")
	require.NotEmpty(t, logs.String())
	require.NotContains(t, err.Error()+logs.String(), "reflected-")
}

func TestRefreshOAuthConfigRejectsAppOnly(t *testing.T) {
	_, err := RefreshOAuthConfig(imodels.Config{
		AuthType: imodels.AuthTypeOAuth2App,
		OAuth:    &imodels.OAuthConfig{Provider: "microsoft", RefreshToken: "stale-delegated-token"},
	})
	require.ErrorContains(t, err, "delegated OAuth config required")
	_, err = RefreshOAuthConfig(imodels.Config{AuthType: imodels.AuthTypeOAuth2})
	require.ErrorContains(t, err, "delegated OAuth config required")
}
