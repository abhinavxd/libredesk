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
