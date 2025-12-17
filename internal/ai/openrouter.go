package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/valyala/fasthttp"
	"github.com/zerodha/logf"
)

// OpenRouterClient implements ProviderClient for OpenRouter API.
type OpenRouterClient struct {
	apiKey string
	model  string
	lo     *logf.Logger
	client *http.Client
}

// NewOpenRouterClient creates a new OpenRouter client.
func NewOpenRouterClient(apiKey, model string, lo *logf.Logger) *OpenRouterClient {
	if model == "" {
		model = "anthropic/claude-3-haiku" // Default model
	}
	return &OpenRouterClient{
		apiKey: apiKey,
		model:  model,
		lo:     lo,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// SendPrompt sends a prompt to the OpenRouter API and returns the response text.
func (o *OpenRouterClient) SendPrompt(payload PromptPayload) (string, error) {
	if o.apiKey == "" {
		return "", ErrApiKeyNotSet
	}

	apiURL := "https://openrouter.ai/api/v1/chat/completions"
	requestBody := map[string]interface{}{
		"model": o.model,
		"messages": []map[string]string{
			{"role": "system", "content": payload.SystemPrompt},
			{"role": "user", "content": payload.UserPrompt},
		},
		"max_tokens":  1024,
		"temperature": 0.7,
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		o.lo.Error("error marshalling request body", "error", err)
		return "", fmt.Errorf("marshalling request body: %w", err)
	}

	req, err := http.NewRequest(fasthttp.MethodPost, apiURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		o.lo.Error("error creating request", "error", err)
		return "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+o.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://libredesk.io")
	req.Header.Set("X-Title", "LibreDesk Helpdesk")

	resp, err := o.client.Do(req)
	if err != nil {
		o.lo.Error("error making HTTP request", "error", err)
		return "", fmt.Errorf("making HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return "", ErrInvalidAPIKey
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		o.lo.Error("non-ok response received from OpenRouter API", "status", resp.Status, "code", resp.StatusCode, "response_text", string(body))
		return "", fmt.Errorf("API error: %s, body: %s", resp.Status, body)
	}

	var responseBody struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&responseBody); err != nil {
		return "", fmt.Errorf("decoding response body: %w", err)
	}

	if len(responseBody.Choices) > 0 {
		return responseBody.Choices[0].Message.Content, nil
	}
	return "", fmt.Errorf("no response found")
}

// TestConnection tests the OpenRouter API connection.
func (o *OpenRouterClient) TestConnection() error {
	if o.apiKey == "" {
		return ErrApiKeyNotSet
	}

	// Simple test with a minimal prompt
	_, err := o.SendPrompt(PromptPayload{
		SystemPrompt: "You are a helpful assistant.",
		UserPrompt:   "Say 'OK' to confirm connection.",
	})
	return err
}
