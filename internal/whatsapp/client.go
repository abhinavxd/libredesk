package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
	"time"

	"github.com/zerodha/logf"
)

const (
	defaultGraphURL = "https://graph.facebook.com"
	defaultTimeout  = 30 * time.Second
	// maxMediaDownloadBytes matches Meta's largest media cap (100MB documents).
	maxMediaDownloadBytes = 100 * 1024 * 1024
)

// Client is a thin wrapper around Meta's Graph API for WhatsApp Cloud.
type Client struct {
	httpClient  *http.Client
	lo          *logf.Logger
	baseURL     string
	onAuthError func(acc Account)
}

// SetAuthErrorHook registers a callback fired whenever Meta rejects the account's token.
func (c *Client) SetAuthErrorHook(fn func(acc Account)) { c.onAuthError = fn }

func (c *Client) notifyAuthError(acc Account, err error) {
	var me *MetaAPIError
	if c.onAuthError != nil && errors.As(err, &me) && (me.StatusCode == http.StatusUnauthorized || me.Code == 190) {
		c.onAuthError(acc)
	}
}

// New returns a Client with default timeouts.
func New(lo *logf.Logger) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: defaultTimeout},
		lo:         lo,
		baseURL:    defaultGraphURL,
	}
}

// NewWithHTTPClient lets callers supply their own http.Client (for tests or SSRF guard).
func NewWithHTTPClient(lo *logf.Logger, httpClient *http.Client) *Client {
	c := New(lo)
	if httpClient != nil {
		c.httpClient = httpClient
	}
	return c
}

// SetBaseURL overrides the Graph API base URL (used by tests).
func (c *Client) SetBaseURL(u string) { c.baseURL = strings.TrimRight(u, "/") }

// ValidateCredentials hits Meta to confirm the token + IDs are valid.
func (c *Client) ValidateCredentials(ctx context.Context, acc Account) error {
	endpoint := fmt.Sprintf("%s/%s/%s", c.baseURL, acc.Version(), acc.PhoneNumberID)
	if _, err := c.doRequest(ctx, http.MethodGet, endpoint, nil, acc); err != nil {
		return err
	}
	endpoint = fmt.Sprintf("%s/%s/%s/phone_numbers", c.baseURL, acc.Version(), acc.WABAID)
	if _, err := c.doRequest(ctx, http.MethodGet, endpoint, nil, acc); err != nil {
		return err
	}
	return nil
}

// SendText sends a plain text message and returns the Meta message ID.
func (c *Client) SendText(ctx context.Context, acc Account, toPhone, body, replyToID string) (string, error) {
	payload := map[string]any{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                toPhone,
		"type":              "text",
		"text":              map[string]any{"body": body, "preview_url": false},
	}
	if replyToID != "" {
		payload["context"] = map[string]string{"message_id": replyToID}
	}
	return c.sendMessage(ctx, acc, payload)
}

// SendMedia sends a media message; mediaType is one of image, video, audio, document, sticker.
func (c *Client) SendMedia(ctx context.Context, acc Account, toPhone, mediaType, mediaID, caption, filename, replyToID string) (string, error) {
	media := map[string]any{"id": mediaID}
	if caption != "" && (mediaType == "image" || mediaType == "video" || mediaType == "document") {
		media["caption"] = caption
	}
	if filename != "" && mediaType == "document" {
		media["filename"] = filename
	}
	payload := map[string]any{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                toPhone,
		"type":              mediaType,
		mediaType:           media,
	}
	if replyToID != "" {
		payload["context"] = map[string]string{"message_id": replyToID}
	}
	return c.sendMessage(ctx, acc, payload)
}

// SendTemplate sends an approved template message with components from BuildSendComponents.
func (c *Client) SendTemplate(ctx context.Context, acc Account, toPhone, name, language string, components []map[string]any) (string, error) {
	tmpl := map[string]any{
		"name":     name,
		"language": map[string]string{"code": language},
	}
	if len(components) > 0 {
		tmpl["components"] = components
	}
	payload := map[string]any{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                toPhone,
		"type":              "template",
		"template":          tmpl,
	}
	return c.sendMessage(ctx, acc, payload)
}

// SubscribeWebhook subscribes the app to the WABA and points its webhook at callbackURL, which Meta verifies with a GET handshake so it must be publicly reachable.
func (c *Client) SubscribeWebhook(ctx context.Context, acc Account, callbackURL, verifyToken string) error {
	endpoint := fmt.Sprintf("%s/%s/%s/subscribed_apps", c.baseURL, acc.Version(), acc.WABAID)
	if _, err := c.doRequest(ctx, http.MethodPost, endpoint, nil, acc); err != nil {
		return fmt.Errorf("subscribing app to waba: %w", err)
	}
	payload := map[string]any{
		"override_callback_uri": callbackURL,
		"verify_token":          verifyToken,
	}
	if _, err := c.doRequest(ctx, http.MethodPost, endpoint, payload, acc); err != nil {
		return fmt.Errorf("overriding waba callback: %w", err)
	}
	return nil
}

// MarkRead marks an inbound message as read on the Meta side.
func (c *Client) MarkRead(ctx context.Context, acc Account, messageID string) error {
	payload := map[string]any{
		"messaging_product": "whatsapp",
		"status":            "read",
		"message_id":        messageID,
	}
	endpoint := fmt.Sprintf("%s/%s/%s/messages", c.baseURL, acc.Version(), acc.PhoneNumberID)
	_, err := c.doRequest(ctx, http.MethodPost, endpoint, payload, acc)
	return err
}

