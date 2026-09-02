// Package whatsapp implements a WhatsApp Cloud API inbox.
package whatsapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/attachment"
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/inbox"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/abhinavxd/libredesk/internal/whatsapp"
	"github.com/zerodha/logf"
)

const ChannelWhatsApp = "whatsapp"

const MetaCallTimeout = 30 * time.Second

const (
	sendMaxAttempts  = 3
	sendRetryBackoff = 2 * time.Second
)

// Meta's published per-media-type upload size caps.
const (
	maxImageBytes    = 5 * 1024 * 1024
	maxVideoBytes    = 16 * 1024 * 1024
	maxAudioBytes    = 16 * 1024 * 1024
	maxDocumentBytes = 100 * 1024 * 1024
)

var supportedMediaMIMETypes = map[string]struct{}{
	"audio/aac":                     {},
	"audio/mp4":                     {},
	"audio/mpeg":                    {},
	"audio/amr":                     {},
	"audio/ogg":                     {},
	"audio/opus":                    {},
	"application/vnd.ms-powerpoint": {},
	"application/msword":            {},
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document":   {},
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": {},
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         {},
	"application/pdf":          {},
	"text/plain":               {},
	"application/vnd.ms-excel": {},
	"image/jpeg":               {},
	"image/png":                {},
	"video/mp4":                {},
	"video/3gpp":               {},
	"video/3gp":                {},
}

// Config is the per-inbox WhatsApp configuration from the inbox config JSONB, with tokens already decrypted.
type Config struct {
	PhoneNumberID      string `json:"phone_number_id"`
	WABAID             string `json:"waba_id"`
	AccessToken        string `json:"access_token"`
	AppSecret          string `json:"app_secret"`
	WebhookVerifyToken string `json:"webhook_verify_token"`
	APIVersion         string `json:"api_version"`

	CSATTemplateLanguage   string `json:"csat_template_language"`
	CSATTemplateBody       string `json:"csat_template_body"`
	CSATTemplateButtonText string `json:"csat_template_button_text"`
}

func (c Config) Account() whatsapp.Account {
	return whatsapp.Account{
		PhoneNumberID: c.PhoneNumberID,
		WABAID:        c.WABAID,
		AccessToken:   c.AccessToken,
		AppSecret:     c.AppSecret,
		APIVersion:    c.APIVersion,
	}
}

// SendMeta is the per-message metadata threaded through OutboundMessage.Meta; a set TemplateName means a template send.
type SendMeta struct {
	ToPhone               string                    `json:"to_phone"`
	ReplyToWAMessageID    string                    `json:"reply_to_wa_message_id,omitempty"`
	TemplateName          string                    `json:"template_name,omitempty"`
	TemplateLanguage      string                    `json:"template_language,omitempty"`
	TemplateParams        map[string]string         `json:"template_params,omitempty"`
	TemplateHeaderType    string                    `json:"template_header_type,omitempty"`
	TemplateHeaderContent string                    `json:"template_header_content,omitempty"`
	TemplateBodyContent   string                    `json:"template_body_content,omitempty"`
	TemplateButtons       []whatsapp.TemplateButton `json:"template_buttons,omitempty"`
}

// SourceIDUpdater persists the Meta message ID for status correlation.
type SourceIDUpdater interface {
	UpdateMessageSourceID(messageUUID, sourceID string) error
}

type WhatsApp struct {
	id            int
	name          string
	config        Config
	client        *whatsapp.Client
	lo            *logf.Logger
	messageStore  inbox.MessageStore
	sourceUpdater SourceIDUpdater
	retryBackoff  time.Duration
}

type Opts struct {
	ID            int
	Name          string
	Config        Config
	Client        *whatsapp.Client
	Lo            *logf.Logger
	SourceUpdater SourceIDUpdater
}

func New(store inbox.MessageStore, opts Opts) (*WhatsApp, error) {
	if opts.Client == nil {
		return nil, fmt.Errorf("whatsapp client is required")
	}
	if opts.Config.PhoneNumberID == "" || opts.Config.AccessToken == "" {
		return nil, fmt.Errorf("phone_number_id and access_token are required")
	}
	if opts.Lo == nil {
		return nil, fmt.Errorf("logger is required")
	}
	return &WhatsApp{
		id:            opts.ID,
		name:          opts.Name,
		config:        opts.Config,
		client:        opts.Client,
		lo:            opts.Lo,
		messageStore:  store,
		sourceUpdater: opts.SourceUpdater,
		retryBackoff:  sendRetryBackoff,
	}, nil
}

