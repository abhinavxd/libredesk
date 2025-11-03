// Package oauth provides OAuth 2.0 authentication support for email channels.
package oauth

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Provider represents an OAuth provider configuration.
type Provider string

const (
	ProviderMicrosoft Provider = "microsoft"
	ProviderGoogle    Provider = "google"
)

// TokenResponse represents the response from an OAuth token endpoint.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope,omitempty"`
}

// ProviderConfig holds OAuth provider-specific configuration.
type ProviderConfig struct {
	TokenEndpoint string
	AuthEndpoint  string
	Scopes        []string
}

// GetProviderConfig returns the configuration for a specific OAuth provider.
func GetProviderConfig(provider Provider) (*ProviderConfig, error) {
	switch provider {
	case ProviderMicrosoft:
		return &ProviderConfig{
			TokenEndpoint: "https://login.microsoftonline.com/common/oauth2/v2.0/token",
			AuthEndpoint:  "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
			Scopes: []string{
				"https://outlook.office.com/IMAP.AccessAsUser.All",
				"https://outlook.office.com/SMTP.Send",
				"offline_access",
			},
		}, nil
	case ProviderGoogle:
		return &ProviderConfig{
			TokenEndpoint: "https://oauth2.googleapis.com/token",
			AuthEndpoint:  "https://accounts.google.com/o/oauth2/v2/auth",
			Scopes: []string{
				"https://mail.google.com/",
				"https://www.googleapis.com/auth/userinfo.email",
			},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported OAuth provider: %s", provider)
	}
}

// BuildXOAuth2String creates the XOAUTH2 authentication string for IMAP/SMTP.
// Format: base64("user=" + username + "^Aauth=Bearer " + access_token + "^A^A")
func BuildXOAuth2String(username, accessToken string) string {
	authString := fmt.Sprintf("user=%s\x01auth=Bearer %s\x01\x01", username, accessToken)
	return base64.StdEncoding.EncodeToString([]byte(authString))
}

// RefreshToken exchanges a refresh token for a new access token.
func RefreshToken(provider Provider, clientID, clientSecret, refreshToken string, tenantID ...string) (*TokenResponse, error) {
	config, err := GetProviderConfig(provider)
	if err != nil {
		return nil, err
	}

	// Build token endpoint URL
	tokenEndpoint := config.TokenEndpoint
	if provider == ProviderMicrosoft && len(tenantID) > 0 && tenantID[0] != "" {
		// Use tenant-specific endpoint for Microsoft
		tokenEndpoint = fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantID[0])
	}

	// Prepare request body
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", clientID)

	// Client secret is required for confidential clients
	if clientSecret != "" {
		data.Set("client_secret", clientSecret)
	}

	// Make HTTP request
	req, err := http.NewRequest("POST", tokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("refreshing token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("parsing token response: %w", err)
	}

	return &tokenResp, nil
}

// ExchangeCodeForToken exchanges an authorization code for access and refresh tokens.
func ExchangeCodeForToken(provider Provider, clientID, clientSecret, code, redirectURI string, tenantID ...string) (*TokenResponse, error) {
	config, err := GetProviderConfig(provider)
	if err != nil {
		return nil, err
	}

	// Build token endpoint URL
	tokenEndpoint := config.TokenEndpoint
	if provider == ProviderMicrosoft && len(tenantID) > 0 && tenantID[0] != "" {
		// Use tenant-specific endpoint for Microsoft
		tokenEndpoint = fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantID[0])
	}

	// Prepare request body
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("client_id", clientID)

	// Client secret is required for confidential clients
	if clientSecret != "" {
		data.Set("client_secret", clientSecret)
	}

	// Make HTTP request
	req, err := http.NewRequest("POST", tokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("exchanging code for token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("parsing token response: %w", err)
	}

	return &tokenResp, nil
}

// BuildAuthorizationURL builds the OAuth authorization URL.
func BuildAuthorizationURL(provider Provider, clientID, redirectURI, state string, tenantID ...string) (string, error) {
	config, err := GetProviderConfig(provider)
	if err != nil {
		return "", err
	}

	// Build auth endpoint URL
	authEndpoint := config.AuthEndpoint
	if provider == ProviderMicrosoft && len(tenantID) > 0 && tenantID[0] != "" {
		// Use tenant-specific endpoint for Microsoft
		authEndpoint = fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/authorize", tenantID[0])
	}

	// Build URL with query parameters
	u, err := url.Parse(authEndpoint)
	if err != nil {
		return "", fmt.Errorf("parsing auth endpoint: %w", err)
	}

	q := u.Query()
	q.Set("client_id", clientID)
	q.Set("response_type", "code")
	q.Set("redirect_uri", redirectURI)
	q.Set("state", state)
	q.Set("scope", strings.Join(config.Scopes, " "))
	q.Set("access_type", "offline")
	q.Set("prompt", "consent")

	u.RawQuery = q.Encode()
	return u.String(), nil
}

// IsTokenExpired checks if an access token has expired or is about to expire.
// Returns true if the token will expire in the next 5 minutes.
func IsTokenExpired(expiresAt time.Time) bool {
	return time.Now().Add(5 * time.Minute).After(expiresAt)
}

// CalculateExpiresAt calculates the expiration time from expires_in seconds.
func CalculateExpiresAt(expiresIn int) time.Time {
	return time.Now().Add(time.Duration(expiresIn) * time.Second)
}

// XOAuth2Auth implements SMTP AUTH for XOAUTH2.
type XOAuth2Auth struct {
	Username string
	Token    string
}

// Start begins the XOAUTH2 authentication.
func (a *XOAuth2Auth) Start(server *bytes.Buffer) (string, []byte, error) {
	authString := BuildXOAuth2String(a.Username, a.Token)
	return "XOAUTH2", []byte(authString), nil
}

// Next continues the authentication (not used for XOAUTH2).
func (a *XOAuth2Auth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		return nil, fmt.Errorf("unexpected server challenge")
	}
	return nil, nil
}
