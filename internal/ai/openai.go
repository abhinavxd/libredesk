package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/ai/models"
	"github.com/zerodha/logf"
)

const (
	defaultOpenAIBaseURL   = "https://api.openai.com/v1"
	defaultCompletionModel = "gpt-4o-mini"
	defaultEmbeddingModel  = "text-embedding-3-small"

	// Transient provider failures (429, 5xx, network errors) are retried with exponential backoff.
	maxRequestRetries = 3
	retryBaseBackoff  = 500 * time.Millisecond
	retryMaxBackoff   = 5 * time.Second

	// Reasoning models reject max_tokens and non-default temperature with structured 400s;
	// those requests are adjusted and retried instead of surfacing the error.
	maxParamAdaptations = 2

	maxProviderResponseBytes = 20 << 20

	// OpenAI embedding models accept up to 8192 tokens; used when a provider doesn't set its own limit.
	defaultEmbeddingMaxTokens = 8192

	// OpenAI's /embeddings caps a request at 2048 array items and ~300k tokens summed across inputs;
	// a batch that breaches either returns 400 and fails wholesale. Stay well under both so a large
	// snippet or URL import (many chunks) is split into several requests instead of silently never
	// indexing. The token budget keeps a margin because the server counts pessimistically.
	maxEmbeddingBatchItems  = 512
	maxEmbeddingBatchTokens = 250000

	// The API rejects an empty/whitespace-only string in the input array with a 400 that fails the
	// whole batch; a lone space is a harmless stand-in that keeps input/output indices aligned.
	emptyEmbeddingPlaceholder = " "
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
func (o *OpenAIClient) SendPrompt(ctx context.Context, payload models.PromptPayload) (string, error) {
	res, err := o.SendChatCompletion(ctx, models.ChatCompletionPayload{
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
func (o *OpenAIClient) SendChatCompletion(ctx context.Context, payload models.ChatCompletionPayload) (models.ChatCompletionResult, error) {
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

	messages := payload.Messages
	sentImages := 0
	for _, msg := range messages {
		sentImages += len(msg.Images)
	}
	if !o.cfg.Vision {
		messages = stripImages(messages)
	}
	o.lo.Debug("chat completion request", "model", model, "messages", len(messages), "images", sentImages, "vision", o.cfg.Vision, "tools", len(payload.Tools))

	body := map[string]any{
		"model":      model,
		"messages":   messages,
		"max_tokens": maxTokens,
	}
	if o.cfg.Temperature != nil {
		body["temperature"] = *o.cfg.Temperature
	}
	if o.cfg.ReasoningEffort != "" {
		body["reasoning_effort"] = o.cfg.ReasoningEffort
	}
	if len(payload.Tools) > 0 {
		body["tools"] = payload.Tools
		body["tool_choice"] = "auto"
	}

	respBytes, err := o.post(ctx, "/chat/completions", body)
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
func (o *OpenAIClient) GetEmbeddings(ctx context.Context, text string) ([]float32, error) {
	vecs, err := o.GetEmbeddingsBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return vecs[0], nil
}

// GetEmbeddingsBatch returns embedding vectors for all texts in a single request.
func (o *OpenAIClient) GetEmbeddingsBatch(ctx context.Context, texts []string) ([][]float32, error) {
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

	maxInputTokens := embeddingTokenLimit(o.cfg)
	inputs := make([]string, len(texts))
	for i, t := range texts {
		if strings.TrimSpace(t) == "" {
			inputs[i] = emptyEmbeddingPlaceholder
			continue
		}
		if capped := capToTokens(t, maxInputTokens); len(capped) < len(t) {
			o.lo.Warn("embedding input exceeded embedding_max_tokens and was truncated; reduce chunk size or use a larger-context embedding model",
				"limit_tokens", maxInputTokens)
			inputs[i] = capped
		} else {
			inputs[i] = t
		}
	}

	out := make([][]float32, len(texts))
	for _, b := range embeddingBatches(inputs) {
		vecs, err := o.embedBatch(ctx, model, inputs[b.start:b.end])
		if err != nil {
			return nil, err
		}
		copy(out[b.start:b.end], vecs)
	}
	return out, nil
}

// embedBatch posts one /embeddings request already known to fit the provider's per-request limits
// and returns the vectors placed back in input order.
func (o *OpenAIClient) embedBatch(ctx context.Context, model string, inputs []string) ([][]float32, error) {
	body := map[string]any{
		"model": model,
		"input": inputs,
	}
	if o.cfg.Dimensions > 0 {
		body["dimensions"] = o.cfg.Dimensions
	}

	respBytes, err := o.post(ctx, "/embeddings", body)
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
	if len(parsed.Data) != len(inputs) {
		return nil, fmt.Errorf("expected %d embeddings, got %d", len(inputs), len(parsed.Data))
	}

	// The API may return embeddings out of order; place each by its index.
	out := make([][]float32, len(inputs))
	seen := make([]bool, len(inputs))
	for _, d := range parsed.Data {
		if d.Index < 0 || d.Index >= len(out) {
			return nil, fmt.Errorf("embedding index %d out of range", d.Index)
		}
		if seen[d.Index] {
			return nil, fmt.Errorf("duplicate embedding index %d", d.Index)
		}
		seen[d.Index] = true
		out[d.Index] = d.Embedding
	}
	return out, nil
}

// stripImages returns messages with image parts removed, so a non-vision model never receives them.
func stripImages(msgs []models.ChatMessage) []models.ChatMessage {
	hasImage := false
	for _, m := range msgs {
		if len(m.Images) > 0 {
			hasImage = true
			break
		}
	}
	if !hasImage {
		return msgs
	}
	out := make([]models.ChatMessage, len(msgs))
	copy(out, msgs)
	for i := range out {
		out[i].Images = nil
	}
	return out
}

func (o *OpenAIClient) post(ctx context.Context, path string, body map[string]any) ([]byte, error) {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshalling request body: %w", err)
	}

	adaptations := 0
	for attempt := 0; ; attempt++ {
		respBytes, retryAfter, retryable, err := o.doRequest(ctx, path, bodyBytes)
		if err == nil {
			return respBytes, nil
		}
		if !retryable {
			if adaptations < maxParamAdaptations {
				if param, ok := adaptUnsupportedParam(body, respBytes); ok {
					adaptations++
					bodyBytes, err = json.Marshal(body)
					if err != nil {
						return nil, fmt.Errorf("marshalling request body: %w", err)
					}
					o.lo.Warn("AI provider rejected a request parameter, retrying without it; clear the field in the provider settings to avoid the extra round trip", "param", param)
					continue
				}
			}
			return nil, err
		}
		if attempt >= maxRequestRetries {
			return nil, err
		}
		delay := retryAfter
		if delay <= 0 {
			delay = backoffDelay(attempt)
		}
		o.lo.Warn("retrying AI provider request after transient error", "attempt", attempt+1, "max_retries", maxRequestRetries, "delay", delay.String(), "error", err)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}
	}
}

// doRequest sends one attempt; retryAfter/retryable tell the caller whether and when to retry.
func (o *OpenAIClient) doRequest(ctx context.Context, path string, bodyBytes []byte) ([]byte, time.Duration, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.cfg.BaseURL+path, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, 0, false, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+o.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		o.lo.Error("error making request to AI provider", "error", err)
		return nil, 0, true, fmt.Errorf("making HTTP request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxProviderResponseBytes))
	if err != nil {
		o.lo.Error("error reading AI provider response", "error", err)
		return nil, 0, true, fmt.Errorf("reading response body: %w", err)
	}
	switch {
	case resp.StatusCode == http.StatusOK:
		return respBytes, 0, false, nil
	case resp.StatusCode == http.StatusUnauthorized:
		o.lo.Error("unauthorized from AI provider (401)", "base_url", o.cfg.BaseURL, "response", string(respBytes))
		return nil, 0, false, ErrInvalidAPIKey
	case resp.StatusCode == http.StatusTooManyRequests:
		o.lo.Error("rate limited by AI provider (429)", "response", string(respBytes))
		return nil, parseRetryAfter(resp.Header.Get("Retry-After")), true, fmt.Errorf("provider API error: status %d: %s", resp.StatusCode, string(respBytes))
	case resp.StatusCode >= 500:
		o.lo.Error("server error from AI provider", "status", resp.StatusCode, "response", string(respBytes))
		return nil, 0, true, fmt.Errorf("provider API error: status %d: %s", resp.StatusCode, string(respBytes))
	default:
		o.lo.Error("non-ok response from AI provider", "status", resp.StatusCode, "response", string(respBytes))
		return respBytes, 0, false, fmt.Errorf("provider API error: status %d: %s", resp.StatusCode, string(respBytes))
	}
}

func backoffDelay(attempt int) time.Duration {
	d := retryBaseBackoff << attempt
	if d > retryMaxBackoff || d <= 0 {
		d = retryMaxBackoff
	}
	return d
}

// adaptUnsupportedParam adjusts body in place when the provider rejected a tuning parameter, reporting which one.
func adaptUnsupportedParam(body map[string]any, resp []byte) (string, bool) {
	var parsed struct {
		Error struct {
			Code  string `json:"code"`
			Param string `json:"param"`
		} `json:"error"`
	}
	if err := json.Unmarshal(resp, &parsed); err != nil {
		return "", false
	}
	if parsed.Error.Code != "unsupported_parameter" && parsed.Error.Code != "unsupported_value" {
		return "", false
	}
	switch parsed.Error.Param {
	case "max_tokens":
		v, ok := body["max_tokens"]
		if !ok {
			return "", false
		}
		delete(body, "max_tokens")
		body["max_completion_tokens"] = v
		return "max_tokens", true
	case "temperature", "reasoning_effort":
		if _, ok := body[parsed.Error.Param]; !ok {
			return "", false
		}
		delete(body, parsed.Error.Param)
		return parsed.Error.Param, true
	}
	return "", false
}

// parseRetryAfter reads the delta-seconds form of Retry-After, capped so a request path never stalls long.
func parseRetryAfter(v string) time.Duration {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	secs, err := strconv.Atoi(v)
	if err != nil || secs < 0 {
		return 0
	}
	d := time.Duration(secs) * time.Second
	if d > retryMaxBackoff {
		d = retryMaxBackoff
	}
	return d
}

// embeddingTokenLimit resolves the per-input token cap, treating an unset (zero) or negative
// config value as the default so existing provider configs keep working.
func embeddingTokenLimit(cfg models.ProviderConfig) int {
	if cfg.EmbeddingMaxTokens > 0 {
		return cfg.EmbeddingMaxTokens
	}
	return defaultEmbeddingMaxTokens
}

type embeddingBatch struct{ start, end int }

// embeddingBatches groups inputs into [start,end) spans each within the provider's per-request item
// and summed-token caps. An input over the token budget on its own still forms a one-item batch (it
// was already capped to the per-input limit upstream, which is below the request budget).
func embeddingBatches(inputs []string) []embeddingBatch {
	var batches []embeddingBatch
	start := 0
	tokens := 0
	for i, in := range inputs {
		t := countTokens(in)
		overItems := i-start >= maxEmbeddingBatchItems
		overTokens := i > start && tokens+t > maxEmbeddingBatchTokens
		if overItems || overTokens {
			batches = append(batches, embeddingBatch{start, i})
			start = i
			tokens = 0
		}
		tokens += t
	}
	if start < len(inputs) {
		batches = append(batches, embeddingBatch{start, len(inputs)})
	}
	return batches
}
