package main

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/attachment"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	telegramChannel "github.com/abhinavxd/libredesk/internal/inbox/channel/telegram"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/abhinavxd/libredesk/internal/telegram"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/fastglue"
)

var telegramChatLocks = &keyedLock{entries: make(map[string]*keyedLockEntry)}

func handleTelegramWebhook(r *fastglue.Request) error {
	app := r.Context.(*App)
	inboxID, err := inboxIDFromPath(r)
	if err != nil {
		return r.SendErrorEnvelope(http.StatusBadRequest, "invalid inbox id", nil, envelope.InputError)
	}
	rec, err := app.inbox.GetDBRecord(inboxID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if rec.Channel != telegramChannel.ChannelTelegram {
		return r.SendErrorEnvelope(http.StatusNotFound, "inbox not found", nil, envelope.NotFoundError)
	}
	cfg, err := telegramConfigFromRecord(rec)
	if err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, "invalid inbox configuration", nil, envelope.GeneralError)
	}
	secret := r.RequestCtx.Request.Header.Peek("X-Telegram-Bot-Api-Secret-Token")
	if cfg.SecretToken == "" || subtle.ConstantTimeCompare(secret, []byte(cfg.SecretToken)) != 1 {
		return r.SendErrorEnvelope(http.StatusForbidden, "invalid webhook secret", nil, envelope.PermissionError)
	}
	if !rec.Enabled {
		return r.SendEnvelope(true)
	}
	var update telegram.Update
	if err := json.Unmarshal(r.RequestCtx.PostBody(), &update); err != nil {
		return r.SendErrorEnvelope(http.StatusBadRequest, "invalid update", nil, envelope.InputError)
	}
	message := update.Message
	if update.CallbackQuery != nil {
		incoming, ok := update.CallbackQuery.IncomingMessage()
		if !ok {
			return r.SendEnvelope(true)
		}
		message = &incoming
	}
	edited := false
	if update.BusinessMessage != nil {
		message = update.BusinessMessage
	}
	if update.EditedMessage != nil {
		message = update.EditedMessage
		edited = true
	}
	if update.EditedBusinessMessage != nil {
		message = update.EditedBusinessMessage
		edited = true
	}
	if message == nil || !message.Inbound() || (!edited && message.SenderBusinessBot != nil && message.SenderBusinessBot.ID == cfg.BotID) {
		return r.SendEnvelope(true)
	}
	ctx, cancel := context.WithTimeout(app.ctx, 60*time.Second)
	defer cancel()
	if update.CallbackQuery != nil {
		if err := app.telegramClient.AnswerCallbackQuery(ctx, cfg.BotToken, update.CallbackQuery.ID); err != nil {
			app.lo.Warn("error answering telegram callback", "inbox_id", inboxID, "error", err)
		}
	}
	if update.CallbackQuery != nil && strings.HasPrefix(update.CallbackQuery.Data, telegram.RatingCallbackPrefix) {
		if message.Outgoing() {
			return r.SendEnvelope(true)
		}
		rating, _ := strconv.Atoi(strings.TrimPrefix(update.CallbackQuery.Data, telegram.RatingCallbackPrefix))
		original := *message
		original.CallbackID = ""
		err = app.conversation.SubmitTelegramRating(original.SourceID(rec.ID), rating)
	} else if edited {
		formatted, kind := message.FormattedContent()
		err = app.conversation.UpdateTelegramMessageContent(rec.ID, strconv.FormatInt(message.Chat.ID, 10), message.SourceID(rec.ID), formatted, message.Content(), kind, message.EditDate)
	} else {
		err = ingestTelegramMessage(ctx, app, rec, cfg, *message)
	}
	if err != nil {
		app.lo.Error("error processing telegram update", "inbox_id", inboxID, "update_id", update.ID, "error", err)
		return r.SendErrorEnvelope(http.StatusServiceUnavailable, "message could not be stored, retry delivery", nil, envelope.GeneralError)
	}
	return r.SendEnvelope(true)
}

