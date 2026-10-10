// Package email provides functionality for an email inbox with multiple SMTP servers and IMAP clients.
package email

import (
	"cmp"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	conversationmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
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

var _ inbox.EmailInbox = (*Email)(nil)

// Email represents the email inbox with multiple SMTP servers and IMAP clients.
type Email struct {
	id                        int
	uuid                      string
	name                      string
	smtpPools                 []*smtppool.Pool
	smtpPoolsMu               sync.RWMutex
	smtpPoolsToken            string
	smtpCfg                   []models.SMTPConfig
	imapCfg                   []models.IMAPConfig
	oauth                     *models.OAuthConfig
	oauthMu                   sync.RWMutex
	authType                  string
	headers                   map[string]string
	lo                        *logf.Logger
	from                      string
	primary                   string
	receiveAddresses          map[string]struct{}
	fromNameTemplate          string
	replyTo                   string
	enablePlusAddressing      bool
	messageStore              inbox.MessageStore
	userStore                 inbox.UserStore
	wg                        sync.WaitGroup
	tokenRefreshCallback      TokenRefreshCallback
	aliasVerificationCallback func(context.Context, string, string) error
	aliasExpiryCallback       func(context.Context, time.Time) error
	authStatusCallback        AuthStatusCallback
}

// TokenRefreshCallback is called when OAuth tokens are refreshed.
// It receives the inbox ID and the updated config with new tokens.
type TokenRefreshCallback func(inboxID int, updatedConfig models.Config) error

// AuthStatusCallback reports the provider's latest verdict on the inbox credentials. Ok=true clears a previously flagged failure.
type AuthStatusCallback func(inboxID int, ok bool)

// Opts holds the options required for the email inbox.
type Opts struct {
	ID                        int
	UUID                      string
	Name                      string
	Aliases                   models.EmailAliases
	Headers                   map[string]string
	Config                    models.Config
	Lo                        *logf.Logger
	TokenRefreshCallback      TokenRefreshCallback // Optional callback for token refresh
	AliasVerificationCallback func(context.Context, string, string) error
	AliasExpiryCallback       func(context.Context, time.Time) error
	AuthStatusCallback        AuthStatusCallback
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

	primary, err := inbox.NormalizeEmailAddress(opts.Config.From)
	receiveSet := make(map[string]struct{})
	if err != nil {
		if opts.Lo != nil {
			opts.Lo.Warn("could not normalize email inbox from address; address ownership will be empty", "from", opts.Config.From, "error", err)
		}
	} else {
		receiveSet[primary] = struct{}{}
	}
	for _, alias := range opts.Aliases {
		address, err := inbox.NormalizeEmailAddress(alias.Email)
		if err != nil {
			if opts.Lo != nil {
				opts.Lo.Warn("could not normalize email alias; address ownership will exclude it", "alias", alias.Email, "error", err)
			}
			continue
		}
		receiveSet[address] = struct{}{}
	}

	e := &Email{
		id:                        opts.ID,
		uuid:                      opts.UUID,
		name:                      opts.Name,
		headers:                   opts.Headers,
		from:                      opts.Config.From,
		primary:                   primary,
		receiveAddresses:          receiveSet,
		fromNameTemplate:          opts.Config.FromNameTemplate,
		replyTo:                   opts.Config.ReplyTo,
		smtpCfg:                   opts.Config.SMTP,
		imapCfg:                   opts.Config.IMAP,
		lo:                        opts.Lo,
		smtpPools:                 pools,
		smtpPoolsToken:            poolsToken,
		messageStore:              store,
		userStore:                 userStore,
		oauth:                     opts.Config.OAuth,
		authType:                  opts.Config.AuthType,
		enablePlusAddressing:      opts.Config.EnablePlusAddressing,
		tokenRefreshCallback:      opts.TokenRefreshCallback,
		aliasVerificationCallback: opts.AliasVerificationCallback,
		aliasExpiryCallback:       opts.AliasExpiryCallback,
		authStatusCallback:        opts.AuthStatusCallback,
	}
	return e, nil
}

// Identifier returns the unique identifier of the inbox which is the database ID.
func (e *Email) Identifier() int {
	return e.id
}

// PrimaryAddress returns the normalized primary email address.
func (e *Email) PrimaryAddress() string {
	return e.primary
}

// ReceivesAddress reports whether the address is configured to receive mail.
func (e *Email) ReceivesAddress(value string) bool {
	address, err := inbox.NormalizeEmailAddress(value)
	if err != nil {
		return false
	}
	_, ok := e.receiveAddresses[address]
	return ok
}

// StartAliasVerification sends a verification message using this inbox's SMTP pool.
func (e *Email) StartAliasVerification(alias, token string) error {
	_, domain, _ := strings.Cut(e.PrimaryAddress(), "@")
	return e.Send(conversationmodels.OutboundMessage{
		From:                   alias,
		To:                     []string{cmp.Or(e.ReplyToAddress(), e.PrimaryAddress())},
		Subject:                "libredesk alias verification",
		ContentType:            conversationmodels.ContentTypeText,
		Content:                "This message verifies the sending capability of a libredesk email alias.",
		SourceID:               "alias-verification-" + token + "@" + domain,
		AliasVerificationToken: token,
	})
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

// Name returns the inbox name.
func (e *Email) Name() string {
	return e.name
}

// FromAddress returns the from address for this inbox.
func (e *Email) FromAddress() string {
	return e.from
}

// FromNameTemplate returns the from display name template for this inbox, empty if unset.
func (e *Email) FromNameTemplate() string {
	return e.fromNameTemplate
}

// ReplyToAddress returns the reply-to address for this inbox, empty if unset.
func (e *Email) ReplyToAddress() string {
	return e.replyTo
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
		SMTP:                 e.smtpCfg,
		IMAP:                 e.imapCfg,
		From:                 e.from,
		FromNameTemplate:     e.fromNameTemplate,
		ReplyTo:              e.replyTo,
		OAuth:                oauth,
		AuthType:             e.authType,
		EnablePlusAddressing: e.enablePlusAddressing,
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
		e.flagAuthError()
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
	e.clearAuthError()
	return oauthCopy, true, nil
}

func (e *Email) flagAuthError() {
	if e.authStatusCallback != nil {
		e.authStatusCallback(e.Identifier(), false)
	}
}

func (e *Email) clearAuthError() {
	if e.authStatusCallback != nil {
		e.authStatusCallback(e.Identifier(), true)
	}
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
