package conversation

import (
	"fmt"
	"time"

	"github.com/abhinavxd/libredesk/internal/envelope"
)

// PortalConversation contains only fields safe to expose to the owning contact.
type PortalConversation struct {
	Total              int       `db:"total" json:"-"`
	ID                 int       `db:"id" json:"id"`
	UUID               string    `db:"uuid" json:"uuid"`
	Subject            string    `db:"subject" json:"subject"`
	Status             string    `db:"status" json:"status"`
	Priority           string    `db:"priority" json:"priority"`
	CreatedAt          time.Time `db:"created_at" json:"created_at"`
	LastActivityAt     time.Time `db:"last_activity_at" json:"last_activity_at"`
	LastMessagePreview string    `db:"last_message_preview" json:"last_message_preview"`
	UnreadCount        int       `db:"unread_count" json:"unread_count"`
}

// GetPortalConversations scopes the query in SQL to one contact and returns no agent data.
func (c *Manager) GetPortalConversations(contactID, page, pageSize int, status, sort, direction string) ([]PortalConversation, int, error) {
	orderColumn := "c.last_interaction_at"
	if sort == "created_at" {
		orderColumn = "c.created_at"
	}
	orderDirection := "DESC"
	if direction == "asc" {
		orderDirection = "ASC"
	}
	query := fmt.Sprintf(`SELECT COUNT(*) OVER() AS total, c.id, c.uuid,
COALESCE(c.subject, '') AS subject, cs.name AS status,
COALESCE(cp.name, '') AS priority, c.created_at,
COALESCE(c.last_interaction_at, c.created_at) AS last_activity_at,
COALESCE(c.last_interaction, '') AS last_message_preview,
(SELECT COUNT(*) FROM conversation_messages m
 WHERE m.conversation_id = c.id AND m.type = 'outgoing' AND m.private = false
 AND m.created_at > COALESCE(c.contact_last_seen_at, c.created_at)
 AND (m.meta IS NULL OR NOT COALESCE((m.meta->>'continuity_email')::boolean, false))) AS unread_count
FROM conversations c
JOIN conversation_statuses cs ON cs.id = c.status_id
LEFT JOIN conversation_priorities cp ON cp.id = c.priority_id
JOIN inboxes i ON i.id = c.inbox_id AND i.deleted_at IS NULL
WHERE c.contact_id = $1 AND ($2 = '' OR cs.name = $2)
ORDER BY %s %s, c.id DESC LIMIT $3 OFFSET $4`, orderColumn, orderDirection)
	items := make([]PortalConversation, 0)
	var total int
	countQuery := `SELECT COUNT(*) FROM conversations c JOIN conversation_statuses cs ON cs.id=c.status_id JOIN inboxes i ON i.id=c.inbox_id AND i.deleted_at IS NULL WHERE c.contact_id=$1 AND ($2='' OR cs.name=$2)`
	if err := c.db.Get(&total, countQuery, contactID, status); err != nil {
		c.lo.Error("error counting portal conversations", "contact_id", contactID, "error", err)
		return nil, 0, envelope.NewError(envelope.GeneralError, c.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if err := c.db.Select(&items, query, contactID, status, pageSize, (page-1)*pageSize); err != nil {
		c.lo.Error("error listing portal conversations", "contact_id", contactID, "error", err)
		return nil, 0, envelope.NewError(envelope.GeneralError, c.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return items, total, nil
}
