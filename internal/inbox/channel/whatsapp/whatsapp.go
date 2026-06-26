// Package whatsapp implements a WhatsApp Cloud API inbox; inbound arrives via the webhook handler in cmd/, so Receive() is a no-op like livechat.
package whatsapp

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
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

// MetaCallTimeout caps a single Meta API call.
const MetaCallTimeout = 30 * time.Second

// Meta's per-media-type upload size caps; a static sticker over maxStickerBytes is sent as a document.
const (
	maxStickerBytes  = 100 * 1024
	maxImageBytes    = 5 * 1024 * 1024
	maxVideoBytes    = 16 * 1024 * 1024
	maxAudioBytes    = 16 * 1024 * 1024
	maxDocumentBytes = 100 * 1024 * 1024
)

// captionMarker tracks the standalone caption text send in the per-message sent-attachment set, alongside attachment UUIDs.
const captionMarker = "__caption__"

// CSAT template defaults applied when the inbox leaves these blank; the language code must match the body's language.
const (
	DefaultCSATTemplateLanguage   = "en_US"
	DefaultCSATTemplateBody       = "Your conversation has been resolved. How did we do? Tap below to rate your experience."
	DefaultCSATTemplateButtonText = "Rate us"
)

// supportedMediaMIMETypes is the set of MIME types Meta accepts for WhatsApp media upload.
var supportedMediaMIMETypes = map[string]struct{}{
	"audio/aac":  {},
	"audio/mp4":  {},
	"audio/mpeg": {},
	"audio/amr":  {},
	"audio/ogg":  {},
	"audio/opus": {},
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
	"image/webp":               {},
	"video/mp4":                {},
	"video/3gpp":               {},
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

// Account converts the on-disk Config into the shape the API client wants.
func (c Config) Account() whatsapp.Account {
	return whatsapp.Account{
		PhoneNumberID: c.PhoneNumberID,
		WABAID:        c.WABAID,
		AccessToken:   c.AccessToken,
		AppSecret:     c.AppSecret,
		APIVersion:    c.APIVersion,
	}
}

func (c Config) CSATLanguage() string {
	if strings.TrimSpace(c.CSATTemplateLanguage) == "" {
		return DefaultCSATTemplateLanguage
	}
	return c.CSATTemplateLanguage
}

func (c Config) CSATBody() string {
	if strings.TrimSpace(c.CSATTemplateBody) == "" {
		return DefaultCSATTemplateBody
	}
	return c.CSATTemplateBody
}

func (c Config) CSATButtonText() string {
	if strings.TrimSpace(c.CSATTemplateButtonText) == "" {
		return DefaultCSATTemplateButtonText
	}
	return c.CSATTemplateButtonText
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

// SourceIDUpdater persists the Meta message ID after a successful send for status correlation, and records per-attachment delivery so a retry does not re-deliver media.
type SourceIDUpdater interface {
	UpdateMessageSourceID(messageUUID, sourceID string) error
	SetWhatsAppSentAttachments(messageUUID string, attachmentUUIDs []string) error
}

// WhatsApp implements inbox.Inbox.
type WhatsApp struct {
	id            int
	name          string
	config        Config
	client        *whatsapp.Client
	lo            *logf.Logger
	messageStore  inbox.MessageStore
	sourceUpdater SourceIDUpdater
}

// Opts holds construction options.
type Opts struct {
	ID            int
	Name          string
	Config        Config
	Client        *whatsapp.Client
	Lo            *logf.Logger
	SourceUpdater SourceIDUpdater
}

// New constructs a WhatsApp inbox.
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
	}, nil
}

// Identifier returns the inbox database ID.
func (w *WhatsApp) Identifier() int { return w.id }

// AppSecret returns the Meta app secret.
func (w *WhatsApp) AppSecret() string { return w.config.AppSecret }

// Receive is a no-op; inbound messages arrive via the webhook handler.
func (w *WhatsApp) Receive(ctx context.Context) error { return nil }

// Channel returns the channel name.
func (w *WhatsApp) Channel() string { return ChannelWhatsApp }

// Name returns the configured inbox name.
func (w *WhatsApp) Name() string { return w.name }

// FromAddress is not applicable to WhatsApp.
func (w *WhatsApp) FromAddress() string { return "" }

// ReplyToAddress is not applicable to WhatsApp.
func (w *WhatsApp) ReplyToAddress() string { return "" }

// FromNameTemplate is not applicable to WhatsApp and always returns empty.
func (w *WhatsApp) FromNameTemplate() string { return "" }

// Close releases any resources. Currently a no-op.
func (w *WhatsApp) Close() error { return nil }

