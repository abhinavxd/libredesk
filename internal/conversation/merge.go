package conversation

import (
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
)

const maxMergeFollow = 8

// MergeConversations moves the source ticket into the target and soft-closes the source.
// The target keeps its contact, assignee, inbox, SLA, and CSAT.
func (m *Manager) MergeConversations(sourceUUID, targetUUID string, actor umodels.User) (models.Conversation, error) {
	if sourceUUID == "" || targetUUID == "" || sourceUUID == targetUUID {
		return models.Conversation{}, envelope.NewError(envelope.InputError, m.i18n.T("conversation.merge.sameTicket"), nil)
	}

	source, err := m.GetConversation(0, sourceUUID, "")
	if err != nil {
		return models.Conversation{}, err
	}
	target, err := m.resolveMergeTarget(targetUUID)
	if err != nil {
		return models.Conversation{}, err
	}
	if source.UUID == target.UUID {
		return models.Conversation{}, envelope.NewError(envelope.InputError, m.i18n.T("conversation.merge.sameTicket"), nil)
	}
	if source.MergedIntoUUID.Valid {
		return models.Conversation{}, envelope.NewError(envelope.InputError, m.i18n.T("conversation.merge.alreadyMerged"), nil)
	}

	tx, err := m.db.Beginx()
	if err != nil {
		m.lo.Error("error beginning merge transaction", "error", err)
		return models.Conversation{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	defer tx.Rollback()

	if _, err := tx.Stmtx(m.q.MergeMoveMessages).Exec(source.ID, target.ID); err != nil {
		m.lo.Error("error moving messages during merge", "error", err)
		return models.Conversation{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if _, err := tx.Stmtx(m.q.MergeCopyTags).Exec(source.ID, target.ID); err != nil {
		m.lo.Error("error copying tags during merge", "error", err)
		return models.Conversation{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if _, err := tx.Stmtx(m.q.MergeCopyParticipants).Exec(source.ID, target.ID); err != nil {
		m.lo.Error("error copying participants during merge", "error", err)
		return models.Conversation{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if _, err := tx.Stmtx(m.q.MergeRefreshTargetLastMessage).Exec(source.ID, target.ID); err != nil {
		m.lo.Error("error refreshing last message during merge", "error", err)
		return models.Conversation{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if _, err := tx.Stmtx(m.q.MergeCloseSource).Exec(source.UUID, target.UUID); err != nil {
		m.lo.Error("error closing source conversation during merge", "error", err)
		return models.Conversation{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if err := tx.Commit(); err != nil {
		m.lo.Error("error committing merge transaction", "error", err)
		return models.Conversation{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	sourceRef := source.ReferenceNumber
	if sourceRef == "" {
		sourceRef = source.UUID
	}
	targetRef := target.ReferenceNumber
	if targetRef == "" {
		targetRef = target.UUID
	}

	_ = m.InsertConversationActivity(models.ActivityMergedFrom, target.UUID, sourceRef, actor)
	_ = m.InsertConversationActivity(models.ActivityMergedInto, source.UUID, targetRef, actor)

	if err := m.UpdateConversationStatus(source.UUID, 0, models.StatusResolved, "", actor); err != nil {
		m.lo.Error("error resolving source after merge", "error", err, "source", source.UUID)
	}

	merged, err := m.GetConversation(0, target.UUID, "")
	if err != nil {
		return models.Conversation{}, err
	}
	m.BroadcastConversationUpdate(source.UUID, map[string]any{
		"merged_into_uuid": target.UUID,
		"status":           models.StatusResolved,
	})
	m.BroadcastConversationUpdate(target.UUID, map[string]any{
		"merged_from_uuid": source.UUID,
	})
	return merged, nil
}

func (m *Manager) resolveMergeTarget(uuid string) (models.Conversation, error) {
	seen := map[string]struct{}{}
	for i := 0; i < maxMergeFollow; i++ {
		conv, err := m.GetConversation(0, uuid, "")
		if err != nil {
			return models.Conversation{}, err
		}
		if !conv.MergedIntoUUID.Valid || conv.MergedIntoUUID.String == "" {
			return conv, nil
		}
		if _, ok := seen[conv.UUID]; ok {
			return models.Conversation{}, envelope.NewError(envelope.InputError, m.i18n.T("conversation.merge.sameTicket"), nil)
		}
		seen[conv.UUID] = struct{}{}
		uuid = conv.MergedIntoUUID.String
	}
	return models.Conversation{}, envelope.NewError(envelope.InputError, m.i18n.T("conversation.merge.sameTicket"), nil)
}

func (m *Manager) followMergedConversation(uuid string) (models.Conversation, error) {
	conv, err := m.GetConversation(0, uuid, "")
	if err != nil {
		return conv, err
	}
	if !conv.MergedIntoUUID.Valid || conv.MergedIntoUUID.String == "" {
		return conv, nil
	}
	target, err := m.resolveMergeTarget(conv.MergedIntoUUID.String)
	if err != nil {
		return conv, err
	}
	return target, nil
}
