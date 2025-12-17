package ai

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// ProviderClient is the interface all providers should implement.
type ProviderClient interface {
	SendPrompt(payload PromptPayload) (string, error)
}

// ProviderType is an enum-like type for different providers.
type ProviderType string

const (
	ProviderOpenAI     ProviderType = "openai"
	ProviderClaude     ProviderType = "claude"
	ProviderOpenRouter ProviderType = "openrouter"
)

// PromptPayload represents the structured input for an LLM provider.
type PromptPayload struct {
	SystemPrompt string `json:"system_prompt"`
	UserPrompt   string `json:"user_prompt"`
}

// ProviderInfo contains information about an AI provider for the frontend.
type ProviderInfo struct {
	Provider  string `json:"provider"`
	Name      string `json:"name"`
	Model     string `json:"model,omitempty"`
	HasAPIKey bool   `json:"has_api_key"`
	IsDefault bool   `json:"is_default"`
}

// SupportedProviders returns a list of supported provider types.
var SupportedProviders = []ProviderInfo{
	{Provider: string(ProviderOpenAI), Name: "OpenAI", Model: "gpt-4o-mini"},
	{Provider: string(ProviderOpenRouter), Name: "OpenRouter", Model: "anthropic/claude-sonnet-4.5"},
}

// Model cache for OpenRouter models
var (
	modelCache      []string
	modelCacheMutex sync.RWMutex
	modelCacheTime  time.Time
	modelCacheTTL   = 24 * time.Hour
)

// OpenRouterModel represents a model from OpenRouter API
type OpenRouterModel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// OpenRouterModelsResponse represents the API response
type OpenRouterModelsResponse struct {
	Data []OpenRouterModel `json:"data"`
}

// providerConfig defines how many models to take from each provider
type providerConfig struct {
	prefix   string
	maxCount int
}

// preferredProviders defines providers and how many models to show from each
var preferredProviders = []providerConfig{
	{"anthropic/", 8},
	{"openai/", 10},
	{"google/", 6},
	{"x-ai/", 4},
	{"moonshotai/", 3},
	{"z-ai/", 4},
	{"deepseek/", 5},
	{"qwen/", 5},
	{"meta-llama/", 5},
	{"mistralai/", 5},
}

// FetchOpenRouterModels fetches models from OpenRouter API with caching
func FetchOpenRouterModels() ([]string, error) {
	modelCacheMutex.RLock()
	if len(modelCache) > 0 && time.Since(modelCacheTime) < modelCacheTTL {
		defer modelCacheMutex.RUnlock()
		return modelCache, nil
	}
	modelCacheMutex.RUnlock()

	// Fetch fresh models
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://openrouter.ai/api/v1/models")
	if err != nil {
		return getFallbackModels(), nil
	}
	defer resp.Body.Close()

	var result OpenRouterModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return getFallbackModels(), nil
	}

	// Filter and balance models across providers
	models := balanceModelsAcrossProviders(result.Data)

	// Update cache
	modelCacheMutex.Lock()
	modelCache = models
	modelCacheTime = time.Now()
	modelCacheMutex.Unlock()

	return models, nil
}

// balanceModelsAcrossProviders takes a balanced selection from each provider
func balanceModelsAcrossProviders(data []OpenRouterModel) []string {
	// Group models by provider
	providerModels := make(map[string][]string)

	for _, model := range data {
		// Skip free tier and special variants
		if strings.HasSuffix(model.ID, ":free") || strings.HasSuffix(model.ID, ":exacto") {
			continue
		}

		// Find which provider this belongs to
		for _, pc := range preferredProviders {
			if strings.HasPrefix(model.ID, pc.prefix) {
				providerModels[pc.prefix] = append(providerModels[pc.prefix], model.ID)
				break
			}
		}
	}

	// Sort each providers models (newer/better models tend to have higher version numbers)
	for prefix := range providerModels {
		models := providerModels[prefix]
		sort.Slice(models, func(i, j int) bool {
			// Sort descending so newer versions come first
			return models[i] > models[j]
		})
	}

	// Take configured number from each provider
	var result []string
	for _, pc := range preferredProviders {
		models := providerModels[pc.prefix]
		count := pc.maxCount
		if len(models) < count {
			count = len(models)
		}
		result = append(result, models[:count]...)
	}

	return result
}

// getFallbackModels returns a static list if API is unavailable
func getFallbackModels() []string {
	return []string{
		// Anthropic
		"anthropic/claude-opus-4.5",
		"anthropic/claude-sonnet-4.5",
		"anthropic/claude-haiku-4.5",
		"anthropic/claude-opus-4",
		"anthropic/claude-sonnet-4",
		// OpenAI
		"openai/gpt-5.2-pro",
		"openai/gpt-5.1",
		"openai/o3-pro",
		"openai/o3",
		"openai/gpt-4o",
		// Google
		"google/gemini-3-pro-preview",
		"google/gemini-2.5-pro",
		"google/gemini-2.5-flash",
		// xAI
		"x-ai/grok-4",
		"x-ai/grok-3",
		// Moonshot
		"moonshotai/kimi-k2",
		// Zhipu
		"z-ai/glm-4.6",
		"z-ai/glm-4.5",
		// DeepSeek
		"deepseek/deepseek-v3.2",
		"deepseek/deepseek-r1",
		// Qwen
		"qwen/qwen3-max",
		"qwen/qwen3-235b-a22b",
		// Meta
		"meta-llama/llama-4-maverick",
		"meta-llama/llama-3.3-70b-instruct",
		// Mistral
		"mistralai/mistral-large-2512",
	}
}

// PopularOpenRouterModels kept for backwards compatibility
var PopularOpenRouterModels = getFallbackModels()
