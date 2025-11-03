package models

import (
	"crypto/tls"
	"encoding/json"
	"net/smtp"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/stringutil"
)

// Authentication type constants.
const (
	AuthTypePassword = "password"
	AuthTypeOAuth2   = "oauth2"
)

// Inbox represents a inbox record in DB.
type Inbox struct {
	ID          int             `db:"id" json:"id"`
	CreatedAt   time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at" json:"updated_at"`
	Name        string          `db:"name" json:"name"`
	Channel     string          `db:"channel" json:"channel"`
	Enabled     bool            `db:"enabled" json:"enabled"`
	CSATEnabled bool            `db:"csat_enabled" json:"csat_enabled"`
	From        string          `db:"from" json:"from"`
	Config      json.RawMessage `db:"config" json:"config"`
}

// Config holds the email inbox configuration with multiple SMTP servers and IMAP clients.
type Config struct {
	AuthType string       `json:"auth_type"`       // AuthTypePassword or AuthTypeOAuth2
	OAuth    *OAuthConfig `json:"oauth,omitempty"` // OAuth config when auth_type is "oauth2"
	SMTP     []SMTPConfig `json:"smtp"`
	IMAP     []IMAPConfig `json:"imap"`
	From     string       `json:"from"`
}

// OAuthConfig holds OAuth 2.0 authentication details.
type OAuthConfig struct {
	Provider     string    `json:"provider"`                // "microsoft" or "google"
	AccessToken  string    `json:"access_token"`            // Current access token
	RefreshToken string    `json:"refresh_token"`           // Refresh token for getting new access tokens
	ExpiresAt    time.Time `json:"expires_at"`              // When the access token expires
	ClientID     string    `json:"client_id,omitempty"`     // OAuth client ID
	ClientSecret string    `json:"client_secret,omitempty"` // OAuth client secret
	TenantID     string    `json:"tenant_id,omitempty"`     // Microsoft tenant ID
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

		clearedConfig, err := json.Marshal(cfg)
		if err != nil {
			return err
		}

		m.Config = clearedConfig

	default:
		return nil
	}

	return nil
}