func ingestTelegramMessage(ctx context.Context, app *App, rec imodels.Inbox, cfg telegramChannel.Config, message telegram.Message) error {
	if !message.Inbound() {
		return nil
	}
	identifier := strconv.FormatInt(message.Chat.ID, 10)
	defer telegramChatLocks.lock(identifier)()
	sourceID := message.SourceID(rec.ID)
	if exists, err := app.conversation.MessageExists(sourceID); err != nil {
		return err
	} else if exists {
		if message.Outgoing() || message.CallbackID != "" {
			return nil
		}
		if err := sendTelegramAutomaticReply(app, rec, cfg, sourceID); err != nil {
			app.lo.Error("error sending telegram automatic reply", "inbox_id", rec.ID, "source_id", sourceID, "error", err)
		}
		return nil
	}
	attachments, notice, err := fetchTelegramAttachment(ctx, app, cfg, message)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	contact := umodels.User{Type: umodels.UserTypeContact, FirstName: message.From.FirstName, LastName: message.From.LastName}
	if message.Outgoing() {
		contact.FirstName = message.Chat.FirstName
		contact.LastName = message.Chat.LastName
	}
	contactID, err := app.user.UpsertContactByChannelIdentity(telegramChannel.ChannelTelegram, identifier, &contact)
	if err != nil {
		return err
	}
	attributes := map[string]any{}
	username := message.From.Username
	if message.Outgoing() {
		username = message.Chat.Username
	} else if message.From.LanguageCode != "" {
		attributes["telegram_language"] = message.From.LanguageCode
	}
	if username != "" {
		attributes["telegram_username"] = username
	}
	if len(attributes) > 0 {
		if err := app.user.SaveCustomAttributes(contactID, attributes, false); err != nil {
			return err
		}
	}
	content := message.Content()
	if notice != "" {
		if content != "" {
			content += "\n\n"
		}
		content += notice
	}
	preview := content
	if preview == "" {
		_, kind := message.Attachment()
		preview = "[" + kind + "]"
	}
	id, uuid, isNew, err := app.conversation.FindOrCreateTelegramConversation(contactID, rec.ID, rec.ReopenWindowHours, preview, message.BusinessConnectionID, message.ThreadID)
	if err != nil {
		return err
	}
	msg := cmodels.Message{
		Channel: telegramChannel.ChannelTelegram, ConversationID: id, ConversationUUID: uuid, SenderID: contactID, SenderType: cmodels.SenderTypeContact,
		Type: cmodels.MessageIncoming, Status: cmodels.MessageStatusReceived, InboxID: rec.ID, Content: content, ContentType: cmodels.ContentTypeText,
		SourceID: null.StringFrom(sourceID), Attachments: attachments,
	}
	meta := map[string]any{}
	if message.Contact != nil {
		meta["telegram_contact"] = message.Contact
	}
	if message.Venue != nil {
		meta["telegram_location"] = message.Venue
	} else if message.Location != nil {
		meta["telegram_location"] = telegram.Venue{Location: *message.Location}
	}
	if message.MediaGroupID != "" {
		meta["telegram_media_group_id"] = message.MediaGroupID
	}
	if message.CallbackID != "" {
		meta["telegram_callback"] = true
	}
	if notice == "" {
		msg.Content, msg.ContentType = message.FormattedContent()
	}

	if message.ReplyTo != nil {
		target := *message.ReplyTo
		target.Chat = message.Chat
		target.BusinessConnectionID = message.BusinessConnectionID
		replyUUID, err := app.conversation.GetTelegramReplyUUID(target.SourceID(rec.ID), uuid)
		if err != nil {
			return err
		}
		meta["reply_to"] = map[string]any{"uuid": replyUUID, "content": target.Content()}
	}
	msg.Meta, _ = json.Marshal(meta)
	if message.Outgoing() {
		actor, err := app.user.GetSystemUser()
		if err != nil {
			return err
		}
		msg.Type = cmodels.MessageOutgoing
		msg.Status = cmodels.MessageStatusSent
		msg.SenderID = actor.ID
		msg.SenderType = cmodels.SenderTypeAgent
	}
	if err := app.conversation.ProcessTelegramMessage(msg, isNew); err != nil {
		return err
	}
	if !message.Outgoing() && message.CallbackID == "" {
		if err := sendTelegramAutomaticReply(app, rec, cfg, sourceID); err != nil {
			app.lo.Error("error sending telegram automatic reply", "inbox_id", rec.ID, "source_id", sourceID, "error", err)
		}
	}
	if isNew {
		if err := fetchTelegramAvatar(ctx, app, cfg, contactID, message.Chat.ID); err != nil {
			app.lo.Warn("error fetching telegram avatar", "contact_id", contactID, "error", err)
		}
	}
	return nil
}

