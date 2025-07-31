package ai

import "github.com/abhinavxd/libredesk/internal/ai/models"

// ProviderClient is the interface all providers should implement.
type ProviderClient interface {
	SendPrompt(payload models.PromptPayload) (string, error)
	SendChatCompletion(payload models.ChatCompletionPayload) (string, error)
	GetEmbeddings(text string) ([]float32, error)
}

// ProviderType is an enum-like type for different providers.
type ProviderType string

const (
	ProviderOpenAI ProviderType = "openai"
	ProviderClaude ProviderType = "claude"
)