func (w *WhatsApp) Identifier() int          { return w.id }
func (w *WhatsApp) Config() Config           { return w.config }
func (w *WhatsApp) Channel() string          { return ChannelWhatsApp }
func (w *WhatsApp) Name() string             { return w.name }
func (w *WhatsApp) FromAddress() string      { return "" }
func (w *WhatsApp) ReplyToAddress() string   { return "" }
func (w *WhatsApp) FromNameTemplate() string { return "" }
func (w *WhatsApp) Close() error             { return nil }

// Receive is a no-op; inbound messages arrive via the webhook handler.
func (w *WhatsApp) Receive(ctx context.Context) error { return nil }

// Retries a 429 or 5xx only, since a send carries no idempotency key and Meta may already have accepted it.
func (w *WhatsApp) Send(message models.OutboundMessage) error {
	var err error
	for attempt := 1; ; attempt++ {
		err = w.send(message)
		if err == nil || attempt >= sendMaxAttempts || !metaRefusedSend(err) {
			return err
		}
		w.lo.Warn("meta refused the whatsapp send, retrying", "message_uuid", message.UUID, "attempt", attempt, "error", err)
		time.Sleep(w.retryBackoff * time.Duration(attempt))
	}
}

func metaRefusedSend(err error) bool {
	var me *whatsapp.MetaAPIError
	if !errors.As(err, &me) {
		return false
	}
	return me.StatusCode == http.StatusTooManyRequests || me.StatusCode >= http.StatusInternalServerError
}

