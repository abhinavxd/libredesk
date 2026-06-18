package conversation

import "github.com/abhinavxd/libredesk/internal/envelope"

// DeletePrivateMessage deletes a private note by UUID. Only messages with
// private=true can be deleted (enforced by the SQL query); sent/incoming
// messages are protected. Returns a NotFound error if no private note matches.
func (m *Manager) DeletePrivateMessage(uuid string) error {
	m.lo.Info("deleting private note", "uuid", uuid)
	res, err := m.q.DeletePrivateMessage.Exec(uuid)
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
