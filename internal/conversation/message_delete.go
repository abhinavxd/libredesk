package conversation

import "github.com/abhinavxd/libredesk/internal/envelope"

// DeleteMessage deletes a private note by UUID.
// Only messages with private=true can be deleted (enforced by the SQL query).
func (m *Manager) DeleteMessage(uuid string) error {
	m.lo.Info("deleting message", "uuid", uuid)
	res, err := m.q.DeleteMessage.Exec(uuid)
	if err != nil {
		m.lo.Error("error deleting message", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorDeleting", "name", m.i18n.Ts("globals.terms.message")), nil)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return envelope.NewError(envelope.NotFoundError, m.i18n.Ts("globals.messages.notFound", "name", m.i18n.Ts("globals.terms.message")), nil)
	}
	return nil
}
