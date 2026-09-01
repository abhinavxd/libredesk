package conversation

import (
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
)

const maxParentFollow = 8

// CreateLinkedConversation opens a child or follow-up ticket on the same contact and inbox.
func (m *Manager) CreateLinkedConversation(parentUUID, origin, subject string, actor umodels.User) (models.Conversation, error) {
	if origin != models.OriginChild && origin != models.OriginFollowUp {
		return models.Conversation{}, envelope.NewError(envelope.InputError, m.i18n.T("conversation.related.invalidOrigin"), nil)
	}
	parent, err := m.followMergedConversation(parentUUID)
	if err != nil {
		return models.Conversation{}, err
	}
	if err := m.ensureNoParentCycle(parent.UUID); err != nil {
		return models.Conversation{}, err
	}

	subject = strings.TrimSpace(subject)
	if subject == "" {
		parentSubj := strings.TrimSpace(parent.Subject.String)
		if origin == models.OriginFollowUp {
			if parentSubj != "" {
				subject = "Follow-up: " + parentSubj
			} else {
				subject = "Follow-up of #" + parent.ReferenceNumber
			}
		} else if parentSubj != "" {
			subject = "Child: " + parentSubj
		} else {
			subject = "Child of #" + parent.ReferenceNumber
		}
	}

	_, uuid, err := m.CreateConversation(parent.ContactID, parent.InboxID, subject, time.Now(), subject, true, nil, nil, 0, 0)
	if err != nil {
		return models.Conversation{}, err
	}
	if _, err := m.q.SetConversationParent.Exec(uuid, parent.UUID, origin); err != nil {
		m.lo.Error("error linking conversation to parent", "error", err, "parent", parent.UUID, "child", uuid)
		_ = m.DeleteConversation(uuid)
		return models.Conversation{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	parentRef := parent.ReferenceNumber
	if parentRef == "" {
		parentRef = parent.UUID
	}
	child, err := m.GetConversation(0, uuid, "")
	if err != nil {
		return models.Conversation{}, err
	}
	childRef := child.ReferenceNumber
	if childRef == "" {
		childRef = child.UUID
	}

	parentActivity := models.ActivityChildCreated
	if origin == models.OriginFollowUp {
		parentActivity = models.ActivityFollowUpCreated
	}
	_ = m.InsertConversationActivity(parentActivity, parent.UUID, childRef, actor)
	_ = m.InsertConversationActivity(models.ActivityOpenedFromParent, child.UUID, parentRef, actor)

	if parent.AssignedTeamID.Valid {
		_ = m.UpdateConversationTeamAssignee(child.UUID, int(parent.AssignedTeamID.Int), actor)
	}
	if parent.AssignedUserID.Valid {
		_ = m.UpdateConversationUserAssignee(child.UUID, int(parent.AssignedUserID.Int), actor)
	}

	return m.GetConversation(0, child.UUID, "")
}

// ListRelatedConversations returns the parent (if any) and direct children / follow-ups.
func (m *Manager) ListRelatedConversations(uuid string) ([]models.RelatedConversation, error) {
	var rows []models.RelatedConversation
	if err := m.q.ListRelatedConversations.Select(&rows, uuid); err != nil {
		m.lo.Error("error listing related conversations", "error", err, "uuid", uuid)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	self, err := m.GetConversation(0, uuid, "")
	if err != nil {
		return nil, err
	}
	for i := range rows {
		if self.ParentUUID.Valid && rows[i].UUID == self.ParentUUID.String {
			rows[i].Relation = "parent"
			continue
		}
		if rows[i].Origin == models.OriginFollowUp {
			rows[i].Relation = models.OriginFollowUp
			continue
		}
		rows[i].Relation = models.OriginChild
	}
	return rows, nil
}

func (m *Manager) ensureNoParentCycle(uuid string) error {
	seen := map[string]struct{}{}
	for i := 0; i < maxParentFollow; i++ {
		if _, ok := seen[uuid]; ok {
			return envelope.NewError(envelope.InputError, m.i18n.T("conversation.related.cycle"), nil)
		}
		seen[uuid] = struct{}{}
		conv, err := m.GetConversation(0, uuid, "")
		if err != nil {
			return err
		}
		if !conv.ParentUUID.Valid || conv.ParentUUID.String == "" {
			return nil
		}
		uuid = conv.ParentUUID.String
	}
	return envelope.NewError(envelope.InputError, m.i18n.T("conversation.related.cycle"), nil)
}
