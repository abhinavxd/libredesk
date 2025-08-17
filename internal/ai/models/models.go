package models

import (
	"time"

	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/volatiletech/null/v9"
)

type Provider struct {
	ID        string    `db:"id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
	Name      string    `db:"name"`
	Provider  string    `db:"provider"`
	Config    string    `db:"config"`
	IsDefault bool      `db:"is_default"`
}

type Prompt struct {
	ID        int       `db:"id" json:"id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
	Title     string    `db:"title" json:"title"`
	Key       string    `db:"key" json:"key"`
	Content   string    `db:"content" json:"content,omitempty"`
}

// ConversationCompletionRequest represents a request for AI conversation completion
type ConversationCompletionRequest struct {
	Messages         []cmodels.Message
	InboxID          int
	ContactID        int
	ConversationUUID string
	AIAssistant      umodels.User
	HelpCenterID     null.Int
}

// ChatMessage represents a single message in a chat
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionPayload represents the structured input for chat completion
type ChatCompletionPayload struct {
	Messages []ChatMessage `json:"messages"`
}

// PromptPayload represents the structured input for an LLM provider.
type PromptPayload struct {
	SystemPrompt string `json:"system_prompt"`
	UserPrompt   string `json:"user_prompt"`
}

// KnowledgeBase represents an AI knowledge base record
type KnowledgeBase struct {
	ID        int       `db:"id" json:"id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
	Type      string    `db:"type" json:"type"`
	Content   string    `db:"content" json:"content"`
	Enabled   bool      `db:"enabled" json:"enabled"`
}

// KnowledgeBaseResult represents a knowledge base entry with similarity score
type KnowledgeBaseResult struct {
	ID         int       `db:"id" json:"id"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at" json:"updated_at"`
	Type       string    `db:"type" json:"type"`
	Content    string    `db:"content" json:"content"`
	Similarity float64   `db:"similarity" json:"similarity"`
}

// UnifiedKnowledgeResult represents a unified search result from knowledge base
type UnifiedKnowledgeResult struct {
	SourceType   string  `db:"source_type" json:"source_type"`
	SourceID     int     `db:"source_id" json:"source_id"`
	Title        string  `db:"title" json:"title"`
	Content      string  `db:"content" json:"content"`
	HelpCenterID *int    `db:"help_center_id" json:"help_center_id"`
	Similarity   float64 `db:"similarity" json:"similarity"`
}
