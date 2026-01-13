package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/inbox"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/email/oauth"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"golang.org/x/oauth2"
)

const (
	FlowTypeNewInbox  = "new_inbox"
	FlowTypeReconnect = "reconnect"
)

// OAuthCredentialsRequest represents the OAuth credentials from the request body.
type OAuthCredentialsRequest struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	TenantID     string `json:"tenant_id,omitempty"` // Optional for Microsoft
	FlowType     string `json:"flow_type,omitempty"` // "new_inbox" or "reconnect"
	InboxID      int    `json:"inbox_id,omitempty"`  // Required for reconnect flow
}

// handleOAuthAuthorize initiates the OAuth authorization flow for creating a new email inbox.
func handleOAuthAuthorize(r *fastglue.Request) error {
	var (
		app      = r.Context.(*App)
		provider = r.RequestCtx.UserValue("provider").(string)
		req      OAuthCredentialsRequest
	)

	if provider != string(oauth.ProviderGoogle) && provider != string(oauth.ProviderMicrosoft) {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			app.i18n.Ts("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}

	// Parse request body
	if err := json.Unmarshal(r.RequestCtx.PostBody(), &req); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			app.i18n.Ts("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}

	// Validate credentials
	if req.ClientID == "" || req.ClientSecret == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.badRequest"), nil, envelope.InputError)
	}

	if req.FlowType != FlowTypeNewInbox && req.FlowType != FlowTypeReconnect {
		req.FlowType = FlowTypeNewInbox
	}

	// Build redirect URI
	redirectURI := app.consts.Load().(*constants).AppBaseURL + "/api/v1/inboxes/oauth/" + provider + "/callback"

	// Generate secure random state
	state, err := stringutil.RandomAlphanumeric(32)
	if err != nil {
		app.lo.Error("Failed to generate OAuth state", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorGenerating", "name", "state"), nil, envelope.GeneralError)
	}

	// Store OAuth data in Redis with 15 min expiry
	redisKey := "inbox_oauth:" + state
	oauthData := map[string]any{
		"provider":      provider,
		"redirect_uri":  redirectURI,
		"client_id":     req.ClientID,
		"client_secret": req.ClientSecret,
		"flow_type":     req.FlowType,
		"inbox_id":      req.InboxID,
	}

	// Add tenant ID for Microsoft if provided
	if provider == string(oauth.ProviderMicrosoft) && req.TenantID != "" {
		oauthData["tenant_id"] = req.TenantID
	}

	if err := app.redis.HSet(ctx, redisKey, oauthData).Err(); err != nil {
		app.lo.Error("Failed to store OAuth state in Redis", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}

	// Set 15 min expiry (auto-cleanup)
	if err := app.redis.Expire(ctx, redisKey, 15*time.Minute).Err(); err != nil {
		app.lo.Error("Failed to set expiry for OAuth state in Redis", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
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
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}

	// Retrieve OAuth data from Redis
	redisKey := "inbox_oauth:" + state
	oauthData, err := app.redis.HGetAll(ctx, redisKey).Result()
	if err != nil || len(oauthData) == 0 {
		app.lo.Error("Invalid or expired OAuth state", "error", err, "state", state)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			app.i18n.Ts("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}

	// Delete key after retrieval (one-time use)
	app.redis.Del(ctx, redisKey)

	// Extract values from Redis hash
	storedProvider := oauthData["provider"]
	redirectURI := oauthData["redirect_uri"]
	clientID := oauthData["client_id"]
	clientSecret := oauthData["client_secret"]
	tenantID := oauthData["tenant_id"]   // Empty string if not set
	flowType := oauthData["flow_type"]   // "new_inbox" or "reconnect"
	inboxIDStr := oauthData["inbox_id"]  // Inbox ID for reconnect flow

	// Validate provider matches URL parameter
	if storedProvider != provider {
		app.lo.Error("Provider mismatch", "stored", storedProvider, "url", provider)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}

	// Exchange authorization code for tokens
	token, err := oauth.ExchangeCodeForToken(
		context.Background(),
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
	userEmail, err := getUserEmailFromProvider(provider, token)
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
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
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

	// Validate flow type matches actual state
	// New inbox flow: reject if inbox already exists
	if flowType == FlowTypeNewInbox && existingInbox != nil {
		app.lo.Error("Inbox already exists for email", "email", userEmail)
		return r.Redirect("/admin/inboxes?error=inbox_already_exists", fasthttp.StatusFound, nil, "")
	}

	// Reconnect flow: validate inbox_id and email match
	if flowType == FlowTypeReconnect {
		// Find the target inbox by ID
		var targetInbox *imodels.Inbox
		for i, existing := range existingInboxes {
			if fmt.Sprintf("%d", existing.ID) == inboxIDStr {
				targetInbox = &existingInboxes[i]
				break
			}
		}

		if targetInbox == nil {
			app.lo.Error("Target inbox not found for reconnect", "inbox_id", inboxIDStr)
			return r.Redirect("/admin/inboxes?error=inbox_not_found", fasthttp.StatusFound, nil, "")
		}

		// Verify the authorized email matches the target inbox's email
		targetEmailAddr, _ := stringutil.ExtractEmail(targetInbox.From)
		if targetEmailAddr != userEmailAddr {
			app.lo.Error("Email mismatch during reconnect", "target_email", targetEmailAddr, "authorized_email", userEmailAddr)
			return r.Redirect("/admin/inboxes?error=email_mismatch", fasthttp.StatusFound, nil, "")
		}

		existingInbox = targetInbox
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
			AccessToken:  token.AccessToken,
			RefreshToken: token.RefreshToken,
			ExpiresAt:    token.Expiry,
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

		// Update inbox config directly (bypasses preservation logic that could corrupt OAuth tokens)
		if err := app.inbox.UpdateConfig(existingInbox.ID, json.RawMessage(configJSON)); err != nil {
			app.lo.Error("Failed to update inbox config", "error", err)
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
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresAt:    token.Expiry,
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
		Channel:     inbox.ChannelEmail,
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
func getUserEmailFromProvider(provider string, token *oauth2.Token) (string, error) {
	switch provider {
	case string(oauth.ProviderMicrosoft):
		idToken, ok := token.Extra("id_token").(string)
		if !ok {
			return "", fmt.Errorf("no id_token")
		}

		parts := strings.Split(idToken, ".")
		if len(parts) < 2 {
			return "", fmt.Errorf("invalid id_token")
		}

		payload, err := base64.RawURLEncoding.DecodeString(parts[1])
		if err != nil {
			return "", err
		}

		var claims map[string]any
		json.Unmarshal(payload, &claims)

		if email, ok := claims["email"].(string); ok && email != "" {
			return email, nil
		}
		if upn, ok := claims["preferred_username"].(string); ok {
			return upn, nil
		}
		return "", fmt.Errorf("email not found")
	case string(oauth.ProviderGoogle):
		req, _ := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
		req.Header.Set("Authorization", "Bearer "+token.AccessToken)

		resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		if email, ok := result["email"].(string); ok {
			return email, nil
		}
		return "", fmt.Errorf("email not found")
	}

	return "", fmt.Errorf("unsupported provider")
}

// getProviderDefaults returns provider-specific SMTP and IMAP configurations.
func getProviderDefaults(provider, emailAddr string) (imodels.SMTPConfig, imodels.IMAPConfig) {
	var smtp imodels.SMTPConfig
	var imap imodels.IMAPConfig

	// Common settings
	smtp.Username = emailAddr
	smtp.AuthProtocol = "login"
	smtp.TLSSkipVerify = false
	smtp.MaxConns = 10
	smtp.MaxMessageRetries = 2
	smtp.IdleTimeout = "20s"
	smtp.PoolWaitTimeout = "30s"

	imap.Username = emailAddr
	imap.Mailbox = "INBOX"
	imap.ReadInterval = "5m"
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
