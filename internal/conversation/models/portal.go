package models

import (
	"time"

	"github.com/volatiletech/null/v9"
)

// PortalConversation contains only customer-visible conversation fields.
type PortalConversation struct {
	Total           int         `db:"total" json:"-"`
	UUID            string      `db:"uuid" json:"uuid"`
	ReferenceNumber string      `db:"reference_number" json:"reference_number"`
	Subject         string      `db:"subject" json:"subject"`
	Status          string      `db:"status" json:"status"`
	CreatedAt       time.Time   `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time   `db:"updated_at" json:"updated_at"`
	LastMessage     null.String `db:"last_message" json:"last_message"`
	LastMessageAt   null.Time   `db:"last_message_at" json:"last_message_at"`
}

// PortalMessage contains only customer-visible message fields.
type PortalMessage struct {
	UUID        string      `json:"uuid"`
	CreatedAt   time.Time   `json:"created_at"`
	Type        string      `json:"type"`
	Content     string      `json:"content"`
	TextContent string      `json:"text_content"`
	ContentType string      `json:"content_type"`
	SenderType  string      `json:"sender_type"`
	AuthorName  string      `json:"author_name"`
	AvatarURL   null.String `json:"avatar_url"`
}
