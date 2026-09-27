package models

import (
	"crypto/tls"
	"encoding/json"
	"net/smtp"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/volatiletech/null/v9"
)

// Authentication type constants.
const (
	AuthTypePassword = "password"
	AuthTypeOAuth2   = "oauth2"
)

// Transport type constants for the email channel.
const (
	TransportSMTPIMAP = "smtp_imap" // Traditional SMTP (send) + IMAP (receive).
	TransportHTTPAPI  = "http_api"  // HTTPS transactional email API (send) + provider webhook (receive).
)

// HTTPAPIProviderResend identifies the Resend HTTP API provider.
const HTTPAPIProviderResend = "resend"

// Inbox represents a inbox record in DB.
type Inbox struct {
	ID                 int             `db:"id" json:"id"`
	UUID               string          `db:"uuid" json:"uuid"`
	CreatedAt          time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt          time.Time       `db:"updated_at" json:"updated_at"`
	Name               string          `db:"name" json:"name"`
	Channel            string          `db:"channel" json:"channel"`
	Enabled            bool            `db:"enabled" json:"enabled"`
	CSATEnabled        bool            `db:"csat_enabled" json:"csat_enabled"`
	PromptTagsOnReply  bool            `db:"prompt_tags_on_reply" json:"prompt_tags_on_reply"`
	From               string          `db:"from" json:"from"`
	FromNameTemplate   string          `db:"from_name_template" json:"from_name_template"`
	Config             json.RawMessage `db:"config" json:"config"`
	Secret             null.String     `db:"secret" json:"secret"`
	LinkedEmailInboxID null.Int        `db:"linked_email_inbox_id" json:"linked_email_inbox_id"`
}

// Config holds the email inbox configuration with multiple SMTP servers and IMAP clients.
type Config struct {
	Transport            string         `json:"transport"` // TransportSMTPIMAP (default) or TransportHTTPAPI
	AuthType             string         `json:"auth_type"` // AuthTypePassword or AuthTypeOAuth2
	OAuth                *OAuthConfig   `json:"oauth"`      // OAuth config when auth_type is "oauth2"
	SMTP                 []SMTPConfig   `json:"smtp"`
	IMAP                 []IMAPConfig   `json:"imap"`
	HTTPAPI              *HTTPAPIConfig `json:"http_api"` // HTTP API config when transport is "http_api"
	From                 string         `json:"from"`
	FromNameTemplate     string         `json:"from_name_template"`
	ReplyTo              string         `json:"reply_to"`
	EnablePlusAddressing bool           `json:"enable_plus_addressing"`
}

// IsHTTPAPITransport returns true if the config uses the HTTP API transport (e.g. Resend)
// instead of SMTP/IMAP.
func (c Config) IsHTTPAPITransport() bool {
	return c.Transport == TransportHTTPAPI
}

// HTTPAPIConfig holds credentials for sending/receiving email via a transactional email HTTP API.
type HTTPAPIConfig struct {
	Provider      string `json:"provider"`       // e.g. "resend"
	APIKey        string `json:"api_key"`        // Provider API key used to send mail
	WebhookSecret string `json:"webhook_secret"` // Secret used to verify inbound webhook signatures
}

// OAuthConfig holds OAuth 2.0 authentication details.
type OAuthConfig struct {
	Provider     string    `json:"provider"`      // "microsoft" or "google"
	AccessToken  string    `json:"access_token"`  // Current access token
	RefreshToken string    `json:"refresh_token"` // Refresh token for getting new access tokens
	ExpiresAt    time.Time `json:"expires_at"`    // When the access token expires
	ClientID     string    `json:"client_id"`     // OAuth client ID
	ClientSecret string    `json:"client_secret"` // OAuth client secret
	TenantID     string    `json:"tenant_id"`     // Microsoft tenant ID
}

// SMTPConfig represents an SMTP server's credentials with the smtppool options.
type SMTPConfig struct {
	Username      string `json:"username"`
	Password      string `json:"password"`
	AuthProtocol  string `json:"auth_protocol"`
	TLSType       string `json:"tls_type"`
	TLSSkipVerify bool   `json:"tls_skip_verify"`

	// SMTP pool options (from embedded smtppool.Opt)
	Host              string `json:"host"`
	Port              int    `json:"port"`
	HelloHostname     string `json:"hello_hostname"`
	MaxConns          int    `json:"max_conns"`
	MaxMessageRetries int    `json:"max_msg_retries"`
	IdleTimeout       string `json:"idle_timeout"`
	PoolWaitTimeout   string `json:"pool_wait_timeout"`
	SSL               bool   `json:"ssl"`
	// Auth is the smtp.Auth authentication scheme.
	Auth smtp.Auth `json:"-"`
	// TLSConfig is the optional TLS configuration.
	TLSConfig *tls.Config `json:"-"`
}

// IMAPConfig holds IMAP client credentials and configuration.
type IMAPConfig struct {
	Host           string `json:"host"`
	Port           int    `json:"port"`
	Username       string `json:"username"`
	Password       string `json:"password"`
	Mailbox        string `json:"mailbox"`
	ReadInterval   string `json:"read_interval"`
	ScanInboxSince string `json:"scan_inbox_since"`
	TLSType        string `json:"tls_type"`
	TLSSkipVerify  bool   `json:"tls_skip_verify"`
}

// ClearPasswords masks all config passwords
func (m *Inbox) ClearPasswords() error {
	switch m.Channel {
	case "email":
		var cfg map[string]interface{}
		if err := json.Unmarshal(m.Config, &cfg); err != nil {
			return err
		}

		dummyPassword := strings.Repeat(stringutil.PasswordDummy, 10)

		// Clear IMAP passwords
		if imapSlice, ok := cfg["imap"].([]interface{}); ok {
			for _, imapItem := range imapSlice {
				if imapMap, ok := imapItem.(map[string]interface{}); ok {
					imapMap["password"] = dummyPassword
				}
			}
		}

		// Clear SMTP passwords
		if smtpSlice, ok := cfg["smtp"].([]interface{}); ok {
			for _, smtpItem := range smtpSlice {
				if smtpMap, ok := smtpItem.(map[string]interface{}); ok {
					smtpMap["password"] = dummyPassword
				}
			}
		}

		// Clear OAuth sensitive fields if present
		if oauthMap, ok := cfg["oauth"].(map[string]interface{}); ok {
			oauthMap["access_token"] = dummyPassword
			oauthMap["refresh_token"] = dummyPassword
			oauthMap["client_secret"] = dummyPassword
		}

		// Clear HTTP API sensitive fields if present
		if httpAPIMap, ok := cfg["http_api"].(map[string]interface{}); ok {
			if apiKey, ok := httpAPIMap["api_key"].(string); ok && apiKey != "" {
				httpAPIMap["api_key"] = dummyPassword
			}
			if webhookSecret, ok := httpAPIMap["webhook_secret"].(string); ok && webhookSecret != "" {
				httpAPIMap["webhook_secret"] = dummyPassword
			}
		}

		clearedConfig, err := json.Marshal(cfg)
		if err != nil {
			return err
		}

		m.Config = clearedConfig
	case "livechat":
		// Mask the secret field for livechat
		if m.Secret.Valid && m.Secret.String != "" {
			m.Secret = null.StringFrom(strings.Repeat(stringutil.PasswordDummy, 10))
		}
	default:
		return nil
	}

	return nil
}
