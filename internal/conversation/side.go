package conversation

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/inbox"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/lib/pq"
)

func (m *Manager) ListSideConversations(conversationUUID string) ([]cmodels.SideConversation, error) {
	conv, err := m.GetConversation(0, conversationUUID, "")
	if err != nil {
		return nil, err
	}
	out := make([]cmodels.SideConversation, 0)
	if err := m.q.ListSideConversations.Select(&out, conv.ID); err != nil && !errors.Is(err, sql.ErrNoRows) {
		m.lo.Error("error listing side conversations", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	for i := range out {
		msgs, err := m.listSideMessages(out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].Messages = msgs
	}
	return out, nil
}

func (m *Manager) CreateSideConversation(conversationUUID string, actorID int, to []string, subject, body string) (cmodels.SideConversation, error) {
	var empty cmodels.SideConversation
	conv, err := m.GetConversation(0, conversationUUID, "")
	if err != nil {
		return empty, err
	}
	to = stringutil.RemoveEmpty(to)
	if len(to) == 0 || strings.TrimSpace(body) == "" {
		return empty, envelope.NewError(envelope.InputError, m.i18n.T("globals.messages.required"), nil)
	}
	if subject == "" {
		subject = conv.Subject.String
	}
	if err := m.q.InsertSideConversation.Get(&empty, conv.ID, subject, pq.Array(to), actorID); err != nil {
		m.lo.Error("error creating side conversation", "error", err)
		return empty, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if err := m.appendAndSendSideMessage(empty, actorID, "outgoing", body); err != nil {
		return empty, err
	}
	empty.Messages, _ = m.listSideMessages(empty.ID)
	return empty, nil
}

func (m *Manager) ReplySideConversation(sideUUID string, actorID int, body string) (cmodels.SideConversation, error) {
	var side cmodels.SideConversation
	if err := m.q.GetSideConversation.Get(&side, sideUUID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return side, envelope.NewError(envelope.NotFoundError, m.i18n.T("globals.messages.notFound"), nil)
		}
		return side, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if strings.TrimSpace(body) == "" {
		return side, envelope.NewError(envelope.InputError, m.i18n.T("globals.messages.required"), nil)
	}
	if err := m.appendAndSendSideMessage(side, actorID, "outgoing", body); err != nil {
		return side, err
	}
	side.Messages, _ = m.listSideMessages(side.ID)
	return side, nil
}

func (m *Manager) AppendInboundSideMessage(sideUUID string, senderID int, body string) error {
	var side cmodels.SideConversation
	if err := m.q.GetSideConversation.Get(&side, sideUUID); err != nil {
		return err
	}
	var msg cmodels.SideMessage
	return m.q.InsertSideMessage.Get(&msg, side.ID, senderID, "incoming", body, cmodels.ContentTypeHTML, nil)
}

func (m *Manager) appendAndSendSideMessage(side cmodels.SideConversation, senderID int, direction, body string) error {
	var msg cmodels.SideMessage
	if err := m.q.InsertSideMessage.Get(&msg, side.ID, senderID, direction, body, cmodels.ContentTypeHTML, nil); err != nil {
		m.lo.Error("error inserting side message", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	inboxID, fromAddr, err := m.sideEmailInbox(side.ConversationID)
	if err != nil {
		return err
	}
	inb, err := m.inboxStore.Get(inboxID)
	if err != nil {
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if inb.Channel() != inbox.ChannelEmail {
		return envelope.NewError(envelope.InputError, m.i18n.T("status.disabledInbox"), nil)
	}
	base := inb.ReplyToAddress()
	if base == "" {
		base = fromAddr
	}
	if addr, err := stringutil.ExtractEmail(base); err == nil {
		base = addr
	}
	replyTo := buildSidePlusAddress(base, side.UUID)
	return inb.Send(cmodels.OutboundMessage{
		From:    fromAddr,
		To:      []string(side.Recipients),
		Subject: side.Subject,
		Content: body,
		ReplyTo: replyTo,
	})
}

func (m *Manager) sideEmailInbox(conversationID int) (int, string, error) {
	conv, err := m.GetConversation(conversationID, "", "")
	if err != nil {
		return 0, "", err
	}
	rec, err := m.inboxStore.GetDBRecord(conv.InboxID)
	if err != nil {
		return 0, "", envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	inboxID := rec.ID
	if rec.Channel != inbox.ChannelEmail {
		if !rec.LinkedEmailInboxID.Valid {
			return 0, "", envelope.NewError(envelope.InputError, m.i18n.T("status.disabledInbox"), nil)
		}
		inboxID = rec.LinkedEmailInboxID.Int
		rec, err = m.inboxStore.GetDBRecord(inboxID)
		if err != nil {
			return 0, "", envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
		}
	}
	return inboxID, rec.From, nil
}

func (m *Manager) listSideMessages(sideID int) ([]cmodels.SideMessage, error) {
	out := make([]cmodels.SideMessage, 0)
	if err := m.q.ListSideMessages.Select(&out, sideID); err != nil && !errors.Is(err, sql.ErrNoRows) {
		m.lo.Error("error listing side messages", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return out, nil
}

func buildSidePlusAddress(email, uuid string) string {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 {
		return email
	}
	return fmt.Sprintf("%s+side-%s@%s", parts[0], uuid, parts[1])
}
