package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/attachment"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	whatsappChannel "github.com/abhinavxd/libredesk/internal/inbox/channel/whatsapp"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/abhinavxd/libredesk/internal/whatsapp"
	"github.com/valyala/fasthttp"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/fastglue"
)

const whatsAppDefaultContactName = "Contact"

// handleWhatsAppWebhookVerify responds to Meta's GET verification challenge.
func handleWhatsAppWebhookVerify(r *fastglue.Request) error {
	app := r.Context.(*App)

	inboxID, err := inboxIDFromPath(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "invalid inbox id", nil, envelope.InputError)
	}

	mode := string(r.RequestCtx.QueryArgs().Peek("hub.mode"))
	token := string(r.RequestCtx.QueryArgs().Peek("hub.verify_token"))
	challenge := string(r.RequestCtx.QueryArgs().Peek("hub.challenge"))

	if mode != "subscribe" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "invalid hub.mode", nil, envelope.InputError)
	}

	cfg, err := whatsAppConfigForInbox(app, inboxID)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "inbox not found", nil, envelope.NotFoundError)
	}

	if cfg.WebhookVerifyToken == "" || token != cfg.WebhookVerifyToken {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "verify token mismatch", nil, envelope.PermissionError)
	}

	r.RequestCtx.SetStatusCode(fasthttp.StatusOK)
	r.RequestCtx.SetBodyString(challenge)
	return nil
}

// handleWhatsAppWebhookEvent processes a Meta webhook delivery.
func handleWhatsAppWebhookEvent(r *fastglue.Request) error {
	app := r.Context.(*App)

	inboxID, err := inboxIDFromPath(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "invalid inbox id", nil, envelope.InputError)
	}

	body := append([]byte(nil), r.RequestCtx.PostBody()...)

	// Verify against the running inbox's secret so the webhook keeps working (and stays durable) during a DB outage.
	appSecret, ok := whatsAppRunningAppSecret(app, inboxID)
	if !ok {
		cfg, err := whatsAppConfigForInbox(app, inboxID)
		if err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "inbox not found", nil, envelope.NotFoundError)
		}
		appSecret = cfg.AppSecret
	}
	if appSecret == "" {
		app.lo.Error("whatsapp webhook rejected: app secret not configured", "inbox_id", inboxID)
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "webhook app secret not configured", nil, envelope.PermissionError)
	}
	signature := string(r.RequestCtx.Request.Header.Peek("X-Hub-Signature-256"))
	if !whatsapp.VerifySignature(body, signature, appSecret) {
		app.lo.Warn("whatsapp webhook signature verification failed", "inbox_id", inboxID)
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "invalid signature", nil, envelope.PermissionError)
	}

	if app.whatsappIngester == nil {
		app.lo.Error("whatsapp ingester not initialized", "inbox_id", inboxID)
		return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "whatsapp ingester unavailable", nil, envelope.GeneralError)
	}
	if err := app.whatsappIngester.Enqueue(inboxID, body); err != nil {
		app.lo.Error("error enqueuing whatsapp webhook to durable stream, asking meta to retry", "inbox_id", inboxID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "busy, retry shortly", nil, envelope.GeneralError)
	}

	return r.SendEnvelope(map[string]string{"status": "ok"})
}

