package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/email/oauth"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// OAuthCredentialsRequest represents the OAuth credentials from the request body.
type OAuthCredentialsRequest struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	TenantID     string `json:"tenant_id,omitempty"` // Optional for Microsoft
}

// handleOAuthAuthorize initiates the OAuth authorization flow for creating a new email inbox.
func handleOAuthAuthorize(r *fastglue.Request) error {
	var (
		app      = r.Context.(*App)
		provider = r.RequestCtx.UserValue("provider").(string)
		req      OAuthCredentialsRequest
	)

	// Validate provider
	if provider != string(oauth.ProviderGoogle) && provider != string(oauth.ProviderMicrosoft) {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			"Invalid provider. Supported providers: google, microsoft", nil, envelope.InputError)
	}

	// Parse request body
	if err := json.Unmarshal(r.RequestCtx.PostBody(), &req); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			"Invalid request body", nil, envelope.InputError)
	}

	// Validate credentials
	if req.ClientID == "" || req.ClientSecret == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			"Client ID and Client Secret are required", nil, envelope.InputError)
	}

	// Build redirect URI
	redirectURI := app.consts.Load().(*constants).AppBaseURL + "/api/v1/inboxes/oauth/" + provider + "/callback"

	// Generate secure random state
	state, err := stringutil.RandomAlphanumeric(32)
	if err != nil {
		app.lo.Error("Failed to generate OAuth state", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorGenerating", "name", "state"), nil, envelope.GeneralError)
	}

	// Store state data and OAuth credentials in session
	sessionData := map[string]any{
		"oauth_state_" + state:         state,
		"oauth_provider_" + state:      provider,
		"oauth_redirect_uri_" + state:  redirectURI,
		"oauth_client_id_" + state:     req.ClientID,
		"oauth_client_secret_" + state: req.ClientSecret,
		"oauth_timestamp_" + state:     strconv.FormatInt(time.Now().Unix(), 10),
	}

	// Add tenant ID for Microsoft if provided
	if provider == string(oauth.ProviderMicrosoft) && req.TenantID != "" {
		sessionData["oauth_tenant_id_"+state] = req.TenantID
	}

	if err := app.auth.SetSessionValues(r, sessionData); err != nil {
		app.lo.Error("Failed to store OAuth state in session", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to initialize OAuth flow", nil, envelope.GeneralError)
	}

	// Build authorization URL with scopes
	authURL, err := oauth.BuildAuthorizationURL(
		oauth.Provider(provider),
		req.ClientID,
		redirectURI,
		state,
		req.TenantID,
	)
	if err != nil {
		app.lo.Error("Failed to build authorization URL", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, envelope.InputError)
	}

	return r.SendEnvelope(authURL)
}

// handleOAuthCallback handles the OAuth callback and auto-creates an inbox.
func handleOAuthCallback(r *fastglue.Request) error {
	var (
		app      = r.Context.(*App)
		provider = r.RequestCtx.UserValue("provider").(string)
		code     = string(r.RequestCtx.QueryArgs().Peek("code"))
		state    = string(r.RequestCtx.QueryArgs().Peek("state"))
	)

	// Check if user denied authorization
	if code == "" {
		errorMsg := string(r.RequestCtx.QueryArgs().Peek("error"))
		errorDesc := string(r.RequestCtx.QueryArgs().Peek("error_description"))
		app.lo.Error("OAuth authorization failed", "error", errorMsg, "description", errorDesc)
		return r.Redirect("/admin/inboxes?error=oauth_denied", fasthttp.StatusFound, nil, "")
	}

	if state == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Missing state parameter", nil, envelope.InputError)
	}

	// Retrieve and validate state from session
	_, err := app.auth.GetSessionValue(r, "oauth_state_"+state)
	if err != nil {
		app.lo.Error("Invalid or expired OAuth state", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			"Invalid or expired state parameter", nil, envelope.InputError)
	}

	// Get individual session values
	providerRaw, err := app.auth.GetSessionValue(r, "oauth_provider_"+state)
	if err != nil {
		app.lo.Error("Failed to get provider from session", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError,
			"Failed to process OAuth callback", nil, envelope.GeneralError)
	}

	redirectURIRaw, err := app.auth.GetSessionValue(r, "oauth_redirect_uri_"+state)
	if err != nil {
		app.lo.Error("Failed to get redirect URI from session", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError,
			"Failed to process OAuth callback", nil, envelope.GeneralError)
	}

	sessionProvider := providerRaw.(string)
	redirectURI := redirectURIRaw.(string)

	// Validate provider matches URL parameter
	if sessionProvider != provider {
		app.lo.Error("Provider mismatch", "session", sessionProvider, "url", provider)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid provider in callback", nil, envelope.InputError)
	}

	// Validate OAuth flow timestamp (must be within 15 minutes)
	timestampRaw, err := app.auth.GetSessionValue(r, "oauth_timestamp_"+state)
	if err != nil {
		app.lo.Error("Failed to get timestamp from session", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "OAuth flow expired or invalid", nil, envelope.InputError)
	}
	timestamp, _ := strconv.ParseInt(timestampRaw.(string), 10, 64)
	if time.Now().Unix()-timestamp > 900 {
		app.lo.Error("OAuth flow expired", "timestamp", timestamp)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "OAuth flow expired. Please try again.", nil, envelope.InputError)
	}

	// Retrieve OAuth credentials from session
	clientIDRaw, err := app.auth.GetSessionValue(r, "oauth_client_id_"+state)
	if err != nil {
		app.lo.Error("Failed to get client ID from session", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError,
			"Failed to process OAuth callback", nil, envelope.GeneralError)
	}

	clientSecretRaw, err := app.auth.GetSessionValue(r, "oauth_client_secret_"+state)
	if err != nil {
		app.lo.Error("Failed to get client secret from session", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError,
			"Failed to process OAuth callback", nil, envelope.GeneralError)
	}

	clientID := clientIDRaw.(string)
	clientSecret := clientSecretRaw.(string)

	// Get tenant ID for Microsoft if stored in session
	var tenantID string
	if provider == string(oauth.ProviderMicrosoft) {
		tenantIDRaw, err := app.auth.GetSessionValue(r, "oauth_tenant_id_"+state)
		if err == nil {
			tenantID = tenantIDRaw.(string)
		}
	}

	// Exchange authorization code for tokens
	tokenResp, err := oauth.ExchangeCodeForToken(
		oauth.Provider(provider),
		clientID,
		clientSecret,
		code,
		redirectURI,
		tenantID,
	)
	if err != nil {
		app.lo.Error("Failed to exchange code for tokens", "error", err)
		return r.Redirect("/admin/inboxes?error=token_exchange_failed", fasthttp.StatusFound, nil, "")
	}

	// Get user email from provider
	userEmail, err := getUserEmailFromProvider(provider, tokenResp.AccessToken)
	if err != nil {
		app.lo.Error("Failed to get user email from provider", "error", err)
		return r.Redirect("/admin/inboxes?error=email_fetch_failed", fasthttp.StatusFound, nil, "")
	}

	if userEmail == "" {
		app.lo.Error("User email not found from provider")
		return r.Redirect("/admin/inboxes?error=email_fetch_failed", fasthttp.StatusFound, nil, "")
	}

	// Check if inbox with this email already exists
	existingInboxes, err := app.inbox.GetAll()
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to check existing inboxes", nil, envelope.GeneralError)
	}

	// Extract email address for comparison (handles "Name <email>" format)
	userEmailAddr, err := stringutil.ExtractEmail(userEmail)
	if err != nil {
		app.lo.Error("error extracting email address", "email", userEmail, "error", err)
		// Fallback
		userEmailAddr = userEmail
	}

	var existingInbox *imodels.Inbox
	for i, existing := range existingInboxes {
		existingEmailAddr, err := stringutil.ExtractEmail(existing.From)
		if err != nil {
			existingEmailAddr = existing.From
		}

		if existingEmailAddr == userEmailAddr {
			existingInbox = &existingInboxes[i]
			break
		}
	}

	// If inbox exists, update it with new OAuth tokens (reconnect flow)
	if existingInbox != nil {
		app.lo.Info("Updating existing inbox with new OAuth tokens", "email", userEmail, "inbox_id", existingInbox.ID)

		// Parse existing config
		var existingConfig imodels.Config
		if err := json.Unmarshal(existingInbox.Config, &existingConfig); err != nil {
			app.lo.Error("Failed to unmarshal existing config", "error", err)
			return r.Redirect("/admin/inboxes?error=config_parse_failed", fasthttp.StatusFound, nil, "")
		}

		// Update OAuth section with new tokens
		oauthConfig := &imodels.OAuthConfig{
			Provider:     provider,
			AccessToken:  tokenResp.AccessToken,
			RefreshToken: tokenResp.RefreshToken,
			ExpiresAt:    oauth.CalculateExpiresAt(tokenResp.ExpiresIn),
			ClientID:     clientID,
			ClientSecret: clientSecret,
			TenantID:     tenantID,
		}
		existingConfig.OAuth = oauthConfig
		existingConfig.AuthType = imodels.AuthTypeOAuth2

		// Marshal updated config
		configJSON, err := json.Marshal(existingConfig)
		if err != nil {
			app.lo.Error("Failed to marshal updated config", "error", err)
			return r.Redirect("/admin/inboxes?error=config_update_failed", fasthttp.StatusFound, nil, "")
		}

		// Update inbox
		existingInbox.Config = json.RawMessage(configJSON)
		_, err = app.inbox.Update(existingInbox.ID, *existingInbox)
		if err != nil {
			app.lo.Error("Failed to update inbox", "error", err)
			return r.Redirect("/admin/inboxes?error=inbox_update_failed", fasthttp.StatusFound, nil, "")
		}

		// Reload inboxes to apply new tokens
		if err := reloadInboxes(app); err != nil {
			app.lo.Error("Failed to reload inboxes", "error", err)
		}

		return r.Redirect("/admin/inboxes?success=oauth_reconnected", fasthttp.StatusFound, nil, "")
	}

	// Get provider-specific defaults
	smtpConfig, imapConfig := getProviderDefaults(provider, userEmail)

	// Create OAuth config for tokens
	oauthConfig := &imodels.OAuthConfig{
		Provider:     provider,
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    oauth.CalculateExpiresAt(tokenResp.ExpiresIn),
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TenantID:     tenantID,
	}

	// Create inbox config
	config := imodels.Config{
		SMTP:     []imodels.SMTPConfig{smtpConfig},
		IMAP:     []imodels.IMAPConfig{imapConfig},
		From:     userEmail,
		AuthType: imodels.AuthTypeOAuth2,
		OAuth:    oauthConfig,
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		app.lo.Error("Failed to marshal inbox config", "error", err)
		return r.Redirect("/admin/inboxes?error=config_creation_failed", fasthttp.StatusFound, nil, "")
	}

	// Create inbox
	newInbox := imodels.Inbox{
		Name:        fmt.Sprintf("%s Inbox", userEmail),
		From:        userEmail,
		Channel:     "email",
		Enabled:     true,
		CSATEnabled: false,
		Config:      json.RawMessage(configJSON),
	}

	_, err = app.inbox.Create(newInbox)
	if err != nil {
		app.lo.Error("Failed to create inbox", "error", err)
		return r.Redirect("/admin/inboxes?error=inbox_creation_failed", fasthttp.StatusFound, nil, "")
	}

	// Reload inboxes to start the new inbox
	if err := reloadInboxes(app); err != nil {
		app.lo.Error("Failed to reload inboxes", "error", err)
	}

	return r.Redirect("/admin/inboxes?success=oauth_connected", fasthttp.StatusFound, nil, "")
}

