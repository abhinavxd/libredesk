package models

import (
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/lib/pq"
)

var discordWebhookPath = regexp.MustCompile(`(?i)^/api(?:/v\d+)?/webhooks/\d+/[\w-]+$`)

const (
	DeliveryHTTP    = "http"
	DeliveryDiscord = "discord"
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
	Delivery  string         `db:"delivery" json:"delivery"`
}

func IsDiscordWebhookURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	switch host {
	case "discord.com", "discordapp.com", "ptb.discord.com", "canary.discord.com":
	default:
		return false
	}
	return discordWebhookPath.MatchString(u.Path)
}

func (w Webhook) IsDiscordURL() bool {
	return IsDiscordWebhookURL(w.URL)
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
