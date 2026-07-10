package ai

import "github.com/abhinavxd/libredesk/internal/ai/models"

// ProviderClient is implemented by every LLM provider client.
type ProviderClient interface {
	SendPrompt(payload models.PromptPayload) (string, error)
	SendChatCompletion(payload models.ChatCompletionPayload) (models.ChatCompletionResult, error)
	GetEmbeddings(text string) ([]float32, error)
	GetEmbeddingsBatch(texts []string) ([][]float32, error)
}
