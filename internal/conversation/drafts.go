package conversation

import (
	"context"
	"encoding/json"
	"time"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
)

// UpsertConversationDraft saves or updates a draft for a conversation.
func (m *Manager) UpsertConversationDraft(conversationID, userID int, content string, meta json.RawMessage) (models.ConversationDraft, error) {
	var draft models.ConversationDraft

	if err := m.q.UpsertConversationDraft.Get(&draft, conversationID, userID, content, meta); err != nil {
		m.lo.Error("error upserting conversation draft", "conversation_id", conversationID, "user_id", userID, "error", err)
		return draft, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "draft"), nil)
	}

	return draft, nil
}

// GetAllUserDrafts retrieves all drafts for a user.
func (m *Manager) GetAllUserDrafts(userID int) ([]models.ConversationDraft, error) {
	var drafts = make([]models.ConversationDraft, 0)
	if err := m.q.GetAllUserDrafts.Select(&drafts, userID); err != nil {
		m.lo.Error("error fetching user drafts", "user_id", userID, "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "drafts"), nil)
	}
	return drafts, nil
}

// DeleteConversationDraft deletes a draft for a conversation by ID or UUID.
func (m *Manager) DeleteConversationDraft(conversationID int, uuid string, userID int) error {
	var uuidParam any
	if uuid != "" {
		uuidParam = uuid
	}

	if _, err := m.q.DeleteConversationDraft.Exec(conversationID, uuidParam, userID); err != nil {
		m.lo.Error("error deleting conversation draft", "conversation_id", conversationID, "uuid", uuid, "user_id", userID, "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorDeleting", "name", "draft"), nil)
	}

	return nil
}

// DeleteStaleDrafts deletes drafts older than the specified retention period.
func (m *Manager) DeleteStaleDrafts(ctx context.Context, retentionPeriod time.Duration) error {
	cutoff := time.Now().Add(-retentionPeriod)
	res, err := m.q.DeleteStaleDrafts.ExecContext(ctx, cutoff)
	if err != nil {
		m.lo.Error("error deleting stale drafts", "error", err)
		return err
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected > 0 {
		m.lo.Info("deleted stale drafts", "count", rowsAffected)
	}

	return nil
}
