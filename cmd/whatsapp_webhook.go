package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/attachment"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/countries"
	"github.com/abhinavxd/libredesk/internal/envelope"
	whatsappChannel "github.com/abhinavxd/libredesk/internal/inbox/channel/whatsapp"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/abhinavxd/libredesk/internal/whatsapp"
	"github.com/valyala/fasthttp"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/fastglue"
)

const whatsAppDefaultContactName = "Customer"

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
				if err := app.whatsappTemplate.HandleStatusUpdate(ibID, ts.TemplateName, ts.Language, ts.Event, ts.Reason); err != nil {
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
	inboxID = resolveWhatsAppInboxByPhoneNumberID(app, inboxID, m.PhoneNumberID)

	cfg, err := whatsAppConfigForInbox(app, inboxID)
	if err != nil {
		return fmt.Errorf("resolving inbox config: %w", err)
	}

	app.lo.Debug("ingesting whatsapp message", "wa_message_id", m.ID, "type", m.Type, "media_id", m.MediaID, "mime", m.MediaMimeType, "context_id", m.ContextID)

	// Skip the media download up front when the message is already ingested (retries, Meta redeliveries).
	if exists, err := app.conversation.MessageExists(m.ID); err != nil {
		return fmt.Errorf("checking duplicate: %w", err)
	} else if exists {
		return nil
	}

	// Download media before taking the per-sender lock; a slow CDN must not stall other senders' workers.
	attachments := fetchWhatsAppAttachments(ctx, app, cfg, m)
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
	if errors.Is(err, sql.ErrNoRows) && cfg.ReopenWindowHours > 0 {
		// Reuse a recently-resolved conversation; the message insert hook reopens it.
		conversationID, conversationUUID, err = app.conversation.GetReopenableConversationForContact(contactID, inboxID, cfg.ReopenWindowHours)
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

	content, contentType := messageContent(m)

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

// fetchWhatsAppAttachments downloads the inbound message's media from Meta as a single-element attachment slice.
func fetchWhatsAppAttachments(ctx context.Context, app *App, cfg whatsappChannel.Config, m whatsapp.ParsedMessage) attachment.Attachments {
	if m.MediaID == "" || app.whatsappClient == nil {
		return nil
	}
	acc := cfg.Account()

	// Processing is async and Meta won't redeliver, so transient CDN errors are retried here.
	var (
		info whatsapp.MediaInfo
		body []byte
		err  error
	)
	for attempt := 1; ; attempt++ {
		err = func() error {
			dlCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
			defer cancel()
			info, err = app.whatsappClient.GetMediaURL(dlCtx, acc, m.MediaID)
			if err != nil {
				return err
			}
			body, err = app.whatsappClient.DownloadMedia(dlCtx, acc, info.URL)
			return err
		}()
		if err == nil {
			break
		}
		if ctx.Err() != nil {
			return nil
		}
		if attempt >= 3 {
			app.lo.Error("error downloading whatsapp media, giving up", "media_id", m.MediaID, "attempts", attempt, "error", err)
			return nil
		}
		app.lo.Warn("error downloading whatsapp media, retrying", "media_id", m.MediaID, "attempt", attempt, "error", err)
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(2 * time.Second):
		}
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
	}
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
		Type:        umodels.UserTypeContact,
		FirstName:   first,
		LastName:    last,
		PhoneNumber: null.StringFrom(m.From),
	}
	id, err := app.user.UpsertContactByChannelIdentity(whatsappChannel.ChannelWhatsApp, m.From, &contact)
	if err != nil {
		return 0, err
	}
	// CreateContact's insert doesn't persist phone_number; backfill without clobbering an agent-edited value.
	countryCode, nationalNumber := countries.SplitPhoneCountryCode(m.From)
	if err := app.user.SetContactPhoneIfMissing(id, nationalNumber, countryCode); err != nil {
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

func messageContent(m whatsapp.ParsedMessage) (string, string) {
	return textPreview(m), cmodels.ContentTypeText
}

func mapWhatsAppStatus(metaStatus string) string {
	if metaStatus == "failed" {
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
	forEachEnabledWhatsAppInbox(app, func(id int, cfg whatsappChannel.Config) bool {
		if cfg.WABAID != "" {
			out[cfg.WABAID] = append(out[cfg.WABAID], id)
		}
		return true
	})
	return out
}

// resolveWhatsAppInboxByPhoneNumberID returns the enabled WhatsApp inbox whose config matches the payload's phone_number_id, falling back to the URL inbox.
func resolveWhatsAppInboxByPhoneNumberID(app *App, urlInboxID int, phoneNumberID string) int {
	if phoneNumberID == "" {
		return urlInboxID
	}
	if cfg, err := whatsAppConfigForInbox(app, urlInboxID); err == nil && cfg.PhoneNumberID == phoneNumberID {
		return urlInboxID
	}
	resolved := urlInboxID
	forEachEnabledWhatsAppInbox(app, func(id int, cfg whatsappChannel.Config) bool {
		if id == urlInboxID || cfg.PhoneNumberID != phoneNumberID {
			return true
		}
		app.lo.Info("routing whatsapp message by phone_number_id", "url_inbox_id", urlInboxID, "resolved_inbox_id", id)
		resolved = id
		return false
	})
	return resolved
}

// forEachEnabledWhatsAppInbox invokes fn with each enabled WhatsApp inbox's ID and decoded config; returning false stops iteration.
func forEachEnabledWhatsAppInbox(app *App, fn func(id int, cfg whatsappChannel.Config) bool) {
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
			continue
		}
		if !fn(rec.ID, cfg) {
			return
		}
	}
}

func whatsAppConfigForInbox(app *App, inboxID int) (whatsappChannel.Config, error) {
	rec, err := app.inbox.GetDBRecord(inboxID)
	if err != nil {
		return whatsappChannel.Config{}, err
	}
	if rec.Channel != whatsappChannel.ChannelWhatsApp {
		return whatsappChannel.Config{}, fmt.Errorf("inbox %d is not a whatsapp inbox", inboxID)
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
