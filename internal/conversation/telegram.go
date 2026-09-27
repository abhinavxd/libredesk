package conversation

import (
	"database/sql"
	"encoding/json"
	"errors"
	"html"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	telegramChannel "github.com/abhinavxd/libredesk/internal/inbox/channel/telegram"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	mmodels "github.com/abhinavxd/libredesk/internal/media/models"
	"github.com/abhinavxd/libredesk/internal/telegram"
)

type TelegramAutoReplyState struct {
	ConversationUUID string `db:"conversation_uuid"`
	ContactID        int    `db:"contact_id"`
	HasOutgoing      bool   `db:"has_outgoing"`
	RecentReply      bool   `db:"recent_reply"`
	AwaySentToday    bool   `db:"away_sent_today"`
}

func (m *Manager) ProcessTelegramMessage(msg models.Message, isNewConversation bool) error {
	if err := m.uploadMessageAttachments(&msg); err != nil {
		return err
	}
	if err := m.InsertMessage(&msg); err != nil {
		return err
	}
	if msg.Type == models.MessageIncoming {
		return m.ProcessIncomingMessageHooks(msg.ConversationUUID, isNewConversation)
	}
	return nil
}

func (m *Manager) UpdateTelegramMessage(sourceID, content string, editedAt int64) error {
	return m.UpdateTelegramMessageContent(sourceID, content, content, models.ContentTypeText, editedAt)
}

func (m *Manager) UpdateTelegramMessageContent(sourceID, content, text, kind string, editedAt int64) error {
	var message struct {
		UUID             string         `db:"uuid"`
		ConversationUUID string         `db:"conversation_uuid"`
		LastMessage      sql.NullString `db:"last_message"`
		LastInteraction  sql.NullString `db:"last_interaction"`
	}
	if err := m.q.UpdateTelegramMessage.Get(&message, sourceID, content, editedAt, text, kind); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	m.BroadcastMessageUpdate(message.ConversationUUID, message.UUID, map[string]any{"content": content, "text_content": text, "content_type": kind})
	m.BroadcastConversationUpdate(message.ConversationUUID, map[string]any{"last_message": message.LastMessage.String, "last_interaction": message.LastInteraction.String})
	return nil
}

func (m *Manager) FindOrCreateTelegramConversation(contactID, inboxID, reopenHours int, preview, businessID string, threadID int64) (int, string, bool, error) {
	var record struct {
		ID   int    `db:"id"`
		UUID string `db:"uuid"`
	}
	err := m.q.GetTelegramConversation.Get(&record, contactID, inboxID, reopenHours, businessID, threadID)
	if err == nil {
		return record.ID, record.UUID, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, "", false, err
	}
	meta := map[string]any{"telegram": telegramChannel.SendMeta{BusinessConnectionID: businessID, ThreadID: threadID}}
	id, uuid, err := m.CreateConversation(contactID, inboxID, preview, time.Now(), "", false, meta, nil, 0, 0)
	return id, uuid, true, err
}

func (m *Manager) RecordTelegramSend(messageUUID string, sourceIDs []string) error {
	if messageUUID == "" || len(sourceIDs) == 0 {
		return nil
	}
	ids, _ := json.Marshal(sourceIDs)
	_, err := m.q.RecordTelegramSend.Exec(messageUUID, sourceIDs[0], ids)
	return err
}

func (m *Manager) SubmitTelegramRating(sourceID string, rating int) error {
	if rating < 1 || rating > 5 {
		return nil
	}
	var message struct {
		UUID             string `db:"uuid"`
		ConversationUUID string `db:"conversation_uuid"`
	}
	if err := m.q.SubmitTelegramRating.Get(&message, sourceID, rating); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	m.BroadcastMessageUpdate(message.ConversationUUID, message.UUID, map[string]any{"meta": map[string]any{"csat_submitted": true, "submitted_rating": rating}})
	return nil
}

