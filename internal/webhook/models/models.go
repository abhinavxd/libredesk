package models

import (
	"time"

	"github.com/lib/pq"
)

// Webhook represents a webhook configuration
type Webhook struct {
	ID        int            `db:"id" json:"id"`
	CreatedAt time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt time.Time      `db:"updated_at" json:"updated_at"`
	Name      string         `db:"name" json:"name"`
	URL       string         `db:"url" json:"url"`
	Events    pq.StringArray `db:"events" json:"events"`
	Secret    string         `db:"secret" json:"secret"`
	IsActive  bool           `db:"is_active" json:"is_active"`
}

// WebhookEvent represents an event that can trigger a webhook
type WebhookEvent string

const (
	// Conversation events
	EventConversationCreated       WebhookEvent = "conversation.created"
	EventConversationStatusChanged WebhookEvent = "conversation.status_changed"
	EventConversationTagsChanged   WebhookEvent = "conversation.tags_changed"
	EventConversationAssigned      WebhookEvent = "conversation.assigned"
	EventConversationUnassigned    WebhookEvent = "conversation.unassigned"

	// Message events
	EventMessageCreated WebhookEvent = "message.created"
	EventMessageUpdated WebhookEvent = "message.updated"

	// Test event
	EventWebhookTest WebhookEvent = "webhook.test"
)
