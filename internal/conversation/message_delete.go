package conversation

import "github.com/abhinavxd/libredesk/internal/envelope"

// DeletePrivateMessage deletes a private note by UUID, scoped to the given
// conversation. Only messages with private=true can be deleted (enforced by the
// SQL query); sent/incoming messages are protected. The conversation scope
// prevents deleting a note from a different conversation than the caller's.
// Returns a NotFound error if no matching private note exists.
func (m *Manager) DeletePrivateMessage(conversationUUID, messageUUID string) error {
	m.lo.Info("deleting private note", "conversation_uuid", conversationUUID, "message_uuid", messageUUID)
	res, err := m.q.DeletePrivateMessage.Exec(messageUUID, conversationUUID)
	if err != nil {
		m.lo.Error("error deleting private note", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return envelope.NewError(envelope.NotFoundError, m.i18n.Ts("globals.messages.notFound", "name", m.i18n.Ts("globals.terms.message")), nil)
	}
	return nil
}
