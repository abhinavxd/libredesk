package models

import (
	"encoding/json"
	"time"

	"github.com/volatiletech/null/v9"
)

// NotificationType represents the type of user notification.
type NotificationType string

const (
	NotificationTypeMention    NotificationType = "mention"
	NotificationTypeAssignment NotificationType = "assignment"
	NotificationTypeSLAWarning NotificationType = "sla_warning"
	NotificationTypeSLABreach  NotificationType = "sla_breach"
)

// UserNotification represents an in-app notification for a user.
type UserNotification struct {
	ID               int              `db:"id" json:"id"`
	CreatedAt        time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time        `db:"updated_at" json:"updated_at"`
	UserID           int              `db:"user_id" json:"user_id"`
	NotificationType NotificationType `db:"notification_type" json:"notification_type"`
	Title            string           `db:"title" json:"title"`
	Body             null.String      `db:"body" json:"body"`
	IsRead           bool             `db:"is_read" json:"is_read"`
	ConversationID   null.Int         `db:"conversation_id" json:"conversation_id"`
	MessageID        null.Int         `db:"message_id" json:"message_id"`
	ActorID          null.Int         `db:"actor_id" json:"actor_id"`
	Meta             json.RawMessage  `db:"meta" json:"meta"`

	// Joined fields from related tables
	ActorFirstName   null.String `db:"actor_first_name" json:"actor_first_name"`
	ActorLastName    null.String `db:"actor_last_name" json:"actor_last_name"`
	ActorAvatarURL   null.String `db:"actor_avatar_url" json:"actor_avatar_url"`
	ConversationUUID null.String `db:"conversation_uuid" json:"conversation_uuid"`
	MessageUUID      null.String `db:"message_uuid" json:"message_uuid"`
}

// NotificationStats holds notification statistics for a user.
type NotificationStats struct {
	UnreadCount int `db:"unread_count" json:"unread_count"`
	TotalCount  int `db:"total_count" json:"total_count"`
}
