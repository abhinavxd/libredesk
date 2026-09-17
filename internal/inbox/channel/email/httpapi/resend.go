package httpapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
)

const resendAPIBaseURL = "https://api.resend.com"

// resendEventInboundEmail is the webhook event type Resend sends for a received inbound email.
const resendEventInboundEmail = "email.received"

// maxInboundMessageBytes caps the size of a downloaded raw inbound message.
const maxInboundMessageBytes = 40 << 20 // 40 MiB

// resendHTTPClient is shared across all Resend provider instances; Resend calls are infrequent
// relative to SMTP so a single pooled client is sufficient. Redirects are constrained to HTTPS
// so a downgrade can't leak the API key or a raw message over cleartext.
var resendHTTPClient = &http.Client{
	Timeout:       30 * time.Second,
	CheckRedirect: httpsOnlyRedirect,
}

// httpsOnlyRedirect refuses any redirect hop that isn't HTTPS and caps the redirect chain
// (setting CheckRedirect replaces net/http's default 10-redirect limit).
func httpsOnlyRedirect(req *http.Request, via []*http.Request) error {
	if req.URL.Scheme != "https" {
		return fmt.Errorf("refusing non-https redirect (scheme %q)", req.URL.Scheme)
	}
	if len(via) >= 10 {
		return fmt.Errorf("stopped after 10 redirects")
	}
	return nil
}

// resendProvider implements Provider for the Resend (https://resend.com) transactional email API.
//
// Outbound uses POST /emails. Inbound uses the email.received webhook (metadata only) plus
// GET /emails/receiving/{id} to fetch the full RFC822 message via the signed raw.download_url
// it returns; that message is then run through the same enmime pipeline as the IMAP path.
type resendProvider struct {
	apiKey        string
	webhookSecret string
	baseURL       string       // overridable in tests; defaults to resendAPIBaseURL
	client        *http.Client // overridable in tests; defaults to resendHTTPClient
}

func newResendProvider(cfg imodels.HTTPAPIConfig) *resendProvider {
	return &resendProvider{
		apiKey:        cfg.APIKey,
		webhookSecret: cfg.WebhookSecret,
		baseURL:       resendAPIBaseURL,
		client:        resendHTTPClient,
	}
}

func (p *resendProvider) Name() string {
	return imodels.HTTPAPIProviderResend
}

type resendAttachment struct {
	Filename    string `json:"filename"`
	Content     string `json:"content"` // base64-encoded
	ContentType string `json:"content_type,omitempty"`
}

type resendSendRequest struct {
	From        string             `json:"from"`
	To          []string           `json:"to,omitempty"`
	CC          []string           `json:"cc,omitempty"`
	BCC         []string           `json:"bcc,omitempty"`
	ReplyTo     string             `json:"reply_to,omitempty"`
	Subject     string             `json:"subject"`
	HTML        string             `json:"html,omitempty"`
	Text        string             `json:"text,omitempty"`
	Headers     map[string]string  `json:"headers,omitempty"`
	Attachments []resendAttachment `json:"attachments,omitempty"`
}

type resendSendResponse struct {
	ID string `json:"id"`
}

type resendErrorResponse struct {
	Name       string `json:"name"`
	Message    string `json:"message"`
	StatusCode int    `json:"statusCode"`
}