// processWhatsAppPayload applies every message/status/template event in one delivery; returning an error retries the whole delivery.
func processWhatsAppPayload(ctx context.Context, app *App, inboxID int, payload *whatsapp.WebhookPayload) error {
	var errs []error

	for _, msg := range payload.ExtractMessages() {
		if err := ingestWhatsAppMessage(ctx, app, inboxID, msg); err != nil {
			app.lo.Error("error ingesting whatsapp message", "inbox_id", inboxID, "wa_message_id", msg.ID, "error", err)
			errs = append(errs, err)
		}
	}

	for _, st := range payload.ExtractStatuses() {
		if err := app.conversation.UpdateMessageStatusBySourceID(st.MessageID, mapWhatsAppStatus(st.Status)); err != nil {
			app.lo.Error("error applying whatsapp status update", "wa_message_id", st.MessageID, "status", st.Status, "error", err)
			errs = append(errs, err)
		}
		if err := app.conversation.RecordWhatsAppStatus(st.MessageID, st.Status, st.Timestamp, st.UserMsg); err != nil {
			app.lo.Error("error recording whatsapp status meta", "wa_message_id", st.MessageID, "status", st.Status, "error", err)
			errs = append(errs, err)
		}
	}

	templateUpdates := payload.ExtractTemplateStatusUpdates()
	if app.whatsappTemplate != nil && len(templateUpdates) > 0 {
		wabaInboxes := whatsAppInboxIDsByWABA(app)
		for _, ts := range templateUpdates {
			ids := wabaInboxes[ts.WABAID]
			if ts.WABAID == "" {
				ids = []int{inboxID}
			}
			for _, ibID := range ids {
				if err := app.whatsappTemplate.HandleStatusUpdate(ibID, ts.MetaTemplateID, ts.TemplateName, ts.Language, ts.Event, ts.Reason); err != nil {
					app.lo.Error("error applying template status update", "inbox_id", ibID, "name", ts.TemplateName, "event", ts.Event, "error", err)
					errs = append(errs, err)
				}
			}
		}
	}

	return errors.Join(errs...)
}

