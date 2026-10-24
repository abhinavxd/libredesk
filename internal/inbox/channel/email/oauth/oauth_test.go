package oauth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestClientCredentialsToken(t *testing.T) {
	var tenantPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NoError(t, r.ParseForm())
		tenantPath = r.URL.Path
		assert.Equal(t, "client_credentials", r.PostForm.Get("grant_type"))
		assert.Equal(t, MicrosoftAppOnlyScope, r.PostForm.Get("scope"))
		assert.Equal(t, "app-id", r.PostForm.Get("client_id"))
		assert.Equal(t, "s3cret", r.PostForm.Get("client_secret"))
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"token_type":"Bearer","expires_in":3599,"access_token":"app-token"}`))
	}))
	t.Cleanup(server.Close)
	original := microsoftTokenURL
	microsoftTokenURL = func(tenantID string) string { return server.URL + "/" + tenantID + "/oauth2/v2.0/token" }
	t.Cleanup(func() { microsoftTokenURL = original })

	token, err := ClientCredentialsToken(context.Background(), "app-id", "s3cret", "contoso.onmicrosoft.com")
	require.NoError(t, err)
	require.Equal(t, "/contoso.onmicrosoft.com/oauth2/v2.0/token", tenantPath)
	require.Equal(t, "app-token", token.AccessToken)
	require.Empty(t, token.RefreshToken)
	require.WithinDuration(t, time.Now().Add(time.Hour), token.Expiry, 5*time.Second)
}

func TestClientCredentialsTokenError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid_client","error_description":"AADSTS7000215: reflected-secret","access_token":"reflected-token"}`))
	}))
	t.Cleanup(server.Close)
	original := microsoftTokenURL
	microsoftTokenURL = func(string) string { return server.URL }
	t.Cleanup(func() { microsoftTokenURL = original })

	_, err := ClientCredentialsToken(context.Background(), "app-id", "wrong", "tenant")
	require.ErrorContains(t, err, "AADSTS7000215")
	require.NotContains(t, err.Error(), "wrong")
	require.NotContains(t, err.Error(), "reflected-")
}

func TestSafeTokenError(t *testing.T) {
	for _, tt := range []struct {
		err  error
		want string
	}{
		{context.DeadlineExceeded, "timed out contacting Microsoft Entra"},
		{context.Canceled, "Microsoft Entra token request canceled"},
		{&oauth2.RetrieveError{ErrorCode: "invalid_client", ErrorDescription: "reflected-secret"}, "invalid_client:"},
		{&oauth2.RetrieveError{ErrorCode: "invalid_scope"}, "invalid_scope:"},
		{&oauth2.RetrieveError{ErrorCode: "unauthorized_client"}, "unauthorized_client:"},
		{&oauth2.RetrieveError{ErrorCode: "server_error"}, "temporarily unavailable"},
		{fmt.Errorf("wrapped: %w", &oauth2.RetrieveError{ErrorDescription: "AADSTS90002: reflected-secret"}), "AADSTS90002: Tenant not found."},
		{&oauth2.RetrieveError{ErrorDescription: "AADSTS7000222: reflected-secret"}, "AADSTS7000222: Client secret expired."},
		{&oauth2.RetrieveError{ErrorDescription: "AADSTS700016: reflected-secret"}, "AADSTS700016: Application not found."},
		{&oauth2.RetrieveError{ErrorCode: "reflected-secret", Body: []byte(`{"access_token":"reflected-token"}`)}, "token request failed"},
		{errors.New("reflected-secret%2B"), "token request failed"},
	} {
		safe := SafeTokenError(tt.err)
		require.ErrorContains(t, safe, tt.want)
		require.NotContains(t, safe.Error(), "reflected-")
		require.Nil(t, errors.Unwrap(safe), "raw provider errors must not remain reachable")
		require.Equal(t, safe, SafeTokenError(safe), "sanitizing at another boundary preserves safe hints")
	}
	require.NoError(t, SafeTokenError(nil))
}

func TestMicrosoftTokenURL(t *testing.T) {
	require.Equal(t, "https://login.microsoftonline.com/contoso.onmicrosoft.com/oauth2/v2.0/token", microsoftTokenURL("contoso.onmicrosoft.com"))
}
