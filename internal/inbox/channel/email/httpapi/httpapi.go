// Package httpapi implements sending and receiving email through HTTPS transactional
// email APIs (e.g. Resend) as an alternative to direct SMTP/IMAP connections.
package httpapi

import (
	"context"
	"fmt"
	"net/http"

	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
)

// Attachment is a file attached to an outbound message.
type Attachment struct {
	Filename    string
	ContentType string
	Content     []byte
}

// OutboundEmail is the normalized shape passed to a Provider for sending.
type OutboundEmail struct {
	From    string
	To      []string
	CC      []string
	BCC     []string
	ReplyTo string
	Subject string
	HTML    string
	Text    string
	// Headers carries additional RFC headers the provider should attach to the outgoing
	// message where supported (Message-ID, In-Reply-To, References, loop-prevention, etc).
	Headers     map[string]string
	Attachments []Attachment
}

// InboundAttachment is a file extracted from an inbound webhook payload.
type InboundAttachment struct {
	Filename    string
	ContentType string
	ContentID   string
	Content     []byte
}

// InboundEmail is the normalized shape a Provider produces from an inbound webhook.
//
// When RawMessage is set it is the full RFC822 message and the caller parses it directly
// (this is what the Resend provider returns, since its webhook only carries metadata and the
// full message is fetched from Resend's API). The structured fields are used by providers
// that deliver parsed content in the webhook itself. MessageID and From are always populated
// where available so the caller can dedupe / drop blocked senders before parsing the body.
type InboundEmail struct {
	RawMessage []byte

	MessageID string
	From      string
	// Recipients is the envelope recipient list (RCPT TO). It's the authoritative address for
	// plus-address conversation routing, since the MIME To: header can be rewritten in transit.
	Recipients []string

	To          []string
	CC          []string
	BCC         []string
	Subject     string
	HTML        string
	Text        string
	InReplyTo   string
	References  []string
	Attachments []InboundAttachment
}

// Provider sends outbound mail through, and parses inbound webhook payloads from, a
// specific HTTPS transactional email API.
type Provider interface {
	// Name returns the provider identifier (matches imodels.HTTPAPIConfig.Provider).
	Name() string

	// Send delivers msg through the provider's API and returns the provider-assigned message ID.
	Send(ctx context.Context, msg OutboundEmail) (messageID string, err error)

	// VerifyAndParseWebhook authenticates an inbound webhook request using the provider's
	// signature scheme and returns a normalized InboundEmail, fetching the full message from
	// the provider's API when the webhook only carries metadata.
	VerifyAndParseWebhook(ctx context.Context, headers http.Header, body []byte) (InboundEmail, error)
}

// New returns a Provider for the given HTTP API config.
func New(cfg imodels.HTTPAPIConfig) (Provider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("http api key is required")
	}
	switch cfg.Provider {
	case imodels.HTTPAPIProviderResend:
		return newResendProvider(cfg), nil
	case "":
		return nil, fmt.Errorf("http api provider is required")
	default:
		return nil, fmt.Errorf("unsupported http api provider %q", cfg.Provider)
	}
}
