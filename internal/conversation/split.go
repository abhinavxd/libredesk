package conversation

import (
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/lib/pq"
)

const maxSplitMessages = 200

// SplitConversation moves selected messages onto a new child ticket.
func (m *Manager) SplitConversation(sourceUUID string, messageUUIDs []string, subject string, actor umodels.User) (models.Conversation, error) {
	if len(messageUUIDs) == 0 {
		return models.Conversation{}, envelope.NewError(envelope.InputError, m.i18n.T("conversation.split.messagesRequired"), nil)
	}
	if len(messageUUIDs) > maxSplitMessages {
		return models.Conversation{}, envelope.NewError(envelope.InputError, m.i18n.T("conversation.split.tooMany"), nil)
	}

	source, err := m.followMergedConversation(sourceUUID)
	if err != nil {
		return models.Conversation{}, err
	}

	seen := map[string]struct{}{}
	var uuids []string
	for _, raw := range messageUUIDs {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		msg, err := m.GetMessage(id)
		if err != nil {
			return models.Conversation{}, envelope.NewError(envelope.InputError, m.i18n.T("conversation.split.invalidMessage"), nil)
		}
		if msg.ConversationID != source.ID || msg.Type == "activity" {
			return models.Conversation{}, envelope.NewError(envelope.InputError, m.i18n.T("conversation.split.invalidMessage"), nil)
		}
		seen[id] = struct{}{}
		uuids = append(uuids, id)
	}
	if len(uuids) == 0 {
		return models.Conversation{}, envelope.NewError(envelope.InputError, m.i18n.T("conversation.split.messagesRequired"), nil)
	}

	subject = strings.TrimSpace(subject)
	if subject == "" {
		parentSubj := strings.TrimSpace(source.Subject.String)
		if parentSubj != "" {
			subject = "Split: " + parentSubj
		} else {
			subject = "Split from #" + source.ReferenceNumber
		}
	}

	_, newUUID, err := m.CreateConversation(source.ContactID, source.InboxID, subject, time.Now(), subject, true, nil, nil, 0, 0)
	if err != nil {
		return models.Conversation{}, err
	}
	if _, err := m.q.SetConversationParent.Exec(newUUID, source.UUID, models.OriginSplit); err != nil {
		m.lo.Error("error linking split conversation", "error", err)
		_ = m.DeleteConversation(newUUID)
		return models.Conversation{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	child, err := m.GetConversation(0, newUUID, "")
	if err != nil {
		return models.Conversation{}, err
	}

	tx, err := m.db.Beginx()
	if err != nil {
		_ = m.DeleteConversation(newUUID)
		return models.Conversation{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	defer tx.Rollback()

	arr := pq.Array(uuids)
	if _, err := tx.Stmtx(m.q.SplitMoveMessages).Exec(source.ID, child.ID, arr); err != nil {
		m.lo.Error("error moving messages during split", "error", err)
		_ = m.DeleteConversation(newUUID)
		return models.Conversation{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if _, err := tx.Stmtx(m.q.SplitMoveMentions).Exec(child.ID, arr); err != nil {
		m.lo.Error("error moving mentions during split", "error", err)
		_ = m.DeleteConversation(newUUID)
		return models.Conversation{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if _, err := tx.Stmtx(m.q.SplitRefreshLastMessage).Exec(source.ID); err != nil {
		m.lo.Error("error refreshing source last message after split", "error", err)
	}
	if _, err := tx.Stmtx(m.q.SplitRefreshLastMessage).Exec(child.ID); err != nil {
		m.lo.Error("error refreshing split last message", "error", err)
	}
	if err := tx.Commit(); err != nil {
		_ = m.DeleteConversation(newUUID)
		return models.Conversation{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	sourceRef := source.ReferenceNumber
	if sourceRef == "" {
		sourceRef = source.UUID
	}
	childRef := child.ReferenceNumber
	if childRef == "" {
		childRef = child.UUID
	}
	_ = m.InsertConversationActivity(models.ActivitySplitFrom, source.UUID, childRef, actor)
	_ = m.InsertConversationActivity(models.ActivitySplitInto, child.UUID, sourceRef, actor)

	if source.AssignedTeamID.Valid {
		_ = m.UpdateConversationTeamAssignee(child.UUID, int(source.AssignedTeamID.Int), actor)
	}
	if source.AssignedUserID.Valid {
		_ = m.UpdateConversationUserAssignee(child.UUID, int(source.AssignedUserID.Int), actor)
	}

	m.BroadcastConversationUpdate(source.UUID, map[string]any{"split_into_uuid": child.UUID})
	return m.GetConversation(0, child.UUID, "")
}
