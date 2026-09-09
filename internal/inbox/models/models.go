package models

import (
	"crypto/tls"
	"encoding/json"
	"net/smtp"
	"sort"
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
	AuthType             string       `json:"auth_type"` // AuthTypePassword or AuthTypeOAuth2
	OAuth                *OAuthConfig `json:"oauth"`     // OAuth config when auth_type is "oauth2"
	SMTP                 []SMTPConfig `json:"smtp"`
	IMAP                 []IMAPConfig `json:"imap"`
	From                 string       `json:"from"`
	FromNameTemplate     string       `json:"from_name_template"`
	ReplyTo              string       `json:"reply_to"`
	EnablePlusAddressing bool         `json:"enable_plus_addressing"`
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

// Mailbox purge outcomes, reported per Message-ID by MailboxPurger implementations.
const (
	// MailMovedToTrash means the mail now sits in the mailbox's Trash folder and can be
	// restored from a mail client.
	MailMovedToTrash = "moved_to_trash"
	// MailExpunged means the mail was permanently removed because the server has no Trash folder.
	MailExpunged = "expunged"
	// MailNotFound means no mail with that Message-ID was in the mailbox.
	MailNotFound = "not_found"
	// MailNotPurged means the mail was found but left untouched because the server offers no safe
	// way to remove it.
	MailNotPurged = "not_purged"
	// MailPurgeFailed means the server rejected the search, the move or the delete.
	MailPurgeFailed = "failed"
)

// MailPurge is what the mailbox purge did with one Message-ID.
type MailPurge struct {
	MessageID string `json:"message_id"`
	Outcome   string `json:"outcome"`
	// Reason carries the detail behind a not_purged or failed outcome.
	Reason string `json:"reason,omitempty"`
}

// MailPurgeResult reports a finished mailbox purge.
type MailPurgeResult struct {
	Mails []MailPurge `json:"mails"`
	// TrashMailbox names the folder the mails were moved to, empty when none was found.
	TrashMailbox string `json:"trash_mailbox,omitempty"`
}

// Record appends an outcome for a Message-ID, replacing an earlier one from another mailbox.
func (r *MailPurgeResult) Record(messageID, outcome, reason string) {
	for i, mail := range r.Mails {
		if mail.MessageID == messageID {
			r.Mails[i] = MailPurge{MessageID: messageID, Outcome: outcome, Reason: reason}
			return
		}
	}
	r.Mails = append(r.Mails, MailPurge{MessageID: messageID, Outcome: outcome, Reason: reason})
}

// Count returns how many mails ended with the given outcome.
func (r MailPurgeResult) Count(outcome string) int {
	var n int
	for _, mail := range r.Mails {
		if mail.Outcome == outcome {
			n++
		}
	}
	return n
}

// Unpurged returns the Message-IDs the purge did not take off the server, in a stable order.
// A mail that was never found is included: the desk cannot tell an already-deleted mail from one
// the search could not match, and both leave the caller with nothing to confirm.
func (r MailPurgeResult) Unpurged() []string {
	var ids []string
	for _, mail := range r.Mails {
		switch mail.Outcome {
		case MailMovedToTrash, MailExpunged:
		default:
			ids = append(ids, mail.MessageID)
		}
	}
	sort.Strings(ids)
	return ids
}
