package conversation

import (
	"database/sql"
	"errors"
	"time"

	"github.com/abhinavxd/libredesk/internal/envelope"
)

// DeletePrivateMessage soft-deletes a private note, unlinks its media for GC, and returns the tombstone text.
func (m *Manager) DeletePrivateMessage(conversationUUID, messageUUID string) (string, error) {
	m.lo.Info("deleting private note", "conversation_uuid", conversationUUID, "message_uuid", messageUUID)

	var previewUpdated bool
	deletedPreview := m.i18n.T("conversation.privateNoteDeleted")
	if err := m.q.DeletePrivateMessage.Get(&previewUpdated, messageUUID, conversationUUID, deletedPreview); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", envelope.NewError(envelope.NotFoundError, m.i18n.Ts("globals.messages.notFound", "name", m.i18n.Ts("globals.terms.message")), nil)
		}
		m.lo.Error("error deleting private note", "error", err)
		return "", envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	m.BroadcastMessageUpdate(conversationUUID, messageUUID, map[string]any{
		"content":      deletedPreview,
		"text_content": deletedPreview,
		"meta":         map[string]any{"deleted_at": time.Now()},
	})
	if previewUpdated {
		m.BroadcastConversationUpdate(conversationUUID, map[string]any{"last_message": deletedPreview})
	}
	return deletedPreview, nil
}
