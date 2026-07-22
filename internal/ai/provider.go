package ai

import (
	"context"

	"github.com/abhinavxd/libredesk/internal/ai/models"
)

// ProviderClient is implemented by every LLM provider client.
type ProviderClient interface {
	SendPrompt(ctx context.Context, payload models.PromptPayload) (string, error)
	SendChatCompletion(ctx context.Context, payload models.ChatCompletionPayload) (models.ChatCompletionResult, error)
	GetEmbeddings(ctx context.Context, text string) ([]float32, error)
	GetEmbeddingsBatch(ctx context.Context, texts []string) ([][]float32, error)
}