func ingestWhatsAppMessage(ctx context.Context, app *App, inboxID int, m whatsapp.ParsedMessage) error {
	if m.ID == "" || m.From == "" {
		return fmt.Errorf("missing message id or sender")
	}

	// Reactions and sync/welcome events would otherwise land as placeholder rows and reset the 24h window.
	switch m.Type {
	case "reaction", "ephemeral", "request_welcome":
		return nil
	}

	// Meta posts all events of an app to one callback URL, so the URL's inbox ID is not authoritative.
	inbRec, cfg, err := resolveWhatsAppInbox(app, inboxID, m.PhoneNumberID)
	if err != nil {
		return fmt.Errorf("resolving inbox: %w", err)
	}
	inboxID = inbRec.ID

	app.lo.Debug("ingesting whatsapp message", "wa_message_id", m.ID, "type", m.Type, "media_id", m.MediaID, "mime", m.MediaMimeType, "context_id", m.ContextID)

	// Skip the media download up front when the message is already ingested (retries, Meta redeliveries).
	if exists, err := app.conversation.MessageExists(m.ID); err != nil {
		return fmt.Errorf("checking duplicate: %w", err)
	} else if exists {
		return nil
	}

	// Download media before taking the per-sender lock; a slow CDN must not stall other senders' workers.
	attachments, err := fetchWhatsAppAttachments(ctx, app, cfg, m)
	if err != nil {
		return fmt.Errorf("downloading whatsapp media: %w", err)
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Serializing per sender keeps duplicate deliveries and concurrent messages from double-creating rows.
	if app.whatsappIngester != nil {
		unlock := app.whatsappIngester.lockSender(m.From)
		defer unlock()
	}

	if exists, err := app.conversation.MessageExists(m.ID); err != nil {
		return fmt.Errorf("checking duplicate: %w", err)
	} else if exists {
		return nil
	}

	contactID, err := upsertWhatsAppContact(app, m)
	if err != nil {
		return fmt.Errorf("resolving contact: %w", err)
	}

	isNewConversation := false
	conversationID, conversationUUID, err := app.conversation.GetLatestOpenConversationForContact(contactID, inboxID)
	if errors.Is(err, sql.ErrNoRows) && inbRec.ReopenWindowHours > 0 {
		// Reuse a recently-resolved conversation; the message insert hook reopens it.
		conversationID, conversationUUID, err = app.conversation.GetReopenableConversationForContact(contactID, inboxID, inbRec.ReopenWindowHours)
	}
	if errors.Is(err, sql.ErrNoRows) {
		conversationID, conversationUUID, err = app.conversation.CreateConversation(
			contactID,
			inboxID,
			textPreview(m),
			time.Now(),
			"",
			false,
			nil,
			nil,
			0,
			0,
		)
		if err != nil {
			return fmt.Errorf("creating conversation: %w", err)
		}
		isNewConversation = true
	} else if err != nil {
		return fmt.Errorf("looking up conversation: %w", err)
	}

	content, contentType := textPreview(m), cmodels.ContentTypeText

	// The "[image]"-style placeholder only stays when the media download failed.
	if m.Text == "" && m.Caption == "" && len(attachments) > 0 {
		content = ""
	}

	msg := cmodels.Message{
		Channel:          whatsappChannel.ChannelWhatsApp,
		ConversationID:   conversationID,
		ConversationUUID: conversationUUID,
		SenderID:         contactID,
		SenderType:       cmodels.SenderTypeContact,
		Type:             cmodels.MessageIncoming,
		Status:           cmodels.MessageStatusReceived,
		InboxID:          inboxID,
		Content:          content,
		ContentType:      contentType,
		SourceID:         null.StringFrom(m.ID),
		Attachments:      attachments,
		Meta:             buildInboundMeta(app, m),
	}

	if _, err := app.conversation.ProcessIncomingWhatsAppMessage(msg, isNewConversation, m.Timestamp); err != nil {
		return fmt.Errorf("processing whatsapp message: %w", err)
	}
	return nil
}

// buildInboundMeta records reply-context and interactive-button provenance; nil when there's nothing to record.
func buildInboundMeta(app *App, m whatsapp.ParsedMessage) json.RawMessage {
	patch := map[string]any{}

	if m.ContextID != "" {
		// Best-effort: the quoted wamid may predate this inbox, so the raw Meta ID is kept alongside.
		if uuid, err := app.conversation.MessageUUIDBySourceID(m.ContextID); err == nil && uuid != "" {
			patch["wa_reply_to_message_uuid"] = uuid
		}
		patch["wa_reply_to_source_id"] = m.ContextID
	}
	if m.ButtonReplyID != "" {
		patch["wa_button_reply_id"] = m.ButtonReplyID
	}
	if m.ListReplyID != "" {
		patch["wa_list_reply_id"] = m.ListReplyID
	}
	if m.Type == "unsupported" {
		patch["wa_unsupported"] = true
	}

	if len(patch) == 0 {
		return nil
	}
	raw, err := json.Marshal(patch)
	if err != nil {
		app.lo.Error("error marshalling whatsapp inbound meta", "wa_message_id", m.ID, "error", err)
		return nil
	}
	return raw
}

// fetchWhatsAppAttachments downloads the inbound media; a permanent (4xx) failure returns (nil, nil) for a placeholder, any other error propagates so the queue retries.
func fetchWhatsAppAttachments(ctx context.Context, app *App, cfg whatsappChannel.Config, m whatsapp.ParsedMessage) (attachment.Attachments, error) {
	if m.MediaID == "" || app.whatsappClient == nil {
		return nil, nil
	}
	acc := cfg.Account()

	var (
		info whatsapp.MediaInfo
		body []byte
		err  error
	)
	for attempt := 1; ; attempt++ {
		dlCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
		info, err = app.whatsappClient.GetMediaURL(dlCtx, acc, m.MediaID)
		if err == nil {
			body, err = app.whatsappClient.DownloadMedia(dlCtx, acc, info.URL)
		}
		cancel()
		if err == nil {
			break
		}
		if ctx.Err() != nil {
			return nil, nil
		}
		if attempt >= 3 {
			if isPermanentMediaError(err) {
				app.lo.Warn("whatsapp media permanently unavailable, inserting placeholder", "media_id", m.MediaID, "attempts", attempt, "error", err)
				return nil, nil
			}
			app.lo.Warn("error downloading whatsapp media, will retry job", "media_id", m.MediaID, "attempts", attempt, "error", err)
			return nil, err
		}
		app.lo.Warn("error downloading whatsapp media, retrying", "media_id", m.MediaID, "attempt", attempt, "error", err)
		select {
		case <-ctx.Done():
			return nil, nil
		case <-time.After(2 * time.Second):
		}
	}

	if len(body) == 0 {
		app.lo.Warn("whatsapp media downloaded empty, inserting placeholder", "media_id", m.MediaID, "type", m.Type)
		return nil, nil
	}

	contentType := info.MimeType
	if contentType == "" {
		contentType = m.MediaMimeType
	}
	filename := m.Filename
	if filename == "" {
		filename = defaultMediaFilename(m.Type, contentType)
	}

	return attachment.Attachments{
		attachment.Attachment{
			Name:        filename,
			ContentType: contentType,
			Content:     body,
			Size:        len(body),
			Disposition: attachment.DispositionAttachment,
		},
	}, nil
}

func isPermanentMediaError(err error) bool {
	var me *whatsapp.MetaAPIError
	if errors.As(err, &me) {
		// 408 and 429 are 4xx but retryable; 5xx are transient.
		if me.StatusCode == http.StatusRequestTimeout || me.StatusCode == http.StatusTooManyRequests {
			return false
		}
		return me.StatusCode >= 400 && me.StatusCode < 500
	}
	// Transport errors.
	var netErr net.Error
	if errors.As(err, &netErr) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, io.ErrUnexpectedEOF) {
		return false
	}
	return true
}