// Send delivers msg via POST https://api.resend.com/emails.
func (p *resendProvider) Send(ctx context.Context, msg OutboundEmail) (string, error) {
	reqBody := resendSendRequest{
		From:    msg.From,
		To:      msg.To,
		CC:      msg.CC,
		BCC:     msg.BCC,
		ReplyTo: msg.ReplyTo,
		Subject: msg.Subject,
		HTML:    msg.HTML,
		Text:    msg.Text,
		Headers: msg.Headers,
	}
	for _, att := range msg.Attachments {
		reqBody.Attachments = append(reqBody.Attachments, resendAttachment{
			Filename:    att.Filename,
			Content:     base64.StdEncoding.EncodeToString(att.Content),
			ContentType: att.ContentType,
		})
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshalling resend request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/emails", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("building resend request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("calling resend api: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading resend response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr resendErrorResponse
		if jsonErr := json.Unmarshal(respBody, &apiErr); jsonErr == nil && apiErr.Message != "" {
			return "", fmt.Errorf("resend api error (%d): %s: %s", resp.StatusCode, apiErr.Name, apiErr.Message)
		}
		return "", fmt.Errorf("resend api error (%d): %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var sendResp resendSendResponse
	if err := json.Unmarshal(respBody, &sendResp); err != nil {
		return "", fmt.Errorf("parsing resend response: %w", err)
	}
	return sendResp.ID, nil
}

// resendInboundEnvelope is the webhook event envelope shared across Resend event types.
// The email.received payload carries metadata only - the body, headers and attachment bytes
// are fetched separately from the Receiving API using email_id.
type resendInboundEnvelope struct {
	Type string `json:"type"`
	Data struct {
		EmailID     string   `json:"email_id"`
		MessageID   string   `json:"message_id"`
		From        string   `json:"from"`
		ReceivedFor []string `json:"received_for"` // envelope recipients (authoritative for plus-address routing)
	} `json:"data"`
}

// resendReceivedEmail is the GET /emails/receiving/{id} response. Only the raw download URL
// is used; the full message is parsed from the downloaded RFC822 bytes.
type resendReceivedEmail struct {
	Raw struct {
		DownloadURL string `json:"download_url"`
	} `json:"raw"`
}

// parseResendInboundEnvelope unmarshals a webhook body and rejects any event that isn't an
// inbound email. Resend delivers every event type (email.sent, email.delivered, ...) to the
// same endpoint, so this guards against a delivery event being turned into a fake message.
func parseResendInboundEnvelope(body []byte) (resendInboundEnvelope, error) {
	var env resendInboundEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return env, fmt.Errorf("parsing resend webhook payload: %w", err)
	}
	if env.Type != resendEventInboundEmail {
		return env, fmt.Errorf("ignoring resend webhook event %q", env.Type)
	}
	if env.Data.EmailID == "" {
		return env, fmt.Errorf("resend webhook missing email_id")
	}
	return env, nil
}

// VerifyAndParseWebhook verifies the Svix-style signature Resend attaches to webhook requests
// and returns the raw RFC822 message (fetched from the Receiving API) plus the metadata the
// caller needs to dedupe and drop blocked senders.
func (p *resendProvider) VerifyAndParseWebhook(ctx context.Context, headers http.Header, body []byte) (InboundEmail, error) {
	if p.webhookSecret == "" {
		return InboundEmail{}, fmt.Errorf("webhook secret not configured")
	}
	if err := verifySvixSignature(p.webhookSecret, headers, body); err != nil {
		return InboundEmail{}, fmt.Errorf("verifying webhook signature: %w", err)
	}

	env, err := parseResendInboundEnvelope(body)
	if err != nil {
		return InboundEmail{}, err
	}

	raw, err := p.fetchRawMessage(ctx, env.Data.EmailID)
	if err != nil {
		return InboundEmail{}, err
	}

	return InboundEmail{
		RawMessage: raw,
		MessageID:  strings.Trim(env.Data.MessageID, "<>"),
		From:       env.Data.From,
		Recipients: env.Data.ReceivedFor,
	}, nil
}

// fetchRawMessage resolves the raw message download URL for a received email and downloads it.
func (p *resendProvider) fetchRawMessage(ctx context.Context, emailID string) ([]byte, error) {
	var received resendReceivedEmail
	if err := p.apiGetJSON(ctx, "/emails/receiving/"+url.PathEscape(emailID), &received); err != nil {
		return nil, fmt.Errorf("fetching received email %s: %w", emailID, err)
	}
	if received.Raw.DownloadURL == "" {
		return nil, fmt.Errorf("received email %s has no raw download url", emailID)
	}
	return p.downloadRaw(ctx, received.Raw.DownloadURL)
}

// apiGetJSON performs an authenticated GET against the Resend API and decodes the JSON body.
func (p *resendProvider) apiGetJSON(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("resend api %s returned %d: %s", path, resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return json.Unmarshal(b, out)
}

// downloadRaw fetches the raw message from a provider-issued signed URL.
func (p *resendProvider) downloadRaw(ctx context.Context, rawURL string) ([]byte, error) {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme != "https" {
		return nil, fmt.Errorf("invalid raw message download url")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading raw message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("downloading raw message returned %d", resp.StatusCode)
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxInboundMessageBytes+1))
	if err != nil {
		return nil, fmt.Errorf("reading raw message: %w", err)
	}
	if len(raw) > maxInboundMessageBytes {
		return nil, fmt.Errorf("raw message exceeds %d bytes", maxInboundMessageBytes)
	}
	return raw, nil
}
