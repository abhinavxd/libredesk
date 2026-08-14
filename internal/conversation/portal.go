package conversation

import (
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
)

// GetPortalConversations returns a page of conversations owned by a contact.
func (c *Manager) GetPortalConversations(contactID, page, pageSize int) ([]models.PortalConversation, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 30
	}
	items := make([]models.PortalConversation, 0)
	err := c.db.Select(&items, `
		SELECT COUNT(*) OVER() AS total, c.uuid, c.reference_number,
		       COALESCE(c.subject, '') AS subject, COALESCE(cs.name, '') AS status,
		       c.created_at, c.updated_at, c.last_message, c.last_message_at
		FROM conversations c
		LEFT JOIN conversation_statuses cs ON cs.id = c.status_id
		WHERE c.contact_id = $1
		ORDER BY COALESCE(c.last_message_at, c.created_at) DESC
		LIMIT $2 OFFSET $3`, contactID, pageSize, (page-1)*pageSize)
	if err != nil {
		c.lo.Error("fetching portal conversations", "contact_id", contactID, "error", err)
		return items, 0, envelope.NewError(envelope.GeneralError, c.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	total := 0
	if len(items) > 0 {
		total = items[0].Total
	}
	return items, total, nil
}

// GetPortalConversation checks ownership before returning customer-visible data.
func (c *Manager) GetPortalConversation(contactID int, uuid string) (models.PortalConversation, error) {
	var item models.PortalConversation
	err := c.db.Get(&item, `
		SELECT c.uuid, c.reference_number, COALESCE(c.subject, '') AS subject,
		       COALESCE(cs.name, '') AS status, c.created_at, c.updated_at,
		       c.last_message, c.last_message_at
		FROM conversations c
		LEFT JOIN conversation_statuses cs ON cs.id = c.status_id
		WHERE c.uuid = $1 AND c.contact_id = $2`, uuid, contactID)
	if err != nil {
		return item, envelope.NewError(envelope.NotFoundError, c.i18n.T("validation.notFoundConversation"), nil)
	}
	return item, nil
}

// GetPortalMessages checks conversation ownership, excludes private content, and sanitizes authors.
func (c *Manager) GetPortalMessages(contactID int, uuid string, page, pageSize int) ([]models.PortalMessage, int, error) {
	if _, err := c.GetPortalConversation(contactID, uuid); err != nil {
		return nil, 0, err
	}
	private := false
	messages, pageSize, err := c.GetConversationMessages(uuid, page, pageSize, &private, []string{models.MessageIncoming, models.MessageOutgoing})
	if err != nil {
		return nil, pageSize, err
	}
	result := make([]models.PortalMessage, 0, len(messages))
	for _, message := range messages {
		result = append(result, models.PortalMessage{
			UUID: message.UUID, CreatedAt: message.CreatedAt, Type: message.Type,
			Content: message.Content, TextContent: message.TextContent, ContentType: message.ContentType,
			SenderType: message.SenderType, AuthorName: message.Author.FullName(), AvatarURL: message.Author.AvatarURL,
		})
	}
	return result, pageSize, nil
}
