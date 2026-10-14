package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/abhinavxd/libredesk/internal/ai/models"
	"github.com/zerodha/logf"
)

func TestLearnedParamAdaptation(t *testing.T) {
	var bodies []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding request body: %v", err)
		}
		bodies = append(bodies, body)
		if _, ok := body["max_tokens"]; ok {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":{"code":"unsupported_parameter","param":"max_tokens"}}`))
			return
		}
		w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer srv.Close()

	lo := logf.New(logf.Opts{})
	cfg := models.ProviderConfig{BaseURL: srv.URL, APIKey: "test", Model: "learned-param-test-model"}
	payload := models.ChatCompletionPayload{Messages: []models.ChatMessage{{Role: "user", Content: "hi"}}}

	if _, err := NewOpenAIClient(cfg, &lo, srv.Client()).SendChatCompletion(context.Background(), payload); err != nil {
		t.Fatalf("first request should succeed after adaptation, got %v", err)
	}
	if len(bodies) != 2 {
		t.Fatalf("first request should take one 400 round trip then retry, got %d requests", len(bodies))
	}

	// A fresh client must remember the swap and send max_completion_tokens up front.
	if _, err := NewOpenAIClient(cfg, &lo, srv.Client()).SendChatCompletion(context.Background(), payload); err != nil {
		t.Fatalf("second request should succeed, got %v", err)
	}
	if len(bodies) != 3 {
		t.Fatalf("second request should take a single round trip, got %d total requests", len(bodies))
	}
	if _, ok := bodies[2]["max_tokens"]; ok {
		t.Fatal("learned request must not send max_tokens")
	}
	if _, ok := bodies[2]["max_completion_tokens"]; !ok {
		t.Fatal("learned request must send max_completion_tokens")
	}
}

func TestEmbeddingBatches(t *testing.T) {
	// A handful of small inputs stays a single request (the common per-snippet reindex path).
	if got := embeddingBatches([]string{"a", "b", "c"}); len(got) != 1 || got[0] != (embeddingBatch{0, 3}) {
		t.Fatalf("small input should be one batch, got %v", got)
	}

	// More items than the per-request item cap split into multiple batches covering every index once.
	n := maxEmbeddingBatchItems*2 + 5
	inputs := make([]string, n)
	for i := range inputs {
		inputs[i] = "x"
	}
	batches := embeddingBatches(inputs)
	if len(batches) != 3 {
		t.Fatalf("expected 3 item-capped batches for %d inputs, got %d", n, len(batches))
	}
	prevEnd := 0
	for _, b := range batches {
		if b.start != prevEnd {
			t.Fatalf("batches must be contiguous: gap/overlap at %d (batch %v)", prevEnd, b)
		}
		if b.end-b.start > maxEmbeddingBatchItems {
			t.Fatalf("batch %v exceeds item cap %d", b, maxEmbeddingBatchItems)
		}
		prevEnd = b.end
	}
	if prevEnd != n {
		t.Fatalf("batches must cover all %d inputs, covered %d", n, prevEnd)
	}

	if got := embeddingBatches(nil); len(got) != 0 {
		t.Fatalf("no inputs should yield no batches, got %v", got)
	}
}

func TestEmbeddingTokenLimit(t *testing.T) {
	if got := embeddingTokenLimit(models.ProviderConfig{}); got != defaultEmbeddingMaxTokens {
		t.Fatalf("unset (0) should default to %d, got %d", defaultEmbeddingMaxTokens, got)
	}
	if got := embeddingTokenLimit(models.ProviderConfig{EmbeddingMaxTokens: -5}); got != defaultEmbeddingMaxTokens {
		t.Fatalf("negative should default to %d, got %d", defaultEmbeddingMaxTokens, got)
	}
	if got := embeddingTokenLimit(models.ProviderConfig{EmbeddingMaxTokens: 512}); got != 512 {
		t.Fatalf("explicit value should be honored, got %d", got)
	}
}

