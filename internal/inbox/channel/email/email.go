// Package email provides functionality for an email inbox with multiple SMTP servers and IMAP clients.
package email

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/abhinavxd/libredesk/internal/inbox"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/email/oauth"
	"github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/knadh/smtppool"
	"github.com/zerodha/logf"
	xoauth2 "golang.org/x/oauth2"
)

const (
	ChannelEmail = "email"
)

// Email represents the email inbox with multiple SMTP servers and IMAP clients.
type Email struct {
	id                   int
	smtpPools            []*smtppool.Pool
	smtpPoolsMu          sync.RWMutex
	smtpPoolsToken       string
	smtpCfg              []models.SMTPConfig
	imapCfg              []models.IMAPConfig
	oauth                *models.OAuthConfig
	oauthMu              sync.RWMutex
	authType             string
	headers              map[string]string
	lo                   *logf.Logger
	from                 string
	messageStore         inbox.MessageStore
	userStore            inbox.UserStore
	wg                   sync.WaitGroup
	tokenRefreshCallback TokenRefreshCallback
}

// TokenRefreshCallback is called when OAuth tokens are refreshed.
// It receives the inbox ID and the updated config with new tokens.
type TokenRefreshCallback func(inboxID int, updatedConfig models.Config) error

// Opts holds the options required for the email inbox.
type Opts struct {
	ID                   int
	Headers              map[string]string
	Config               models.Config
	Lo                   *logf.Logger
	TokenRefreshCallback TokenRefreshCallback // Optional callback for token refresh
}

// New returns a new instance of the email inbox.
func New(store inbox.MessageStore, userStore inbox.UserStore, opts Opts) (*Email, error) {
	pools, err := NewSmtpPool(opts.Config.SMTP, opts.Config.OAuth)
	if err != nil {
		return nil, err
	}

	var poolsToken string
	if opts.Config.OAuth != nil {
		poolsToken = opts.Config.OAuth.AccessToken
	}

	e := &Email{
		id:                   opts.ID,
		headers:              opts.Headers,
		from:                 opts.Config.From,
		smtpCfg:              opts.Config.SMTP,
		imapCfg:              opts.Config.IMAP,
		lo:                   opts.Lo,
		smtpPools:            pools,
		smtpPoolsToken:       poolsToken,
		messageStore:         store,
		userStore:            userStore,
		oauth:                opts.Config.OAuth,
		authType:             opts.Config.AuthType,
		tokenRefreshCallback: opts.TokenRefreshCallback,
	}
	return e, nil
}

// Identifier returns the unique identifier of the inbox which is the database ID.
func (e *Email) Identifier() int {
	return e.id
}

// Receive starts reading incoming messages for each IMAP client.
func (e *Email) Receive(ctx context.Context) error {
	for _, cfg := range e.imapCfg {
		e.wg.Add(1)
		go func(cfg models.IMAPConfig) {
			defer e.wg.Done()
			if err := e.ReadIncomingMessages(ctx, cfg); err != nil {
				e.lo.Error("error reading incoming messages", "error", err)
			}
		}(cfg)
	}
	e.wg.Wait()
	return nil
}

// Close cloes email channel by closing the smtp pool
func (e *Email) Close() error {
	return e.closeSMTPPool()
}

// FromAddress returns the from address for this inbox.
func (e *Email) FromAddress() string {
	return e.from
}

// Channel returns the channel name for this inbox.
func (e *Email) Channel() string {
	return ChannelEmail
}

// getCurrentConfig returns the current config with all SMTP and IMAP settings.
func (e *Email) getCurrentConfig() models.Config {
	e.oauthMu.RLock()
	oauth := e.oauth
	e.oauthMu.RUnlock()

	return models.Config{
		SMTP:     e.smtpCfg,
		IMAP:     e.imapCfg,
		From:     e.from,
		OAuth:    oauth,
		AuthType: e.authType,
	}
}

// refreshOAuthIfNeeded checks if OAuth token is expired and refreshes it if needed.
// Returns a copy of the oauth config and whether it was refreshed.
func (e *Email) refreshOAuthIfNeeded() (*models.OAuthConfig, bool, error) {
	if e.authType != models.AuthTypeOAuth2 {
		return nil, false, nil
	}

	e.oauthMu.Lock()

	// Check if token is expired
	if !oauth.IsTokenExpired(e.oauth.ExpiresAt) {
		// Token is still valid, just copy and return
		oauthCopy := e.oauth
		e.oauthMu.Unlock()
		return oauthCopy, false, nil
	}

	e.lo.Info("OAuth token expired, attempting refresh", "inbox_id", e.Identifier(), "expires_at", e.oauth.ExpiresAt)

	// Attempt to refresh the token
	newOAuth, err := RefreshOAuthConfig(e.oauth)
	if err != nil {
		e.oauthMu.Unlock()
		e.lo.Error("Failed to refresh OAuth token", "inbox_id", e.Identifier(), "error", err)
		return nil, false, fmt.Errorf("OAuth token expired and refresh failed for inbox %d: %w", e.Identifier(), err)
	}

	// Update config with new tokens
	e.oauth = newOAuth
	oauthCopy := newOAuth
	e.oauthMu.Unlock()

	// Persist tokens via callback if available.
	if e.tokenRefreshCallback != nil {
		updatedConfig := e.getCurrentConfig()
		if err := e.tokenRefreshCallback(e.Identifier(), updatedConfig); err != nil {
			e.lo.Error("Failed to persist refreshed tokens", "inbox_id", e.Identifier(), "error", err)
		}
	}

	e.lo.Info("Successfully refreshed OAuth token", "inbox_id", e.Identifier())
	return oauthCopy, true, nil
}

// closeSMTPPool closes the smtp pool.
func (e *Email) closeSMTPPool() error {
	e.smtpPoolsMu.Lock()
	for _, p := range e.smtpPools {
		p.Close()
	}
	e.smtpPoolsMu.Unlock()
	return nil
}

// RefreshOAuthConfig refreshes an expired OAuth token and returns a new OAuth config.
func RefreshOAuthConfig(currentToken *models.OAuthConfig) (*models.OAuthConfig, error) {
	if currentToken.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh token available")
	}

	clientID := currentToken.ClientID
	clientSecret := currentToken.ClientSecret
	tenantID := currentToken.TenantID

	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("OAuth credentials missing for provider '%s'", currentToken.Provider)
	}

	cfg, err := oauth.GetOAuth2Config(
		oauth.Provider(currentToken.Provider),
		clientID,
		clientSecret,
		"",
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth config: %w", err)
	}

	oldToken := &xoauth2.Token{
		RefreshToken: currentToken.RefreshToken,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	src := cfg.TokenSource(ctx, oldToken)
	newToken, err := src.Token()
	if err != nil {
		return nil, fmt.Errorf("token refresh failed: %w", err)
	}

	// Create new OAuth config with refreshed tokens, preserving credentials
	newOAuthConfig := &models.OAuthConfig{
		Provider:     currentToken.Provider,
		AccessToken:  newToken.AccessToken,
		RefreshToken: newToken.RefreshToken,
		ExpiresAt:    newToken.Expiry,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TenantID:     tenantID,
	}

	// Use new refresh token if provided, else keep old one
	if newOAuthConfig.RefreshToken == "" {
		newOAuthConfig.RefreshToken = currentToken.RefreshToken
	}

	return newOAuthConfig, nil
}
