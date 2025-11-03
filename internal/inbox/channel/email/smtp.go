package email

import (
	"crypto/tls"
	"fmt"
	"math/rand"
	"net/smtp"
	"net/textproto"
	"time"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/knadh/smtppool"
)

const (
	headerReturnPath              = "Return-Path"
	headerMessageID               = "Message-ID"
	headerReferences              = "References"
	headerInReplyTo               = "In-Reply-To"
	headerLibredeskLoopPrevention = "X-Libredesk-Loop-Prevention"
	headerAutoreply               = "X-Autoreply"
	headerAutoSubmitted           = "Auto-Submitted"

	dispositionInline = "inline"
)

// NewSmtpPool returns a smtppool
func NewSmtpPool(configs []imodels.SMTPConfig, oauth *imodels.OAuthConfig) ([]*smtppool.Pool, error) {
	pools := make([]*smtppool.Pool, 0, len(configs))

	for _, cfg := range configs {
		var auth smtp.Auth

		// Check if OAuth authentication should be used
		if oauth != nil {
			auth = &XOAuth2SMTPAuth{
				Username: cfg.Username,
				Token:    oauth.AccessToken,
			}
		} else {
			// Use traditional authentication methods
			switch cfg.AuthProtocol {
			case "cram":
				auth = smtp.CRAMMD5Auth(cfg.Username, cfg.Password)
			case "plain":
				auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
			case "login":
				auth = &smtppool.LoginAuth{Username: cfg.Username, Password: cfg.Password}
			case "", "none":
				// No authentication
			default:
				return nil, fmt.Errorf("unknown SMTP auth type '%s'", cfg.AuthProtocol)
			}
		}
		cfg.Auth = auth

		// TLS config
		if cfg.TLSType != "none" {
			cfg.TLSConfig = &tls.Config{}
			if cfg.TLSSkipVerify {
				cfg.TLSConfig.InsecureSkipVerify = cfg.TLSSkipVerify
			} else {
				cfg.TLSConfig.ServerName = cfg.Host
			}

			// SSL/TLS, not STARTTLS
			if cfg.TLSType == "tls" {
				cfg.SSL = true
			}
		}

		// Parse timeouts.
		idleTimeout, err := time.ParseDuration(cfg.IdleTimeout)
		if err != nil {
			idleTimeout = 30 * time.Second
		}
		poolWaitTimeout, err := time.ParseDuration(cfg.PoolWaitTimeout)
		if err != nil {
			poolWaitTimeout = 40 * time.Second
		}

		pool, err := smtppool.New(smtppool.Opt{
			Host:              cfg.Host,
			Port:              cfg.Port,
			HelloHostname:     cfg.HelloHostname,
			MaxConns:          cfg.MaxConns,
			MaxMessageRetries: cfg.MaxMessageRetries,
			IdleTimeout:       idleTimeout,
			PoolWaitTimeout:   poolWaitTimeout,
			SSL:               cfg.SSL,
			Auth:              cfg.Auth,
			TLSConfig:         cfg.TLSConfig,
		})
		if err != nil {
			return nil, err
		}
		pools = append(pools, pool)
	}

	return pools, nil
}

// Send sends an email using one of the configured SMTP servers.
func (e *Email) Send(m models.Message) error {
	// Refresh OAuth token if needed
	oauthConfig, tokensRefreshed, err := e.refreshOAuthIfNeeded()
	if err != nil {
		return err
	}

	// If tokens were refreshed, recreate SMTP pools
	if tokensRefreshed {
		// Close existing pools
		for _, p := range e.smtpPools {
			p.Close()
		}

		// Create new pools with refreshed tokens
		newPools, err := NewSmtpPool(e.smtpCfg, oauthConfig)
		if err != nil {
			e.lo.Error("Failed to recreate SMTP pools after token refresh", "inbox_id", e.Identifier(), "error", err)
			return fmt.Errorf("failed to recreate SMTP pools: %w", err)
		}
		e.smtpPools = newPools
	}

	// Select a random SMTP server if there are multiple
	var (
		serverCount = len(e.smtpPools)
		server      *smtppool.Pool
	)
	if serverCount > 1 {
		server = e.smtpPools[rand.Intn(serverCount)]
	} else {
		server = e.smtpPools[0]
	}

	// Prepare attachments if there are any
	var attachments []smtppool.Attachment
	if m.Attachments != nil {
		attachments = make([]smtppool.Attachment, 0, len(m.Attachments))
		for _, file := range m.Attachments {
			attachment := smtppool.Attachment{
				Filename: file.Name,
				Header:   file.Header,
				Content:  make([]byte, len(file.Content)),
			}
			copy(attachment.Content, file.Content)
			attachments = append(attachments, attachment)
		}
	}

	email := smtppool.Email{
		From:        m.From,
		To:          m.To,
		Cc:          m.CC,
		Bcc:         m.BCC,
		Subject:     m.Subject,
		Attachments: attachments,
		Headers:     textproto.MIMEHeader{},
	}

	// Set libredesk loop prevention header to from address.
	emailAddress, err := stringutil.ExtractEmail(m.From)
	if err != nil {
		e.lo.Error("Failed to extract email address from the 'From' header", "error", err)
		return fmt.Errorf("failed to extract email address from 'From' header: %w", err)
	}
	email.Headers.Set(headerLibredeskLoopPrevention, emailAddress)

	// Attach SMTP level headers
	for key, value := range e.headers {
		email.Headers.Set(key, value)
	}

	// Attach email level headers
	for key, value := range m.Headers {
		email.Headers.Set(key, value[0])
	}

	// Set In-Reply-To header
	if m.InReplyTo != "" {
		email.Headers.Set(headerInReplyTo, "<"+m.InReplyTo+">")
		e.lo.Debug("In-Reply-To header set", "message_id", m.InReplyTo)
	}

	// Set message id header
	if m.SourceID.String != "" {
		email.Headers.Set(headerMessageID, fmt.Sprintf("<%s>", m.SourceID.String))
		e.lo.Debug("Message-ID header set", "message_id", m.SourceID.String)
	}

	// Set references header
	var references string
	for _, ref := range m.References {
		references += "<" + ref + "> "
	}
	e.lo.Debug("References header set", "references", references)
	email.Headers.Set(headerReferences, references)

	// Set email content
	switch m.ContentType {
	case "plain":
		email.Text = []byte(m.Content)
	default:
		email.HTML = []byte(m.Content)
		if len(m.AltContent) > 0 {
			email.Text = []byte(m.AltContent)
		}
	}
	return server.Send(email)
}
