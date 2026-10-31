// Package oauth provides OAuth 2.0 authentication support for email channels.
package oauth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/microsoft"
)

// Provider represents an OAuth provider.
type Provider string

const (
	ProviderMicrosoft Provider = "microsoft"
	ProviderGoogle    Provider = "google"
)

// Scopes for each provider.
var (
	MicrosoftScopes = []string{
		"https://outlook.office.com/IMAP.AccessAsUser.All",
		"https://outlook.office.com/SMTP.Send",
		"offline_access",
		"openid",
		"email",
	}
	GoogleScopes = []string{
		"https://mail.google.com/",
		"https://www.googleapis.com/auth/userinfo.email",
	}
)

// MicrosoftAppOnlyScope is the client credentials scope for Exchange Online IMAP and SMTP AUTH.
const MicrosoftAppOnlyScope = "https://outlook.office365.com/.default"

// microsoftTokenURL returns the Microsoft Entra token endpoint for a tenant.
var microsoftTokenURL = func(tenantID string) string {
	return microsoft.AzureADEndpoint(tenantID).TokenURL
}

// ClientCredentialsToken requests an app-only access token for Exchange Online IMAP and SMTP AUTH.
// The grant issues no refresh token; request a new token before the current one expires.
func ClientCredentialsToken(ctx context.Context, clientID, clientSecret, tenantID string) (*oauth2.Token, error) {
	cfg := clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     microsoftTokenURL(tenantID),
		Scopes:       []string{MicrosoftAppOnlyScope},
		AuthStyle:    oauth2.AuthStyleInParams,
	}
	token, err := cfg.Token(ctx)
	return token, SafeTokenError(err)
}

type safeTokenError string

func (e safeTokenError) Error() string { return string(e) }

// SafeTokenError discards untrusted provider diagnostics, which can include credentials or response bodies.
func SafeTokenError(err error) error {
	if err == nil {
		return nil
	}
	var safe safeTokenError
	if errors.As(err, &safe) {
		return safe
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return safeTokenError("timed out contacting Microsoft Entra")
	}
	if errors.Is(err, context.Canceled) {
		return safeTokenError("Microsoft Entra token request canceled")
	}
	var response *oauth2.RetrieveError
	if errors.As(err, &response) {
		for code, hint := range map[string]string{
			"AADSTS7000215": "Invalid client secret. Enter the secret value, not its ID.",
			"AADSTS7000222": "Client secret expired. Create a new secret.",
			"AADSTS700016":  "Application not found. Check the client ID and tenant ID.",
			"AADSTS90002":   "Tenant not found. Check the tenant ID.",
		} {
			if strings.HasPrefix(response.ErrorDescription, code+":") {
				return safeTokenError(code + ": " + hint)
			}
		}
		switch response.ErrorCode {
		case "invalid_client":
			return safeTokenError("invalid_client: Check the client ID, secret and tenant ID.")
		case "unauthorized_client":
			return safeTokenError("unauthorized_client: Check application permissions and admin consent.")
		case "invalid_scope":
			return safeTokenError("invalid_scope: Check the Exchange Online application permissions.")
		case "temporarily_unavailable", "server_error":
			return safeTokenError("Microsoft Entra is temporarily unavailable. Try again.")
		}
	}
	return safeTokenError("Microsoft Entra token request failed. Check credentials and network connectivity.")
}

// GetOAuth2Config returns an oauth2.Config for the given provider.
func GetOAuth2Config(provider Provider, clientID, clientSecret, redirectURI string, tenantID ...string) (*oauth2.Config, error) {
	switch provider {
	case ProviderMicrosoft:
		tenant := "common"
		if len(tenantID) > 0 && tenantID[0] != "" {
			tenant = tenantID[0]
		}
		return &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURI,
			Scopes:       MicrosoftScopes,
			Endpoint:     microsoft.AzureADEndpoint(tenant),
		}, nil
	case ProviderGoogle:
		return &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURI,
			Scopes:       GoogleScopes,
			Endpoint:     google.Endpoint,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported OAuth provider: %s", provider)
	}
}

// ExchangeCodeForToken exchanges an authorization code for access and refresh tokens.
func ExchangeCodeForToken(ctx context.Context, provider Provider, clientID, clientSecret, code, redirectURI string, tenantID ...string) (*oauth2.Token, error) {
	cfg, err := GetOAuth2Config(provider, clientID, clientSecret, redirectURI, tenantID...)
	if err != nil {
		return nil, err
	}
	return cfg.Exchange(ctx, code)
}

// RefreshToken exchanges a refresh token for a new access token.
func RefreshToken(ctx context.Context, provider Provider, clientID, clientSecret, refreshToken string, tenantID ...string) (*oauth2.Token, error) {
	cfg, err := GetOAuth2Config(provider, clientID, clientSecret, "", tenantID...)
	if err != nil {
		return nil, err
	}

	oldToken := &oauth2.Token{
		RefreshToken: refreshToken,
	}

	src := cfg.TokenSource(ctx, oldToken)
	return src.Token()
}

// BuildAuthorizationURL builds the OAuth authorization URL.
func BuildAuthorizationURL(provider Provider, clientID, redirectURI, state string, tenantID ...string) (string, error) {
	cfg, err := GetOAuth2Config(provider, "", "", redirectURI, tenantID...)
	if err != nil {
		return "", err
	}
	cfg.ClientID = clientID

	// Google requires prompt=consent to issue a refresh token on re-authentication.
	if provider == ProviderGoogle {
		return cfg.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce), nil
	}
	return cfg.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "select_account")), nil
}

// IsTokenExpired checks if an access token has expired or is about to expire.
// Returns true if the token will expire in the next 5 minutes.
func IsTokenExpired(expiresAt time.Time) bool {
	return time.Now().Add(5 * time.Minute).After(expiresAt)
}
