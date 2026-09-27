package conversation

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/livechat/proactive"
)

func (m *Manager) CreateProactiveConversation(delivery proactive.Delivery, contactID int, reply string, attrs, meta map[string]any, maxConversations int, window time.Duration) (models.Message, error) {
	var snapshot proactive.Snapshot
	if err := json.Unmarshal(delivery.Snapshot, &snapshot); err != nil {
		return models.Message{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if attrs == nil {
		attrs = map[string]any{}
	}
	attrsJSON, err := json.Marshal(attrs)
	if err != nil {
		return models.Message{}, m.proactiveError(err)
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return models.Message{}, m.proactiveError(err)
	}
	tx, err := m.db.Beginx()
	if err != nil {
		return models.Message{}, m.proactiveError(err)
	}
	defer tx.Rollback()
	var existing string
	if err := tx.Stmtx(m.q.LockCampaignDelivery).Get(&existing, delivery.ID); err != nil {
		return models.Message{}, m.proactiveError(err)
	}
	if existing != "" {
		return models.Message{ConversationUUID: existing}, nil
	}
	var id int
	var uuid string
	now := time.Now()
	if err := tx.Stmtx(m.q.InsertConversation).QueryRow(contactID, models.StatusOpen, delivery.InboxID, reply, now, "", "", false, metaJSON, attrsJSON, now.Add(-window), maxConversations, m.subjectRefFormat).Scan(&id, &uuid); err != nil {
		if err == sql.ErrNoRows {
			return models.Message{}, envelope.NewError(envelope.RateLimitError, m.i18n.T("globals.messages.tooManyRequests"), nil)
		}
		return models.Message{}, m.proactiveError(err)
	}
	if _, err := tx.Stmtx(m.q.AssignProactiveTeam).Exec(id, snapshot.TeamID); err != nil {
		return models.Message{}, m.proactiveError(err)
	}
	invitationMeta, _ := json.Marshal(map[string]any{"proactive_sender": snapshot.Sender, "campaign_id": delivery.CampaignID})
	messages := []models.Message{
		{Type: models.MessageOutgoing, Status: models.MessageStatusSent, SenderID: snapshot.SenderID, SenderType: models.SenderTypeAgent, Content: snapshot.Message, Meta: invitationMeta},
		{Type: models.MessageIncoming, Status: models.MessageStatusReceived, SenderID: contactID, SenderType: models.SenderTypeContact, Content: reply},
	}
	inlineUUIDs := make([][]string, len(messages))
	for i := range messages {
		messages[i].ConversationID = id
		messages[i].ConversationUUID = uuid
		messages[i].ContentType = models.ContentTypeText
		uuids, err := m.InsertMessageTx(tx, &messages[i])
		if err != nil {
			return models.Message{}, err
		}
		inlineUUIDs[i] = uuids
	}
	if _, err := tx.Stmtx(m.q.CompleteCampaignDelivery).Exec(delivery.ID, uuid, contactID); err != nil {
		return models.Message{}, m.proactiveError(err)
	}
	if err := tx.Commit(); err != nil {
		return models.Message{}, m.proactiveError(err)
	}
	if item, err := m.GetConversationListItem(uuid); err == nil {
		m.BroadcastNewConversation(&item)
	}
	for i := range messages {
		m.AfterMessageInsert(&messages[i], inlineUUIDs[i])
	}
	return messages[1], nil
}

func (m *Manager) proactiveError(err error) error {
	m.lo.Error("error creating proactive conversation", "error", err)
	return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.errorSendingMessage"), nil)
}

func proactiveSender(meta json.RawMessage) string {
	var snapshot struct {
		Sender string `json:"proactive_sender"`
	}
	_ = json.Unmarshal(meta, &snapshot)
	return snapshot.Sender
}