func (w *WhatsApp) send(message models.OutboundMessage) error {
	meta, err := parseSendMeta(message.Meta)
	if err != nil {
		return fmt.Errorf("parsing whatsapp send meta: %w", err)
	}
	if meta.ToPhone == "" {
		return fmt.Errorf("missing recipient phone number on outbound message")
	}

	// An attachment costs two calls: the media upload and the send.
	timeout := MetaCallTimeout
	if len(message.Attachments) > 0 {
		timeout = 2 * MetaCallTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	acc := w.config.Account()
	var sourceID string

	switch {
	case meta.TemplateName != "":
		components := whatsapp.BuildSendComponents(whatsapp.TemplateSendParts{
			HeaderType:    meta.TemplateHeaderType,
			HeaderContent: meta.TemplateHeaderContent,
			BodyContent:   meta.TemplateBodyContent,
			Buttons:       meta.TemplateButtons,
			Params:        meta.TemplateParams,
		})
		sourceID, err = w.client.SendTemplate(ctx, acc, meta.ToPhone, meta.TemplateName, meta.TemplateLanguage, components)

	case len(message.Attachments) > 0:
		sourceID, err = w.sendAttachment(ctx, acc, meta, message)

	case strings.TrimSpace(textBody(message)) != "":
		sourceID, err = w.client.SendText(ctx, acc, meta.ToPhone, textBody(message), meta.ReplyToWAMessageID)

	default:
		return fmt.Errorf("outbound message has no content")
	}

	if sourceID != "" && w.sourceUpdater != nil {
		if upErr := w.sourceUpdater.UpdateMessageSourceID(message.UUID, sourceID); upErr != nil {
			w.lo.Error("failed to persist whatsapp source id", "message_uuid", message.UUID, "source_id", sourceID, "error", upErr)
		}
	}
	return err
}

// sendAttachment uploads and sends one attachment; Meta accepts only one media per message.
func (w *WhatsApp) sendAttachment(ctx context.Context, acc whatsapp.Account, meta SendMeta, message models.OutboundMessage) (string, error) {
	if len(message.Attachments) > 1 {
		return "", fmt.Errorf("whatsapp accepts one attachment per message, got %d", len(message.Attachments))
	}
	if bad := rejectedAttachments(message.Attachments); len(bad) > 0 {
		return "", fmt.Errorf("WhatsApp can't send these files: %s", strings.Join(bad, "; "))
	}

	att := message.Attachments[0]
	mediaID, err := w.client.UploadMedia(ctx, acc, att.Content, att.ContentType, att.Name)
	if err != nil {
		return "", fmt.Errorf("uploading attachment to meta: %w", err)
	}
	return w.client.SendMedia(ctx, acc, meta.ToPhone, mediaTypeForAttachment(att), mediaID, strings.TrimSpace(textBody(message)), att.Name, meta.ReplyToWAMessageID)
}

// SupportsCaption reports whether media of this content type can carry a caption; audio can't.
func SupportsCaption(contentType string) bool {
	return mediaTypeForAttachment(attachment.Attachment{ContentType: contentType}) != "audio"
}

// RejectMediaReason returns why WhatsApp won't accept the file, or an empty string when it will.
func RejectMediaReason(name, contentType string, size int) string {
	reasons := rejectedAttachments([]attachment.Attachment{{Name: name, ContentType: contentType, Size: size}})
	if len(reasons) == 0 {
		return ""
	}
	return reasons[0]
}

func parseSendMeta(raw json.RawMessage) (SendMeta, error) {
	var meta SendMeta
	if len(raw) == 0 {
		return meta, nil
	}
	// SendMeta lives under a "whatsapp" key in message.meta to avoid colliding with email's to/cc keys.
	var envelope struct {
		WhatsApp json.RawMessage `json:"whatsapp"`
	}
	if err := json.Unmarshal(raw, &envelope); err == nil && len(envelope.WhatsApp) > 0 {
		if err := json.Unmarshal(envelope.WhatsApp, &meta); err != nil {
			return meta, fmt.Errorf("decoding whatsapp meta envelope: %w", err)
		}
		return meta, nil
	}
	if err := json.Unmarshal(raw, &meta); err != nil {
		return meta, err
	}
	return meta, nil
}

// textBody returns the plain-text body; raw HTML must never reach WhatsApp verbatim.
func textBody(m models.OutboundMessage) string {
	if m.TextContent != "" {
		return m.TextContent
	}
	if m.ContentType == models.ContentTypeHTML {
		return stringutil.HTML2Text(m.Content)
	}
	return m.Content
}

func rejectedAttachments(atts []attachment.Attachment) []string {
	var reasons []string
	for _, att := range atts {
		mime := normalizeMIME(att.ContentType)
		if mime == "image/webp" {
			reasons = append(reasons, fmt.Sprintf("%s (WebP images aren't supported; convert to JPEG or PNG)", att.Name))
			continue
		}
		if _, ok := supportedMediaMIMETypes[mime]; !ok {
			reasons = append(reasons, fmt.Sprintf("%s (unsupported type %s)", att.Name, att.ContentType))
			continue
		}
		mediaType := mediaTypeForAttachment(att)
		if max := maxMediaBytes(mediaType); att.Size > max {
			reasons = append(reasons, fmt.Sprintf("%s (%s exceeds the %s %s limit)", att.Name, humanBytes(att.Size), humanBytes(max), mediaType))
		}
	}
	return reasons
}

func maxMediaBytes(mediaType string) int {
	switch mediaType {
	case "image":
		return effectiveLimit(maxImageBytes)
	case "video":
		return effectiveLimit(maxVideoBytes)
	case "audio":
		return effectiveLimit(maxAudioBytes)
	}
	return effectiveLimit(maxDocumentBytes)
}

func humanBytes(n int) string {
	switch {
	case n >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
	case n >= 1024:
		return fmt.Sprintf("%.0f KB", float64(n)/1024)
	}
	return fmt.Sprintf("%d B", n)
}

func normalizeMIME(contentType string) string {
	mime := strings.ToLower(strings.TrimSpace(contentType))
	if i := strings.Index(mime, ";"); i >= 0 {
		mime = strings.TrimSpace(mime[:i])
	}
	return mime
}

func mediaTypeForAttachment(att attachment.Attachment) string {
	switch normalizeMIME(att.ContentType) {
	case "image/jpeg", "image/png":
		return "image"
	case "video/mp4", "video/3gpp", "video/3gp":
		return "video"
	case "audio/aac", "audio/mp4", "audio/mpeg", "audio/amr", "audio/ogg", "audio/opus":
		return "audio"
	}
	return "document"
}

// effectiveLimit keeps 2% headroom below Meta's hard caps to absorb size-measurement skew at the boundary.
func effectiveLimit(n int) int {
	return n * 98 / 100
}