func fetchTelegramAttachment(ctx context.Context, app *App, cfg telegramChannel.Config, message telegram.Message) (attachment.Attachments, string, error) {
	file, kind := message.Attachment()
	if file.ID == "" {
		if message.Content() == "" {
			return nil, app.i18n.T("conversation.telegram.unsupportedMessage"), nil
		}
		return nil, "", nil
	}
	maxBytes := min(int64(telegram.MaxDownloadBytes), int64(app.consts.Load().(*constants).MaxFileUploadSizeMB)*1024*1024)
	if file.Size > maxBytes {
		return nil, app.i18n.T("conversation.telegram.fileUnavailable"), nil
	}
	info, body, err := app.telegramClient.Download(ctx, cfg.BotToken, file.ID)
	if err != nil {
		var apiErr *telegram.APIError
		if errors.Is(err, telegram.ErrFileTooLarge) || (errors.As(err, &apiErr) && (apiErr.Code == 400 || apiErr.Code == 404)) {
			return nil, app.i18n.T("conversation.telegram.fileUnavailable"), nil
		}
		return nil, "", err
	}
	if len(body) == 0 || int64(len(body)) > maxBytes {
		return nil, app.i18n.T("conversation.telegram.fileUnavailable"), nil
	}
	name := file.Name
	if name == "" {
		name = filepath.Base(info.Path)
	}
	contentType := file.MIME
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(name))
	}
	if contentType == "" {
		contentType = http.DetectContentType(body)
	}
	if name == "" || name == "." {
		name = defaultMediaFilename(kind, contentType)
	}
	return attachment.Attachments{{Name: name, ContentType: contentType, Content: body, Size: len(body), Disposition: attachment.DispositionAttachment}}, "", nil
}

func fetchTelegramAvatar(ctx context.Context, app *App, cfg telegramChannel.Config, contactID int, telegramID int64) error {
	contact, err := app.user.Get(contactID, "", []string{umodels.UserTypeContact})
	if err != nil {
		return err
	}
	if contact.AvatarURL.Valid && contact.AvatarURL.String != "" {
		return nil
	}
	fileID, err := app.telegramClient.GetProfilePhoto(ctx, cfg.BotToken, telegramID)
	if err != nil || fileID == "" {
		return err
	}
	_, content, err := app.telegramClient.Download(ctx, cfg.BotToken, fileID)
	if err != nil {
		return err
	}
	media, err := saveUserAvatar(app, contactID, "avatar.jpg", "image/jpeg", bytes.NewReader(content), true /** private **/)
	if err != nil {
		return err
	}
	app.conversation.BroadcastContactUpdate(contactID, map[string]any{"avatar_url": app.media.GetSignedURL(media.UUID)})
	return nil
}

func telegramCallbackURL(root string, inboxID int) string {
	if root == "" {
		return ""
	}
	return fmt.Sprintf("%s/webhooks/telegram/%d", strings.TrimRight(root, "/"), inboxID)
}