// GetMediaURL fetches the temporary download URL + metadata for a media ID.
func (c *Client) GetMediaURL(ctx context.Context, acc Account, mediaID string) (MediaInfo, error) {
	endpoint := fmt.Sprintf("%s/%s/%s", c.baseURL, acc.Version(), mediaID)
	body, err := c.doRequest(ctx, http.MethodGet, endpoint, nil, acc)
	if err != nil {
		return MediaInfo{}, err
	}
	var info MediaInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return MediaInfo{}, fmt.Errorf("decoding media info: %w", err)
	}
	return info, nil
}

// DownloadMedia fetches a media object's bytes from a GetMediaURL-issued CDN URL.
func (c *Client) DownloadMedia(ctx context.Context, acc Account, mediaURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mediaURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building media request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+acc.AccessToken)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading media: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		metaErr := parseMetaError(resp.StatusCode, respBody)
		c.notifyAuthError(acc, metaErr)
		return nil, metaErr
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxMediaDownloadBytes+1))
	if err != nil {
		return nil, fmt.Errorf("reading media body: %w", err)
	}
	if len(body) > maxMediaDownloadBytes {
		return nil, fmt.Errorf("media exceeds %d bytes", maxMediaDownloadBytes)
	}
	return body, nil
}

// UploadMedia uploads bytes to Meta and returns the resulting media ID.
func (c *Client) UploadMedia(ctx context.Context, acc Account, content []byte, contentType, filename string) (string, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if err := mw.WriteField("messaging_product", "whatsapp"); err != nil {
		return "", err
	}
	if err := mw.WriteField("type", contentType); err != nil {
		return "", err
	}
	// Meta validates the file part's own Content-Type, which CreateFormFile hardcodes to octet-stream.
	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename=%q`, filename))
	partHeader.Set("Content-Type", contentType)
	fw, err := mw.CreatePart(partHeader)
	if err != nil {
		return "", err
	}
	if _, err := fw.Write(content); err != nil {
		return "", err
	}
	if err := mw.Close(); err != nil {
		return "", err
	}

	endpoint := fmt.Sprintf("%s/%s/%s/media", c.baseURL, acc.Version(), acc.PhoneNumberID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+acc.AccessToken)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("uploading media: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		metaErr := parseMetaError(resp.StatusCode, respBody)
		c.notifyAuthError(acc, metaErr)
		return "", metaErr
	}
	var out UploadMediaResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return "", fmt.Errorf("decoding upload response: %w", err)
	}
	return out.ID, nil
}

// FetchTemplates lists templates from a WABA, walking pagination until exhausted.
func (c *Client) FetchTemplates(ctx context.Context, acc Account) ([]MetaTemplate, error) {
	endpoint := fmt.Sprintf("%s/%s/%s/message_templates?limit=100", c.baseURL, acc.Version(), acc.WABAID)
	var out []MetaTemplate
	for endpoint != "" {
		body, err := c.doRequest(ctx, http.MethodGet, endpoint, nil, acc)
		if err != nil {
			return nil, err
		}
		var page templateListResponse
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("decoding template list: %w", err)
		}
		out = append(out, page.Data...)
		endpoint = page.Paging.Next
	}
	return out, nil
}

// SubmitTemplate creates a new template. Returns Meta's template ID.
func (c *Client) SubmitTemplate(ctx context.Context, acc Account, t TemplateSubmission) (string, error) {
	endpoint := fmt.Sprintf("%s/%s/%s/message_templates", c.baseURL, acc.Version(), acc.WABAID)
	body, err := c.doRequest(ctx, http.MethodPost, endpoint, t, acc)
	if err != nil {
		return "", err
	}
	var resp struct {
		ID       string `json:"id"`
		Status   string `json:"status"`
		Category string `json:"category"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("decoding submit response: %w", err)
	}
	return resp.ID, nil
}

// DeleteTemplate removes a template by name (Meta deletes all language variants).
func (c *Client) DeleteTemplate(ctx context.Context, acc Account, name string) error {
	endpoint := fmt.Sprintf("%s/%s/%s/message_templates?name=%s", c.baseURL, acc.Version(), acc.WABAID, url.QueryEscape(name))
	_, err := c.doRequest(ctx, http.MethodDelete, endpoint, nil, acc)
	return err
}

func (c *Client) sendMessage(ctx context.Context, acc Account, payload any) (string, error) {
	endpoint := fmt.Sprintf("%s/%s/%s/messages", c.baseURL, acc.Version(), acc.PhoneNumberID)
	body, err := c.doRequest(ctx, http.MethodPost, endpoint, payload, acc)
	if err != nil {
		return "", err
	}
	var resp SendResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("decoding send response: %w", err)
	}
	if len(resp.Messages) == 0 {
		return "", fmt.Errorf("no message id in send response")
	}
	return resp.Messages[0].ID, nil
}

func (c *Client) doRequest(ctx context.Context, method, endpoint string, body any, acc Account) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encoding request body: %w", err)
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+acc.AccessToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling meta api: %w", err)
	}
	defer resp.Body.Close()
	respBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, fmt.Errorf("reading meta response: %w", readErr)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		metaErr := parseMetaError(resp.StatusCode, respBody)
		c.notifyAuthError(acc, metaErr)
		return respBody, metaErr
	}
	return respBody, nil
}

func parseMetaError(statusCode int, respBody []byte) error {
	var env metaErrorEnvelope
	if err := json.Unmarshal(respBody, &env); err != nil || env.Error.Message == "" {
		return fmt.Errorf("meta api returned status %d: %s", statusCode, string(respBody))
	}
	return &MetaAPIError{
		StatusCode: statusCode,
		Message:    env.Error.Message,
		Type:       env.Error.Type,
		Code:       env.Error.Code,
		Subcode:    env.Error.ErrorSubcode,
		UserMsg:    env.Error.ErrorUserMsg,
		FBTraceID:  env.Error.FBTraceID,
	}
}