func (m *Manager) prepareTelegramOutbound(inboxRecord imodels.Inbox, conversationUUID, content string, media []mmodels.Media, meta map[string]any) error {
	var conv struct {
		InboxID   int             `db:"inbox_id"`
		ContactID int             `db:"contact_id"`
		Meta      json.RawMessage `db:"meta"`
	}
	if err := m.q.GetTelegramConversationTarget.Get(&conv, conversationUUID); err != nil {
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if conv.InboxID != inboxRecord.ID {
		return envelope.NewError(envelope.InputError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	identity, err := m.userStore.GetChannelIdentity(conv.ContactID, telegramChannel.ChannelTelegram)
	if err != nil {
		return err
	}
	chatID, err := strconv.ParseInt(identity, 10, 64)
	if err != nil || chatID <= 0 {
		return envelope.NewError(envelope.InputError, m.i18n.T("conversation.telegram.error.noChat"), nil)
	}
	if len(media) > telegram.MaxAlbumSize {
		return envelope.NewError(envelope.InputError, m.i18n.T("conversation.telegram.error.albumSize"), nil)
	}
	for _, file := range media {
		if file.Size > telegram.MaxUploadBytes {
			return envelope.NewError(envelope.InputError, m.i18n.T("conversation.telegram.error.fileTooLarge"), nil)
		}
	}
	var buttons []telegram.Button
	if value, ok := meta["telegram_buttons"]; ok {
		encoded, err := json.Marshal(value)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(encoded, &buttons); err != nil {
			return envelope.NewError(envelope.InputError, m.i18n.T("conversation.telegram.error.buttons"), nil)
		}
		if err := telegram.ValidateButtons(buttons); err != nil || (len(media) > 1 && len(buttons) > 0) {
			return envelope.NewError(envelope.InputError, m.i18n.T("conversation.telegram.error.buttons"), nil)
		}
	}
	_, text := telegram.FormatHTML(content)
	if strings.TrimSpace(text) == "" && len(media) == 0 {
		return envelope.NewError(envelope.InputError, m.i18n.T("globals.messages.messageContentRequired"), nil)
	}
	limit := telegram.MaxTextLength
	if len(media) > 0 {
		limit = telegram.MaxCaptionLength
	}
	if utf8.RuneCountInString(text) > limit {
		return envelope.NewError(envelope.InputError, m.i18n.Ts("conversation.telegram.error.tooLong", "limit", strconv.Itoa(limit)), nil)
	}
	var routing struct {
		Telegram telegramChannel.SendMeta `json:"telegram"`
	}
	if err := json.Unmarshal(conv.Meta, &routing); err != nil {
		return err
	}
	routing.Telegram.ChatID = chatID
	routing.Telegram.Buttons = buttons
	if meta["is_csat"] == true {
		routing.Telegram.CSATUUID, _ = meta["csat_uuid"].(string)
	}
	if replyUUID, ok := meta["reply_to_message_uuid"].(string); ok && replyUUID != "" {
		var target struct {
			SourceID string `db:"source_id"`
			Content  string `db:"content"`
		}
		if err := m.q.GetTelegramReplyTarget.Get(&target, replyUUID, conversationUUID); err != nil {
			return envelope.NewError(envelope.InputError, m.i18n.T("conversation.telegram.error.replyTarget"), nil)
		}
		id, err := telegram.MessageIDFromSource(target.SourceID, inboxRecord.ID, chatID, routing.Telegram.BusinessConnectionID)
		if err != nil {
			return envelope.NewError(envelope.InputError, m.i18n.T("conversation.telegram.error.replyTarget"), nil)
		}
		routing.Telegram.ReplyToMessageID = id
		meta["reply_to"] = map[string]any{"uuid": replyUUID, "content": target.Content}
	}
	encoded, _ := json.Marshal(routing.Telegram)
	meta["telegram"] = json.RawMessage(encoded)
	return nil
}

func (m *Manager) GetTelegramReplyUUID(sourceID, conversationUUID string) (string, error) {
	var uuid string
	err := m.q.GetTelegramReplyUUID.Get(&uuid, sourceID, conversationUUID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return uuid, err
}

func (m *Manager) TelegramCSATButtons(uuid string) ([]telegram.Button, error) {
	root, err := m.settingsStore.GetAppRootURL()
	if err != nil {
		return nil, err
	}
	buttons := telegram.RatingButtons()
	buttons = append(buttons, telegram.Button{Text: m.i18n.T("globals.messages.additionalFeedback"), URL: m.csatStore.MakePublicURL(root, uuid)})
	return buttons, nil
}

func (m *Manager) sendTelegramCSAT(actorID int, conv models.Conversation, uuid string) error {
	rec, err := m.inboxStore.GetDBRecord(conv.InboxID)
	if err != nil {
		return err
	}
	var cfg telegramChannel.Config
	if err := json.Unmarshal(rec.Config, &cfg); err != nil {
		return err
	}
	content := cfg.CSATMessage
	if strings.TrimSpace(content) == "" {
		content = m.i18n.T("csat.rateYourInteraction")
	}
	_, err = m.QueueReply(nil, conv.InboxID, actorID, conv.ContactID, conv.UUID,
		strings.ReplaceAll(html.EscapeString(content), "\n", "<br>"), nil, nil, nil, map[string]any{"is_csat": true, "is_automated": true, "csat_uuid": uuid, "telegram_buttons": telegram.RatingButtons()})
	return err
}

func (m *Manager) GetTelegramAutoReplyState(sourceID, timezone string) (TelegramAutoReplyState, error) {
	var state TelegramAutoReplyState
	err := m.q.GetTelegramAutoReplyState.Get(&state, sourceID, timezone)
	if errors.Is(err, sql.ErrNoRows) {
		return state, nil
	}
	return state, err
}
