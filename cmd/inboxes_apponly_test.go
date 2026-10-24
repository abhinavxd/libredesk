package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/stretchr/testify/require"
	"github.com/zerodha/logf"
	"golang.org/x/oauth2"
)

type appOnlyTokenCall struct{ clientID, clientSecret, tenantID string }

func stubAppOnlyTokenRequest(t *testing.T, err error) *[]appOnlyTokenCall {
	t.Helper()
	var calls []appOnlyTokenCall
	original := requestAppOnlyToken
	requestAppOnlyToken = func(_ context.Context, clientID, clientSecret, tenantID string) (*oauth2.Token, error) {
		calls = append(calls, appOnlyTokenCall{clientID, clientSecret, tenantID})
		if err != nil {
			return nil, err
		}
		return &oauth2.Token{AccessToken: "token"}, nil
	}
	t.Cleanup(func() { requestAppOnlyToken = original })
	return &calls
}

func appOnlyConfigJSON(t *testing.T, authType, tenantID, clientID, secret string) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(imodels.Config{
		AuthType: authType,
		OAuth:    &imodels.OAuthConfig{Provider: "microsoft", TenantID: tenantID, ClientID: clientID, ClientSecret: secret},
	})
	require.NoError(t, err)
	return b
}

func TestCheckAppOnlyCredentials(t *testing.T) {
	app := newValidatorTestApp(t)
	stored := appOnlyConfigJSON(t, imodels.AuthTypeOAuth2App, "tenant", "client", "stored-secret")

	t.Run("create checks the credentials", func(t *testing.T) {
		calls := stubAppOnlyTokenRequest(t, nil)
		require.NoError(t, checkAppOnlyCredentials(app, appOnlyConfigJSON(t, imodels.AuthTypeOAuth2App, "tenant", "client", "secret"), nil))
		require.Equal(t, []appOnlyTokenCall{{"client", "secret", "tenant"}}, *calls)
	})
	t.Run("rejected credentials fail the save", func(t *testing.T) {
		stubAppOnlyTokenRequest(t, &oauth2.RetrieveError{ErrorCode: "invalid_client", ErrorDescription: "AADSTS7000215: Invalid client secret provided.\r\nTrace ID: abc"})
		err := checkAppOnlyCredentials(app, appOnlyConfigJSON(t, imodels.AuthTypeOAuth2App, "tenant", "client", "wrong"), nil)
		require.ErrorContains(t, err, "AADSTS7000215: Invalid client secret.")
		require.NotContains(t, err.Error(), "Trace ID")
		require.NotContains(t, err.Error(), "wrong")
	})
	t.Run("unchanged update skips the check", func(t *testing.T) {
		calls := stubAppOnlyTokenRequest(t, errors.New("must not be called"))
		require.NoError(t, checkAppOnlyCredentials(app, appOnlyConfigJSON(t, imodels.AuthTypeOAuth2App, "tenant", "client", ""), stored))
		require.Empty(t, *calls)
	})
	t.Run("changed client uses the stored secret", func(t *testing.T) {
		calls := stubAppOnlyTokenRequest(t, nil)
		require.NoError(t, checkAppOnlyCredentials(app, appOnlyConfigJSON(t, imodels.AuthTypeOAuth2App, "tenant", "new-client", ""), stored))
		require.Equal(t, []appOnlyTokenCall{{"new-client", "stored-secret", "tenant"}}, *calls)
	})
	t.Run("switch from delegated is checked", func(t *testing.T) {
		calls := stubAppOnlyTokenRequest(t, nil)
		delegated := appOnlyConfigJSON(t, imodels.AuthTypeOAuth2, "tenant", "client", "stored-secret")
		require.NoError(t, checkAppOnlyCredentials(app, appOnlyConfigJSON(t, imodels.AuthTypeOAuth2App, "tenant", "client", ""), delegated))
		require.Len(t, *calls, 1)
	})
	t.Run("other auth types are not checked", func(t *testing.T) {
		calls := stubAppOnlyTokenRequest(t, errors.New("must not be called"))
		require.NoError(t, checkAppOnlyCredentials(app, appOnlyConfigJSON(t, imodels.AuthTypeOAuth2, "tenant", "client", "secret"), nil))
		require.NoError(t, checkAppOnlyCredentials(app, json.RawMessage(`{"auth_type":"password"}`), nil))
		require.Empty(t, *calls)
	})
}

func TestAppOnlyCredentialErrorsAreSafe(t *testing.T) {
	for _, providerErr := range []error{
		&oauth2.RetrieveError{ErrorCode: "invalid_client", ErrorDescription: "AADSTS7000215: reflected-secret", Body: []byte(`{"access_token":"reflected-token"}`)},
		&oauth2.RetrieveError{ErrorCode: "reflected-secret", ErrorDescription: "reflected-token"},
		&oauth2.RetrieveError{Body: []byte(`{"access_token":"reflected-token"}`)},
		fmt.Errorf("wrapped: %w", &oauth2.RetrieveError{ErrorDescription: "AADSTS90002: reflected-secret"}),
		errors.New("reflected-secret%2B and reflected-token"),
	} {
		app := newValidatorTestApp(t)
		var logs bytes.Buffer
		lo := logf.New(logf.Opts{Writer: &logs})
		app.lo = &lo
		stubAppOnlyTokenRequest(t, providerErr)
		err := checkAppOnlyCredentials(app, appOnlyConfigJSON(t, imodels.AuthTypeOAuth2App, "tenant", "client", "reflected-secret"), nil)
		require.Error(t, err)
		require.NotEmpty(t, logs.String())
		require.NotContains(t, err.Error()+logs.String(), "reflected-")
	}
}

func TestIsAppOnlyInbox(t *testing.T) {
	email := func(cfg string) imodels.Inbox { return imodels.Inbox{Channel: "email", Config: json.RawMessage(cfg)} }
	require.True(t, isAppOnlyInbox(email(`{"auth_type":"oauth2_app"}`)))
	require.False(t, isAppOnlyInbox(email(`{"auth_type":"oauth2"}`)))
	require.False(t, isAppOnlyInbox(email(`not json`)))
	require.False(t, isAppOnlyInbox(imodels.Inbox{Channel: "livechat", Config: json.RawMessage(`{"auth_type":"oauth2_app"}`)}))
}