func defaultMediaFilename(messageType, mime string) string {
	if i := strings.Index(mime, ";"); i >= 0 {
		mime = strings.TrimSpace(mime[:i])
	}
	ext := "bin"
	if i := strings.LastIndex(mime, "/"); i >= 0 && i+1 < len(mime) {
		ext = mime[i+1:]
	}
	switch messageType {
	case "image":
		return "image." + ext
	case "video":
		return "video." + ext
	case "audio", "voice":
		return "audio." + ext
	case "document":
		return "document." + ext
	case "sticker":
		return "sticker." + ext
	}
	return "attachment." + ext
}

func upsertWhatsAppContact(app *App, m whatsapp.ParsedMessage) (int, error) {
	first, last := splitName(m.ContactName)
	contact := umodels.User{
		Type:      umodels.UserTypeContact,
		FirstName: first,
		LastName:  last,
	}
	id, err := app.user.UpsertContactByChannelIdentity(whatsappChannel.ChannelWhatsApp, m.From, &contact)
	if err != nil {
		return 0, err
	}
	if err := app.user.SetContactPhoneIfMissing(id, m.From, ""); err != nil {
		app.lo.Error("error setting whatsapp contact phone", "user_id", id, "error", err)
	}
	// A contact created from a message without a profile name picks the real name up later.
	if m.ContactName != "" {
		if err := app.user.UpdateContactNameIfDefault(id, first, last, whatsAppDefaultContactName); err != nil {
			app.lo.Error("error updating whatsapp contact name", "user_id", id, "error", err)
		}
	}
	return id, nil
}

func splitName(name string) (string, string) {
	if name == "" {
		return whatsAppDefaultContactName, ""
	}
	first, last, _ := strings.Cut(name, " ")
	return first, last
}

func textPreview(m whatsapp.ParsedMessage) string {
	if m.Text != "" {
		return m.Text
	}
	if m.Caption != "" {
		return m.Caption
	}
	switch m.Type {
	case "image":
		return "[image]"
	case "video":
		return "[video]"
	case "audio", "voice":
		return "[audio]"
	case "document":
		return "[document]"
	case "sticker":
		return "[sticker]"
	case "unsupported":
		// Meta refuses to deliver some message types (e.g. animated stickers) to the Cloud API.
		return "[unsupported message: not delivered by WhatsApp]"
	}
	return "[whatsapp message]"
}


func mapWhatsAppStatus(metaStatus string) string {
	if metaStatus == cmodels.MessageStatusFailed {
		return cmodels.MessageStatusFailed
	}
	return cmodels.MessageStatusSent
}

func inboxIDFromPath(r *fastglue.Request) (int, error) {
	raw, ok := r.RequestCtx.UserValue("inbox_id").(string)
	if !ok || raw == "" {
		return 0, fmt.Errorf("missing inbox_id")
	}
	return strconv.Atoi(raw)
}

// whatsAppInboxIDsByWABA maps each non-empty WABA id to its enabled WhatsApp inbox IDs.
func whatsAppInboxIDsByWABA(app *App) map[string][]int {
	out := map[string][]int{}
	forEachEnabledWhatsAppInbox(app, func(rec imodels.Inbox, cfg whatsappChannel.Config) bool {
		if cfg.WABAID != "" {
			out[cfg.WABAID] = append(out[cfg.WABAID], rec.ID)
		}
		return true
	})
	return out
}