// Gemini 3 returns a thought signature in extra_content on each tool call and rejects the
// next turn unless the assistant message replays it unchanged.
func TestToolCallExtraContentRoundTrip(t *testing.T) {
	const extra = `{"google":{"thought_signature":"c2ln"}}`
	var bodies []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding request body: %v", err)
		}
		bodies = append(bodies, body)
		w.Write([]byte(`{"choices":[{"message":{"content":"","tool_calls":[{"id":"call_1","type":"function","function":{"name":"search_articles","arguments":"{}"},"extra_content":` + extra + `}]}}]}`))
	}))
	defer srv.Close()

	lo := logf.New(logf.Opts{})
	client := NewOpenAIClient(models.ProviderConfig{BaseURL: srv.URL, APIKey: "test", Model: "extra-content-test-model"}, &lo, srv.Client())
	payload := models.ChatCompletionPayload{Messages: []models.ChatMessage{{Role: "user", Content: "hi"}}}

	res, err := client.SendChatCompletion(context.Background(), payload)
	if err != nil {
		t.Fatalf("first request: %v", err)
	}
	payload.Messages = append(payload.Messages,
		models.ChatMessage{Role: models.RoleAssistant, ToolCalls: res.ToolCalls},
		models.ChatMessage{Role: models.RoleTool, ToolCallID: "call_1", Content: "[]"},
	)
	if _, err := client.SendChatCompletion(context.Background(), payload); err != nil {
		t.Fatalf("second request: %v", err)
	}

	toolCall := bodies[1]["messages"].([]any)[1].(map[string]any)["tool_calls"].([]any)[0].(map[string]any)
	got, err := json.Marshal(toolCall["extra_content"])
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != extra {
		t.Fatalf("replayed tool call extra_content = %s, want %s", got, extra)
	}
}

// GPT reasoning models reject function tools with reasoning_effort on /chat/completions using a
// null error code and ask for reasoning_effort "none"; the request is retried that way and remembered.
func TestReasoningEffortWithToolsAdaptation(t *testing.T) {
	const rejection = `{"error":{"message":"Function tools with reasoning_effort are not supported for test-reasoning-model in /v1/chat/completions. To use function tools, use /v1/responses or set reasoning_effort to 'none'.","type":"invalid_request_error","param":"reasoning_effort","code":null}}`
	var bodies []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding request body: %v", err)
		}
		bodies = append(bodies, body)
		if _, hasTools := body["tools"]; hasTools && body["reasoning_effort"] != "none" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(rejection))
			return
		}
		w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer srv.Close()

	lo := logf.New(logf.Opts{})
	cfg := models.ProviderConfig{BaseURL: srv.URL, APIKey: "test", Model: "test-reasoning-model", ReasoningEffort: "medium"}
	withTools := models.ChatCompletionPayload{
		Messages: []models.ChatMessage{{Role: "user", Content: "hi"}},
		Tools:    []models.ToolDef{{Type: "function", Function: models.ToolFunction{Name: "search_articles", Parameters: []byte(`{}`)}}},
	}

	if _, err := NewOpenAIClient(cfg, &lo, srv.Client()).SendChatCompletion(context.Background(), withTools); err != nil {
		t.Fatalf("request with tools should succeed after adaptation, got %v", err)
	}
	if len(bodies) != 2 {
		t.Fatalf("expected one rejected request then a retry, got %d requests", len(bodies))
	}
	if bodies[1]["reasoning_effort"] != "none" {
		t.Fatalf("retry must send reasoning_effort none, got %v", bodies[1]["reasoning_effort"])
	}

	// A fresh client must remember it and send none up front for requests with tools.
	if _, err := NewOpenAIClient(cfg, &lo, srv.Client()).SendChatCompletion(context.Background(), withTools); err != nil {
		t.Fatalf("second request with tools should succeed, got %v", err)
	}
	if len(bodies) != 3 || bodies[2]["reasoning_effort"] != "none" {
		t.Fatalf("learned request should send reasoning_effort none in a single round trip, got %d requests, last %v", len(bodies), bodies[len(bodies)-1]["reasoning_effort"])
	}

	// Requests without tools keep the configured effort.
	noTools := models.ChatCompletionPayload{Messages: withTools.Messages}
	if _, err := NewOpenAIClient(cfg, &lo, srv.Client()).SendChatCompletion(context.Background(), noTools); err != nil {
		t.Fatalf("request without tools should succeed, got %v", err)
	}
	if bodies[3]["reasoning_effort"] != "medium" {
		t.Fatalf("request without tools must keep the configured reasoning_effort, got %v", bodies[3]["reasoning_effort"])
	}
}
