package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/abhinavxd/libredesk/internal/ai/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/logf"
)

type OpenAIClient struct {
	apikey      string
	lo          *logf.Logger
	client      *http.Client
	model       string
	max_tokens  int
	temperature float64
	url         string
	timeout     time.Duration
}

// NewOpenAIClient creates a new OpenAIClient with config values.
func NewOpenAIClient(apiKey, model, url string, temperature float64, maxTokens int, timeout time.Duration, lo *logf.Logger) *OpenAIClient {
	return &OpenAIClient{
		apikey:      apiKey,
		lo:          lo,
		client:      &http.Client{Timeout: timeout},
		model:       model,
		url:         url,
		timeout:     timeout,
		max_tokens:  maxTokens,
		temperature: temperature,
	}
}

// makeRequest creates and executes an HTTP request to the OpenAI API
func (o *OpenAIClient) makeRequest(requestBody any, operation string) (*http.Response, error) {
	if o.apikey == "" {
		return nil, ErrApiKeyNotSet
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		o.lo.Error("error marshalling "+operation+" request body", "error", err)
		return nil, fmt.Errorf("marshalling %s request body: %w", operation, err)
	}

	req, err := http.NewRequest(fasthttp.MethodPost, o.url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		o.lo.Error("error creating "+operation+" request", "error", err)
		return nil, fmt.Errorf("error creating %s request: %w", operation, err)
	}

	req.Header.Set("Authorization", "Bearer "+o.apikey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		o.lo.Error("error making "+operation+" HTTP request", "error", err)
		return nil, fmt.Errorf("making %s HTTP request: %w", operation, err)
	}

	return resp, nil
}

// handleResponse processes the HTTP response and handles common error cases
func (o *OpenAIClient) handleResponse(resp *http.Response, operation string) error {
	if resp.StatusCode == http.StatusUnauthorized {
		return ErrInvalidAPIKey
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		o.lo.Error("non-ok response from openai "+operation+" API", "status", resp.Status, "code", resp.StatusCode, "response_text", body)
		return fmt.Errorf("%s API error: %s, body: %s", operation, resp.Status, body)
	}

	return nil
}

// decodeChatResponse decodes chat completion responses (used by SendPrompt and SendChatCompletion)
func (o *OpenAIClient) decodeChatResponse(resp *http.Response, operation string) (string, error) {
	var responseBody struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&responseBody); err != nil {
		return "", fmt.Errorf("decoding %s response body: %w", operation, err)
	}

	if len(responseBody.Choices) > 0 {
		return responseBody.Choices[0].Message.Content, nil
	}
	return "", fmt.Errorf("no response found")
}

// SendPrompt sends a prompt to the OpenAI API and returns the response text.
func (o *OpenAIClient) SendPrompt(payload models.PromptPayload) (string, error) {
	requestBody := map[string]any{
		"model": o.model,
		"messages": []map[string]string{
			{"role": "system", "content": payload.SystemPrompt},
			{"role": "user", "content": payload.UserPrompt},
		},
		"max_tokens":  o.max_tokens,
		"temperature": o.temperature,
	}

	resp, err := o.makeRequest(requestBody, "prompt")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if err := o.handleResponse(resp, "prompt"); err != nil {
		return "", err
	}

	return o.decodeChatResponse(resp, "prompt")
}

// SendChatCompletion sends a chat completion request with message history to the OpenAI API.
func (o *OpenAIClient) SendChatCompletion(payload models.ChatCompletionPayload) (string, error) {
	// Convert our ChatMessage format to OpenAI's format
	openAIMessages := make([]map[string]string, len(payload.Messages))
	for i, msg := range payload.Messages {
		openAIMessages[i] = map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}

	requestBody := map[string]any{
		"model":       o.model,
		"messages":    openAIMessages,
		"max_tokens":  o.max_tokens,
		"temperature": o.temperature,
	}

	// TODO: Remove after debugging
	for i, msg := range openAIMessages {
		fmt.Println("openai chat message", "index", i)
		fmt.Printf("Role: %s\nContent: %s\n\n\n", msg["role"], msg["content"])
	}

	resp, err := o.makeRequest(requestBody, "chat completion")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if err := o.handleResponse(resp, "chat completion"); err != nil {
		return "", err
	}

	return o.decodeChatResponse(resp, "chat completion")
}

// GetEmbeddings retrieves embeddings for the given text using the OpenAI API.
func (o *OpenAIClient) GetEmbeddings(text string) ([]float32, error) {
	requestBody := map[string]any{
		"model": o.model,
		"input": text,
	}

	resp, err := o.makeRequest(requestBody, "embeddings")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := o.handleResponse(resp, "embeddings"); err != nil {
		return nil, err
	}

	var responseBody struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&responseBody); err != nil {
		return nil, fmt.Errorf("decoding embeddings response body: %w", err)
	}

	if len(responseBody.Data) > 0 {
		return responseBody.Data[0].Embedding, nil
	}
	return nil, fmt.Errorf("no embeddings found in response")
}
