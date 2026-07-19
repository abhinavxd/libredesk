package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx/types"
)

const (
	// OpenAI-compatible chat message roles.
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"

	ProviderTypeCompletion = "completion"
	ProviderTypeEmbedding  = "embedding"

	KnowledgeTypeSnippet = "snippet"

	// Source types for embeddings.
	SourceSnippet = "snippet"

	// ai_knowledge_base.source values.
	KnowledgeSourceManual       = "manual"
	KnowledgeSourceConversation = "conversation"
	KnowledgeSourceURL          = "url"
)

// Provider is a row in ai_providers (one per type: completion / embedding).
type Provider struct {
	ID        int            `db:"id" json:"id"`
	CreatedAt time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt time.Time      `db:"updated_at" json:"updated_at"`
	Name      string         `db:"name" json:"name"`
	Provider  string         `db:"provider" json:"provider"`
	Type      string         `db:"type" json:"type"`
	Config    types.JSONText `db:"config" json:"config"`
	IsDefault bool           `db:"is_default" json:"is_default"`
}

// ProviderConfig is the decoded ai_providers.config JSONB. OpenAI-compatible.
type ProviderConfig struct {
	Provider    string   `json:"provider"`
	BaseURL     string   `json:"base_url"`
	APIKey      string   `json:"api_key"`
	Model       string   `json:"model"`
	Temperature *float64 `json:"temperature,omitempty"`
	MaxTokens   int      `json:"max_tokens"`
	Dimensions  int      `json:"dimensions"`
	// EmbeddingMaxTokens caps each embedding input to the model's context limit; defaults to
	// OpenAI's 8192. Lower it (e.g. 512) for local models like bge/nomic-embed served over an
	// OpenAI-compatible endpoint so their requests don't get rejected.
	EmbeddingMaxTokens int    `json:"embedding_max_tokens"`
	Instructions       string `json:"instructions"`
	// ReasoningEffort, when set, is sent as-is. Reasoning models (e.g. GPT-5.x) need "none" to use
	// tools on /chat/completions; leave blank for models that reject the param.
	ReasoningEffort string `json:"reasoning_effort,omitempty"`
	// Vision marks the completion model as accepting image input; when false the harness drops image parts.
	Vision bool `json:"vision"`
}

// Prompt backs the agent-facing rephrase/summarize actions in the reply box.
type Prompt struct {
	ID        int       `db:"id" json:"id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
	Title     string    `db:"title" json:"title"`
	Key       string    `db:"key" json:"key"`
	Content   string    `db:"content" json:"content,omitempty"`
}

type KnowledgeBaseItem struct {
	ID        int       `db:"id" json:"id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
	Type      string    `db:"type" json:"type"`
	Title     string    `db:"title" json:"title"`
	Content   string    `db:"content" json:"content"`
	Enabled   bool      `db:"enabled" json:"enabled"`
	Source    string    `db:"source" json:"source"`
	SourceURL string    `db:"source_url" json:"source_url"`
	// EmbeddedFingerprint is the content+model+dimensions signature last successfully embedded; empty means it needs (re)embedding.
	EmbeddedFingerprint string `db:"embedded_fingerprint" json:"-"`
}

// Tool is a custom, admin-defined HTTP tool the assistant can call.
type Tool struct {
	ID          int            `db:"id" json:"id"`
	CreatedAt   time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time      `db:"updated_at" json:"updated_at"`
	Name        string         `db:"name" json:"name"`
	Description string         `db:"description" json:"description"`
	URL         string         `db:"url" json:"url"`
	Method      string         `db:"method" json:"method"`
	Auth        types.JSONText `db:"auth" json:"auth"`
	Parameters  types.JSONText `db:"parameters" json:"parameters"`
	Enabled     bool           `db:"enabled" json:"enabled"`
}

// ToolAuth is the decoded ai_tools.auth JSONB: headers injected on every request.
type ToolAuth struct {
	Headers []ToolAuthHeader `json:"headers"`
}

// ToolAuthHeader is a single HTTP header name/value pair sent with a tool request.
type ToolAuthHeader struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Embedding is a stored chunk vector row.
type Embedding struct {
	ID         int64  `db:"id"`
	SourceType string `db:"source_type"`
	SourceID   int64  `db:"source_id"`
	ChunkText  string `db:"chunk_text"`
	Embedding  []byte `db:"embedding"`
	Dimensions int    `db:"dimensions"`
}

// SearchResult is one hit from the in-memory embedding search.
type SearchResult struct {
	SourceType string  `json:"source_type"`
	SourceID   int     `json:"source_id"`
	ChunkText  string  `json:"chunk_text"`
	Score      float64 `json:"score"`
}

// ChatMessage is one OpenAI-compatible chat message.
type ChatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	Name       string     `json:"name,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`

	// Images are folded into the OpenAI multimodal content array by MarshalJSON; empty keeps the plain string content.
	Images []ChatImage `json:"-"`
}

// ChatImage is one image part sent to vision-capable models.
type ChatImage struct {
	MediaType string // e.g. image/png, image/jpeg
	Data      string // base64-encoded image bytes
}

// CopilotMessage is one persisted turn of an agent's copilot chat on a conversation.
type CopilotMessage struct {
	Role    string `db:"role" json:"role"`
	Content string `db:"content" json:"content"`
}

// MarshalJSON emits plain string content when there are no images, or the OpenAI multimodal
// content array (text part followed by image_url parts) when images are attached.
func (m ChatMessage) MarshalJSON() ([]byte, error) {
	type wire struct {
		Role       string     `json:"role"`
		Content    any        `json:"content"`
		Name       string     `json:"name,omitempty"`
		ToolCallID string     `json:"tool_call_id,omitempty"`
		ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	}
	w := wire{Role: m.Role, Name: m.Name, ToolCallID: m.ToolCallID, ToolCalls: m.ToolCalls}
	if len(m.Images) == 0 {
		w.Content = m.Content
		return json.Marshal(w)
	}
	parts := make([]map[string]any, 0, len(m.Images)+1)
	if strings.TrimSpace(m.Content) != "" {
		parts = append(parts, map[string]any{"type": "text", "text": m.Content})
	}
	for _, img := range m.Images {
		parts = append(parts, map[string]any{
			"type":      "image_url",
			"image_url": map[string]any{"url": fmt.Sprintf("data:%s;base64,%s", img.MediaType, img.Data)},
		})
	}
	w.Content = parts
	return json.Marshal(w)
}

type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function ToolCallFunction `json:"function"`
}

type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolDef is a tool advertised to the model in a chat completion request.
type ToolDef struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  types.JSONText `json:"parameters"`
}

type PromptPayload struct {
	SystemPrompt string `json:"system_prompt"`
	UserPrompt   string `json:"user_prompt"`
}

type ChatCompletionPayload struct {
	Messages []ChatMessage `json:"messages"`
	Tools    []ToolDef     `json:"tools,omitempty"`
}

// ChatCompletionResult is the parsed assistant turn: either text or tool calls.
type ChatCompletionResult struct {
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	Usage     TokenUsage `json:"usage"`
}

// TokenUsage carries the provider's token accounting for one completion.
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
