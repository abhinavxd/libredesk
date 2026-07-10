package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/ai/models"
	"github.com/zerodha/logf"
)

const (
	defaultOpenAIBaseURL   = "https://api.openai.com/v1"
	defaultCompletionModel = "gpt-4o-mini"
	defaultEmbeddingModel  = "text-embedding-3-small"
)

// OpenAIClient talks to any OpenAI-compatible API (base URL selects the host).
type OpenAIClient struct {
	cfg    models.ProviderConfig
	lo     *logf.Logger
	client *http.Client
}

func NewOpenAIClient(cfg models.ProviderConfig, lo *logf.Logger) *OpenAIClient {
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultOpenAIBaseURL
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	return &OpenAIClient{
		cfg:    cfg,
		lo:     lo,
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

// SendPrompt runs a single system+user prompt and returns the assistant text.
func (o *OpenAIClient) SendPrompt(payload models.PromptPayload) (string, error) {
	res, err := o.SendChatCompletion(models.ChatCompletionPayload{
		Messages: []models.ChatMessage{
			{Role: "system", Content: payload.SystemPrompt},
			{Role: "user", Content: payload.UserPrompt},
		},
	})
	if err != nil {
		return "", err
	}
	return res.Content, nil
}

// SendChatCompletion posts a chat completion, optionally advertising tools.
func (o *OpenAIClient) SendChatCompletion(payload models.ChatCompletionPayload) (models.ChatCompletionResult, error) {
	if o.cfg.APIKey == "" {
		return models.ChatCompletionResult{}, ErrApiKeyNotSet
	}

	model := o.cfg.Model
	if model == "" {
		model = defaultCompletionModel
	}
	maxTokens := o.cfg.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1024
	}

	body := map[string]any{
		"model":      model,
		"messages":   payload.Messages,
		"max_tokens": maxTokens,
	}
	if o.cfg.Temperature != nil {
		body["temperature"] = *o.cfg.Temperature
	}
	if len(payload.Tools) > 0 {
		body["tools"] = payload.Tools
		body["tool_choice"] = "auto"
	}

	respBytes, err := o.post("/chat/completions", body)
	if err != nil {
		return models.ChatCompletionResult{}, err
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content   string            `json:"content"`
				ToolCalls []models.ToolCall `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return models.ChatCompletionResult{}, fmt.Errorf("decoding response body: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return models.ChatCompletionResult{}, fmt.Errorf("no response found")
	}
	return models.ChatCompletionResult{
		Content:   parsed.Choices[0].Message.Content,
		ToolCalls: parsed.Choices[0].Message.ToolCalls,
	}, nil
}

// GetEmbeddings returns the embedding vector for the given text.
func (o *OpenAIClient) GetEmbeddings(text string) ([]float32, error) {
	vecs, err := o.GetEmbeddingsBatch([]string{text})
	if err != nil {
		return nil, err
	}
	return vecs[0], nil
}

// GetEmbeddingsBatch returns embedding vectors for all texts in a single request.
func (o *OpenAIClient) GetEmbeddingsBatch(texts []string) ([][]float32, error) {
	if o.cfg.APIKey == "" {
		return nil, ErrApiKeyNotSet
	}
	if len(texts) == 0 {
		return nil, nil
	}

	model := o.cfg.Model
	if model == "" {
		model = defaultEmbeddingModel
	}
	body := map[string]any{
		"model": model,
		"input": texts,
	}
	if o.cfg.Dimensions > 0 {
		body["dimensions"] = o.cfg.Dimensions
	}

	respBytes, err := o.post("/embeddings", body)
	if err != nil {
		return nil, err
	}

	var parsed struct {
		Data []struct {
			Index     int       `json:"index"`
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return nil, fmt.Errorf("decoding embedding response: %w", err)
	}
	if len(parsed.Data) != len(texts) {
		return nil, fmt.Errorf("expected %d embeddings, got %d", len(texts), len(parsed.Data))
	}

	// The API may return embeddings out of order; place each by its index.
	out := make([][]float32, len(texts))
	for _, d := range parsed.Data {
		if d.Index < 0 || d.Index >= len(out) {
			return nil, fmt.Errorf("embedding index %d out of range", d.Index)
		}
		out[d.Index] = d.Embedding
	}
	return out, nil
}

func (o *OpenAIClient) post(path string, body map[string]any) ([]byte, error) {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshalling request body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, o.cfg.BaseURL+path, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+o.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		o.lo.Error("error making request to AI provider", "error", err)
		return nil, fmt.Errorf("making HTTP request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusUnauthorized {
		o.lo.Error("unauthorized from AI provider (401)", "base_url", o.cfg.BaseURL, "response", string(respBytes))
		return nil, ErrInvalidAPIKey
	}
	if resp.StatusCode != http.StatusOK {
		o.lo.Error("non-ok response from AI provider", "status", resp.StatusCode, "response", string(respBytes))
		return nil, fmt.Errorf("provider API error: status %d: %s", resp.StatusCode, string(respBytes))
	}
	return respBytes, nil
}