// getUserEmailFromProvider fetches the user's email from the OAuth provider.
func getUserEmailFromProvider(provider, accessToken string) (string, error) {
	var (
		apiURL     string
		emailField string
	)

	switch provider {
	case string(oauth.ProviderGoogle):
		apiURL = "https://www.googleapis.com/oauth2/v2/userinfo"
		emailField = "email"
	case string(oauth.ProviderMicrosoft):
		apiURL = "https://graph.microsoft.com/v1.0/me"
		emailField = "mail" // Microsoft uses "mail" not "email"
	default:
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}

	// Create HTTP request
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	// Send request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Parse response
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}

	email, ok := result[emailField].(string)
	if !ok || email == "" {
		// For Microsoft, try "userPrincipalName" as fallback
		if provider == string(oauth.ProviderMicrosoft) {
			if upn, ok := result["userPrincipalName"].(string); ok {
				return upn, nil
			}
		}
		return "", fmt.Errorf("email not found in response")
	}

	return email, nil
}

// getProviderDefaults returns provider-specific SMTP and IMAP configurations.
func getProviderDefaults(provider, emailAddr string) (imodels.SMTPConfig, imodels.IMAPConfig) {
	var smtp imodels.SMTPConfig
	var imap imodels.IMAPConfig

	// Common settings
	smtp.Username = emailAddr
	smtp.AuthProtocol = "login"
	smtp.TLSSkipVerify = false
	// SMTP pool options (from embedded smtppool.Opt)
	smtp.MaxConns = 10
	smtp.MaxMessageRetries = 2
	smtp.IdleTimeout = "20s"
	smtp.PoolWaitTimeout = "30s"

	imap.Username = emailAddr
	imap.Mailbox = "INBOX"
	imap.ReadInterval = "10s"
	imap.ScanInboxSince = "24h"
	imap.TLSSkipVerify = false

	// Provider-specific settings
	switch provider {
	case string(oauth.ProviderGoogle):
		smtp.Host = "smtp.gmail.com"
		smtp.Port = 587
		smtp.TLSType = "starttls"
		imap.Host = "imap.gmail.com"
		imap.Port = 993
		imap.TLSType = "tls"
	case string(oauth.ProviderMicrosoft):
		smtp.Host = "smtp.office365.com"
		smtp.Port = 587
		smtp.TLSType = "starttls"
		imap.Host = "outlook.office365.com"
		imap.Port = 993
		imap.TLSType = "tls"
	}

	return smtp, imap
}