// resolveWhatsAppInbox returns the inbox record and decoded config for the payload's phone_number_id, falling back to the URL inbox.
func resolveWhatsAppInbox(app *App, urlInboxID int, phoneNumberID string) (imodels.Inbox, whatsappChannel.Config, error) {
	rec, urlErr := app.inbox.GetDBRecord(urlInboxID)
	var cfg whatsappChannel.Config
	if urlErr == nil {
		cfg, urlErr = whatsAppConfigFromRecord(rec)
	}
	if urlErr == nil && (phoneNumberID == "" || cfg.PhoneNumberID == phoneNumberID) {
		return rec, cfg, nil
	}

	// A bad URL inbox must not block routing by the payload's phone_number_id.
	if phoneNumberID != "" {
		var (
			found    bool
			foundRec imodels.Inbox
			foundCfg whatsappChannel.Config
		)
		forEachEnabledWhatsAppInbox(app, func(r imodels.Inbox, c whatsappChannel.Config) bool {
			if c.PhoneNumberID != phoneNumberID {
				return true
			}
			foundRec, foundCfg, found = r, c, true
			return false
		})
		if found {
			if foundRec.ID != urlInboxID {
				app.lo.Info("routing whatsapp message by phone_number_id", "url_inbox_id", urlInboxID, "resolved_inbox_id", foundRec.ID)
			}
			return foundRec, foundCfg, nil
		}
	}

	if urlErr != nil {
		return imodels.Inbox{}, whatsappChannel.Config{}, urlErr
	}
	return rec, cfg, nil
}

// forEachEnabledWhatsAppInbox invokes fn with each enabled WhatsApp inbox's record and decoded config; returning false stops iteration.
func forEachEnabledWhatsAppInbox(app *App, fn func(rec imodels.Inbox, cfg whatsappChannel.Config) bool) {
	inboxes, err := app.inbox.GetAll()
	if err != nil {
		return
	}
	for _, rec := range inboxes {
		if rec.Channel != whatsappChannel.ChannelWhatsApp || !rec.Enabled {
			continue
		}
		var cfg whatsappChannel.Config
		if err := json.Unmarshal(rec.Config, &cfg); err != nil {
			app.lo.Warn("skipping whatsapp inbox with unparseable config", "inbox_id", rec.ID, "error", err)
			continue
		}
		if !fn(rec, cfg) {
			return
		}
	}
}

func whatsAppConfigForInbox(app *App, inboxID int) (whatsappChannel.Config, error) {
	rec, err := app.inbox.GetDBRecord(inboxID)
	if err != nil {
		return whatsappChannel.Config{}, err
	}
	return whatsAppConfigFromRecord(rec)
}

func whatsAppConfigFromRecord(rec imodels.Inbox) (whatsappChannel.Config, error) {
	if rec.Channel != whatsappChannel.ChannelWhatsApp {
		return whatsappChannel.Config{}, fmt.Errorf("inbox %d is not a whatsapp inbox", rec.ID)
	}
	var cfg whatsappChannel.Config
	if err := json.Unmarshal(rec.Config, &cfg); err != nil {
		return whatsappChannel.Config{}, fmt.Errorf("decoding whatsapp inbox config: %w", err)
	}
	return cfg, nil
}

// whatsAppRunningAppSecret returns the app secret from the in-memory inbox; ok is false for a disabled or unregistered inbox.
func whatsAppRunningAppSecret(app *App, inboxID int) (string, bool) {
	inb, err := app.inbox.Get(inboxID)
	if err != nil {
		return "", false
	}
	wa, ok := inb.(interface{ AppSecret() string })
	if !ok {
		return "", false
	}
	return wa.AppSecret(), true
}

// markWhatsAppMessageRead sends a read receipt to Meta for an inbound message; best-effort, logs and swallows failures.
func markWhatsAppMessageRead(app *App, inboxID int, sourceID string) {
	if app.whatsappClient == nil || sourceID == "" {
		return
	}
	cfg, err := whatsAppConfigForInbox(app, inboxID)
	if err != nil {
		app.lo.Error("error fetching inbox config for whatsapp read receipt", "inbox_id", inboxID, "error", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), whatsappChannel.MetaCallTimeout)
	defer cancel()
	if err := app.whatsappClient.MarkRead(ctx, cfg.Account(), sourceID); err != nil {
		app.lo.Warn("error marking whatsapp message read", "inbox_id", inboxID, "source_id", sourceID, "error", err)
	}
}