// Send dispatches an outbound message to Meta.
func (w *WhatsApp) Send(message models.OutboundMessage) error {
	meta, err := parseSendMeta(message.Meta)
	if err != nil {
		return fmt.Errorf("parsing whatsapp send meta: %w", err)
	}
	if meta.ToPhone == "" {
		return fmt.Errorf("missing recipient phone number on outbound message")
	}

	timeout := MetaCallTimeout
	if n := len(message.Attachments); n > 0 {
		timeout = time.Duration(1+n) * MetaCallTimeout
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
		sourceID, err = w.sendAttachments(ctx, acc, meta, message)

	case strings.TrimSpace(textBody(message)) != "":
		sourceID, err = w.client.SendText(ctx, acc, meta.ToPhone, textBody(message), meta.ReplyToWAMessageID)

	default:
		return fmt.Errorf("outbound message has no content")
	}

	// Persist the source id even on a partial-failure error path, so a status webhook for the already-delivered media still correlates.
	if sourceID != "" && w.sourceUpdater != nil {
		if upErr := w.sourceUpdater.UpdateMessageSourceID(message.UUID, sourceID); upErr != nil {
			w.lo.Error("failed to persist whatsapp source id", "message_uuid", message.UUID, "source_id", sourceID, "error", upErr)
		}
	}
	return err
}

// sendAttachments sends one media message per attachment with the caption and reply context on the first; a caption that can't ride the first media type (audio/sticker) is sent as standalone text first.
// Attachments already delivered on a prior attempt (tracked in message meta) are skipped, so retrying a partially-sent message never re-delivers media.
func (w *WhatsApp) sendAttachments(ctx context.Context, acc whatsapp.Account, meta SendMeta, message models.OutboundMessage) (string, error) {
	if bad := rejectedAttachments(message.Attachments); len(bad) > 0 {
		return "", fmt.Errorf("WhatsApp can't send these files: %s", strings.Join(bad, "; "))
	}

	caption := strings.TrimSpace(textBody(message))
	replyTo := meta.ReplyToWAMessageID
	var firstID string

	sent := parseSentAttachmentMarkers(message.Meta)
	fail := func(err error) (string, error) {
		if w.sourceUpdater != nil {
			if perr := w.sourceUpdater.SetWhatsAppSentAttachments(message.UUID, markerKeys(sent)); perr != nil {
				w.lo.Error("failed to persist whatsapp sent-attachment markers", "message_uuid", message.UUID, "error", perr)
			}
		}
		return firstID, err
	}

	if caption != "" && len(message.Attachments) > 0 {
		if t := mediaTypeForAttachment(message.Attachments[0]); t != "image" && t != "video" && t != "document" {
			if !sent[captionMarker] {
				id, err := w.client.SendText(ctx, acc, meta.ToPhone, caption, replyTo)
				if err != nil {
					return fail(err)
				}
				sent[captionMarker] = true
				firstID = id
			}
			caption, replyTo = "", ""
		}
	}

	for i, att := range message.Attachments {
		if att.UUID != "" && sent[att.UUID] {
			continue
		}
		mediaType := mediaTypeForAttachment(att)
		mediaID, err := w.client.UploadMedia(ctx, acc, att.Content, att.ContentType, att.Name)
		if err != nil {
			return fail(fmt.Errorf("uploading attachment to meta: %w", err))
		}

		var attCaption, attReply string
		if i == 0 {
			attCaption = caption
			attReply = replyTo
		}

		id, err := w.client.SendMedia(ctx, acc, meta.ToPhone, mediaType, mediaID, attCaption, att.Name, attReply)
		if err != nil {
			return fail(err)
		}
		if att.UUID != "" {
			sent[att.UUID] = true
		}
		if i == 0 && firstID == "" {
			firstID = id
		}
	}
	return firstID, nil
}

// parseSentAttachmentMarkers reads the set of attachment UUIDs (and the caption marker) already delivered for this message from its meta.
func parseSentAttachmentMarkers(raw json.RawMessage) map[string]bool {
	out := map[string]bool{}
	if len(raw) == 0 {
		return out
	}
	var env struct {
		Sent []string `json:"whatsapp_sent_attachments"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return out
	}
	for _, k := range env.Sent {
		out[k] = true
	}
	return out
}

func markerKeys(sent map[string]bool) []string {
	keys := make([]string, 0, len(sent))
	for k := range sent {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
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

// rejectedAttachments returns a human-readable reason for every attachment Meta would reject on type or size.
func rejectedAttachments(atts []attachment.Attachment) []string {
	var reasons []string
	for _, att := range atts {
		if _, ok := supportedMediaMIMETypes[normalizeMIME(att.ContentType)]; !ok {
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
		return maxImageBytes
	case "video":
		return maxVideoBytes
	case "audio":
		return maxAudioBytes
	case "sticker":
		return maxStickerBytes
	}
	return maxDocumentBytes
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

// mediaTypeForAttachment maps a mime type to the WhatsApp media type; formats Meta doesn't accept natively go as documents.
func mediaTypeForAttachment(att attachment.Attachment) string {
	switch normalizeMIME(att.ContentType) {
	case "image/jpeg", "image/png":
		return "image"
	case "image/webp":
		if att.Size > 0 && att.Size <= maxStickerBytes {
			return "sticker"
		}
		return "document"
	case "video/mp4", "video/3gpp", "video/3gp":
		return "video"
	case "audio/aac", "audio/mp4", "audio/mpeg", "audio/amr", "audio/ogg", "audio/opus":
		return "audio"
	}
	return "document"
}
