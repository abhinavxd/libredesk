package httpapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
)

const resendAPIBaseURL = "https://api.resend.com"

// resendClient is shared across all Resend provider instances; Resend calls are infrequent
// relative to SMTP so a single pooled client is sufficient.
var resendClient = &http.Client{Timeout: 20 * time.Second}

// resendProvider implements Provider for the Resend (https://resend.com) transactional email API.
//
// NOTE: field names for both the send request and the inbound webhook payload below are based
// on Resend's publicly documented API shape at the time this was written. Resend's inbound-email
// webhook feature is newer and more likely to have changed - verify payload field names against
// https://resend.com/docs before relying on inbound parsing in production, and adjust
// parseInboundPayload accordingly if they differ.
type resendProvider struct {
	apiKey        string
	webhookSecret string
	baseURL       string // overridable in tests; defaults to resendAPIBaseURL
}

func newResendProvider(cfg imodels.HTTPAPIConfig) *resendProvider {
	return &resendProvider{
		apiKey:        cfg.APIKey,
		webhookSecret: cfg.WebhookSecret,
		baseURL:       resendAPIBaseURL,
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

	resp, err := resendClient.Do(httpReq)
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

// resendInboundEnvelope wraps the resend webhook event envelope shared across event types.
type resendInboundEnvelope struct {
	Type string            `json:"type"`
	Data resendInboundData `json:"data"`
}

type resendInboundData struct {
	EmailID     string                    `json:"email_id"`
	MessageID   string                    `json:"message_id"`
	From        string                    `json:"from"`
	To          []string                  `json:"to"`
	CC          []string                  `json:"cc"`
	BCC         []string                  `json:"bcc"`
	Subject     string                    `json:"subject"`
	HTML        string                    `json:"html"`
	Text        string                    `json:"text"`
	Headers     json.RawMessage           `json:"headers"`
	Attachments []resendInboundAttachment `json:"attachments"`
}

type resendInboundAttachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	ContentID   string `json:"content_id"`
	Content     string `json:"content"` // base64-encoded
}

// VerifyAndParseWebhook verifies the Svix-style signature Resend attaches to webhook requests,
// then parses the inbound-email payload into a normalized InboundEmail.
func (p *resendProvider) VerifyAndParseWebhook(headers http.Header, body []byte) (InboundEmail, error) {
	if p.webhookSecret == "" {
		return InboundEmail{}, fmt.Errorf("webhook secret not configured")
	}
	if err := verifySvixSignature(p.webhookSecret, headers, body); err != nil {
		return InboundEmail{}, fmt.Errorf("verifying webhook signature: %w", err)
	}
	return parseResendInboundPayload(body)
}

func parseResendInboundPayload(body []byte) (InboundEmail, error) {
	var envelope resendInboundEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return InboundEmail{}, fmt.Errorf("parsing resend webhook payload: %w", err)
	}

	data := envelope.Data
	messageID := data.MessageID
	if messageID == "" {
		messageID = data.EmailID
	}

	headerMap := parseResendHeaders(data.Headers)

	inbound := InboundEmail{
		MessageID:  strings.Trim(messageID, "<>"),
		From:       data.From,
		To:         data.To,
		CC:         data.CC,
		BCC:        data.BCC,
		Subject:    data.Subject,
		HTML:       data.HTML,
		Text:       data.Text,
		InReplyTo:  strings.Trim(headerMap["In-Reply-To"], "<>"),
		References: splitReferences(headerMap["References"]),
	}
	for _, att := range data.Attachments {
		content, err := base64.StdEncoding.DecodeString(att.Content)
		if err != nil {
			continue
		}
		inbound.Attachments = append(inbound.Attachments, InboundAttachment{
			Filename:    att.Filename,
			ContentType: att.ContentType,
			ContentID:   att.ContentID,
			Content:     content,
		})
	}
	return inbound, nil
}

// parseResendHeaders accepts either a {"Header-Name": "value"} map or a
// [{"name": "Header-Name", "value": "value"}] array, since it's unconfirmed which shape
// Resend's inbound webhook uses for raw headers.
func parseResendHeaders(raw json.RawMessage) map[string]string {
	headers := map[string]string{}
	if len(raw) == 0 {
		return headers
	}

	var asMap map[string]string
	if err := json.Unmarshal(raw, &asMap); err == nil {
		return asMap
	}

	var asList []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}
	if err := json.Unmarshal(raw, &asList); err == nil {
		for _, h := range asList {
			headers[h.Name] = h.Value
		}
	}
	return headers
}

func splitReferences(references string) []string {
	if references == "" {
		return nil
	}
	fields := strings.Fields(references)
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		out = append(out, strings.Trim(f, " <>"))
	}
	return out
}
