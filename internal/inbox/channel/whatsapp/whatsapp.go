// Package whatsapp implements a WhatsApp Cloud API inbox; inbound arrives via the webhook handler in cmd/, so Receive() is a no-op like livechat.
package whatsapp

import (
	"context"
	"encoding/json"
	"fmt"
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

// maxStickerBytes is Meta's static sticker size cap; larger webp files are sent as documents.
const maxStickerBytes = 100 * 1024

// Config is the per-inbox WhatsApp configuration from the inbox config JSONB, with tokens already decrypted.
type Config struct {
	PhoneNumberID      string `json:"phone_number_id"`
	WABAID             string `json:"waba_id"`
	AccessToken        string `json:"access_token"`
	AppSecret          string `json:"app_secret"`
	WebhookVerifyToken string `json:"webhook_verify_token"`
	APIVersion         string `json:"api_version"`
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
	TemplateHeaderMediaID string                    `json:"template_header_media_id,omitempty"`
	TemplateButtons       []whatsapp.TemplateButton `json:"template_buttons,omitempty"`
}

// SourceIDUpdater persists the Meta message ID after a successful send for status correlation.
type SourceIDUpdater interface {
	UpdateMessageSourceID(messageUUID, sourceID string) error
}

// WhatsApp implements inbox.Inbox.
type WhatsApp struct {
	id            int
	config        Config
	from          string
	client        *whatsapp.Client
	lo            *logf.Logger
	messageStore  inbox.MessageStore
	sourceUpdater SourceIDUpdater
}

// Opts holds construction options.
type Opts struct {
	ID            int
	Config        Config
	From          string
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
		config:        opts.Config,
		from:          opts.From,
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

// FromAddress returns the configured display "from" string.
func (w *WhatsApp) FromAddress() string { return w.from }

// ReplyToAddress is not applicable to WhatsApp.
func (w *WhatsApp) ReplyToAddress() string { return "" }

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
			HeaderMediaID: meta.TemplateHeaderMediaID,
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

	if err != nil {
		return err
	}
	if sourceID != "" && w.sourceUpdater != nil {
		if upErr := w.sourceUpdater.UpdateMessageSourceID(message.UUID, sourceID); upErr != nil {
			w.lo.Error("failed to persist whatsapp source id", "message_uuid", message.UUID, "source_id", sourceID, "error", upErr)
		}
	}
	return nil
}

// sendAttachments sends one media message per attachment with the caption and reply context on the first; a caption that can't ride the first media type (audio/sticker) is sent as standalone text first.
func (w *WhatsApp) sendAttachments(ctx context.Context, acc whatsapp.Account, meta SendMeta, message models.OutboundMessage) (string, error) {
	caption := strings.TrimSpace(textBody(message))
	replyTo := meta.ReplyToWAMessageID
	var firstID string

	if caption != "" && len(message.Attachments) > 0 {
		if t := mediaTypeForAttachment(message.Attachments[0]); t != "image" && t != "video" && t != "document" {
			id, err := w.client.SendText(ctx, acc, meta.ToPhone, caption, replyTo)
			if err != nil {
				return "", err
			}
			firstID, caption, replyTo = id, "", ""
		}
	}

	for i, att := range message.Attachments {
		mediaType := mediaTypeForAttachment(att)
		mediaID, err := w.client.UploadMedia(ctx, acc, att.Content, att.ContentType, att.Name)
		if err != nil {
			return firstID, fmt.Errorf("uploading attachment to meta: %w", err)
		}

		var attCaption, attReply string
		if i == 0 {
			attCaption = caption
			attReply = replyTo
		}

		id, err := w.client.SendMedia(ctx, acc, meta.ToPhone, mediaType, mediaID, attCaption, att.Name, attReply)
		if err != nil {
			return firstID, err
		}
		if i == 0 && firstID == "" {
			firstID = id
		}
	}
	return firstID, nil
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

// mediaTypeForAttachment maps a mime type to the WhatsApp media type; formats Meta doesn't accept natively go as documents.
func mediaTypeForAttachment(att attachment.Attachment) string {
	mime := strings.ToLower(strings.TrimSpace(att.ContentType))
	if i := strings.Index(mime, ";"); i >= 0 {
		mime = strings.TrimSpace(mime[:i])
	}
	switch mime {
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
