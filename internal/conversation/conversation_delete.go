package conversation

import (
	"github.com/abhinavxd/libredesk/internal/envelope"
)

// DeletedAttachment identifies a stored file whose media row was removed along with its conversation.
// Media rows have no foreign key to conversation_messages, so the caller has to drop the files itself.
type DeletedAttachment struct {
	UUID        string `db:"uuid"`
	ContentType string `db:"content_type"`
}

// ConversationDeletion is what a deleted conversation leaves behind outside the database:
// the stored attachment files and the Message-IDs of the mails that created it.
type ConversationDeletion struct {
	Attachments      []DeletedAttachment
	IncomingSourceID []string
}

// DeleteConversationWithData deletes a conversation and everything that belongs to it.
//
// Messages, activity, participants, mentions, last-seen rows, drafts, tags, CSAT responses,
// applied SLAs, AI events and notifications all cascade from conversations, so a single delete
// clears them. Media rows are polymorphic and do not cascade, so they are removed in the same
// transaction and the files they point at are returned for the caller to delete from the store.
func (m *Manager) DeleteConversationWithData(conversationID int, uuid string) (ConversationDeletion, error) {
	var out ConversationDeletion

	tx, err := m.db.Beginx()
	if err != nil {
		m.lo.Error("error beginning conversation delete transaction", "uuid", uuid, "error", err)
		return out, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	defer tx.Rollback()

	if err := tx.Stmtx(m.q.GetIncomingMessageSourceIDs).Select(&out.IncomingSourceID, conversationID); err != nil {
		m.lo.Error("error fetching conversation message source ids", "uuid", uuid, "error", err)
		return out, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if err := tx.Stmtx(m.q.GetConversationAttachments).Select(&out.Attachments, conversationID); err != nil {
		m.lo.Error("error fetching conversation attachments", "uuid", uuid, "error", err)
		return out, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	// Must run before the conversation delete, it resolves the messages that are about to cascade away.
	if _, err := tx.Stmtx(m.q.DeleteConversationAttachments).Exec(conversationID); err != nil {
		m.lo.Error("error deleting conversation attachments", "uuid", uuid, "error", err)
		return out, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	res, err := tx.Stmtx(m.q.DeleteConversation).Exec(uuid)
	if err != nil {
		m.lo.Error("error deleting conversation", "uuid", uuid, "error", err)
		return out, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return out, envelope.NewError(envelope.NotFoundError, m.i18n.Ts("globals.messages.notFound", "name", m.i18n.Ts("globals.terms.conversation")), nil)
	}

	if err := tx.Commit(); err != nil {
		m.lo.Error("error committing conversation delete", "uuid", uuid, "error", err)
		return out, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	m.lo.Info("deleted conversation", "uuid", uuid, "attachments", len(out.Attachments), "incoming_mails", len(out.IncomingSourceID))
	return out, nil
}
