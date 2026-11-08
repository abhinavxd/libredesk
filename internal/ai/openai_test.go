package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

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

	var logs bytes.Buffer
	lo := logf.New(logf.Opts{Writer: &logs})
	cfg := models.ProviderConfig{BaseURL: srv.URL, APIKey: "test", Model: "learned-param-test-model"}
	payload := models.ChatCompletionPayload{Messages: []models.ChatMessage{{Role: "user", Content: "hi"}}}

	if _, err := NewOpenAIClient(cfg, &lo, srv.Client(), time.UTC).SendChatCompletion(context.Background(), payload); err != nil {
		t.Fatalf("first request should succeed after adaptation, got %v", err)
	}
	if len(bodies) != 2 {
		t.Fatalf("first request should take one 400 round trip then retry, got %d requests", len(bodies))
	}

	if strings.Contains(logs.String(), "level=error") || !strings.Contains(logs.String(), "level=warn") {
		t.Fatalf("recovered adaptation has incorrect severity: %s", logs.String())
	}

	// A fresh client must remember the swap and send max_completion_tokens up front.
	if _, err := NewOpenAIClient(cfg, &lo, srv.Client(), time.UTC).SendChatCompletion(context.Background(), payload); err != nil {
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
	client := NewOpenAIClient(models.ProviderConfig{BaseURL: srv.URL, APIKey: "test", Model: "extra-content-test-model"}, &lo, srv.Client(), time.UTC)
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

	toolCall := bodies[1]["messages"].([]any)[2].(map[string]any)["tool_calls"].([]any)[0].(map[string]any)
	got, err := json.Marshal(toolCall["extra_content"])
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != extra {
		t.Fatalf("replayed tool call extra_content = %s, want %s", got, extra)
	}
}

func TestTimeContextPreservesHistory(t *testing.T) {
	now := time.Date(2026, time.November, 2, 0, 15, 0, 0, time.FixedZone("NPT", 5*3600+45*60))
	for _, msgs := range [][]models.ChatMessage{
		nil,
		{{Role: models.RoleUser, Content: "What day is it?"}},
		{{Role: models.RoleSystem, Content: "Help the customer."}, {Role: models.RoleUser, Content: "Hi"}},
		{{Role: models.RoleUser, Images: []models.ChatImage{{MediaType: "image/png", Data: "image"}}}},
	} {
		before := slices.Clone(msgs)
		got := withTimeContext(msgs, now, time.UTC)
		if got[0].Role != models.RoleSystem || !strings.Contains(got[0].Content, "Current date and time: 2026-11-01T18:30:00Z (Sunday, UTC).") {
			t.Fatalf("missing UTC clock: %#v", got)
		}
		if !reflect.DeepEqual(msgs, before) {
			t.Fatalf("input history mutated: %#v", msgs)
		}
		if len(msgs) > 0 && msgs[0].Role == models.RoleSystem {
			if len(got) != len(msgs) || !strings.HasPrefix(got[0].Content, msgs[0].Content+"\n\n") || !reflect.DeepEqual(got[1:], msgs[1:]) {
				t.Fatalf("existing instructions or history changed: %#v", got)
			}
		} else if len(got) != len(msgs)+1 || !reflect.DeepEqual(got[1:], msgs) && len(msgs) > 0 {
			t.Fatalf("history changed when adding the system message: %#v", got)
		}
		next := withTimeContext(msgs, now.Add(24*time.Hour), time.UTC)
		if !strings.Contains(next[0].Content, "2026-11-02T18:30:00Z (Monday, UTC)") || strings.Count(next[0].Content, "Current date and time:") != 1 {
			t.Fatalf("clock was stale or repeated: %#v", next)
		}
	}
}

func TestTimeContextUsesWorkspaceTimezone(t *testing.T) {
	now := time.Date(2026, time.November, 9, 20, 30, 0, 0, time.UTC)
	got := withTimeContext(nil, now, time.FixedZone("Asia/Kolkata", 5*3600+30*60))[0].Content
	for _, want := range []string{
		"Current date and time: 2026-11-09T20:30:00Z (Monday, UTC).",
		"Workspace time zone: Asia/Kolkata. Local date and time: 2026-11-10T02:00:00+05:30 (Tuesday).",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("time context missing %q: %q", want, got)
		}
	}
	if utcOnly := withTimeContext(nil, now, time.UTC)[0].Content; strings.Contains(utcOnly, "Workspace time zone") {
		t.Fatalf("UTC workspace repeated the clock: %q", utcOnly)
	}
}

func TestCompletionRequestsIncludeCurrentTime(t *testing.T) {
	var requests [][]models.ChatMessage
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Messages []models.ChatMessage `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding request: %v", err)
		}
		requests = append(requests, body.Messages)
		w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer srv.Close()
	lo := logf.New(logf.Opts{})
	client := NewOpenAIClient(models.ProviderConfig{BaseURL: srv.URL, APIKey: "test"}, &lo, srv.Client(), time.UTC)
	before := time.Now().UTC().Truncate(time.Second)
	if _, err := client.SendPrompt(t.Context(), models.PromptPayload{SystemPrompt: "Summarize.", UserPrompt: "Conversation"}); err != nil {
		t.Fatal(err)
	}
	payload := models.ChatCompletionPayload{Messages: []models.ChatMessage{{Role: models.RoleSystem, Content: "Help."}, {Role: models.RoleUser, Content: "Hi"}}}
	for range 2 {
		if _, err := client.SendChatCompletion(t.Context(), payload); err != nil {
			t.Fatal(err)
		}
	}
	after := time.Now().UTC()
	if len(requests) != 3 || payload.Messages[0].Content != "Help." {
		t.Fatalf("unexpected requests or mutated input: %d, %#v", len(requests), payload.Messages)
	}
	for _, messages := range requests {
		if len(messages) != 2 || messages[0].Role != models.RoleSystem {
			t.Fatalf("unexpected request messages: %#v", messages)
		}
		_, clock, ok := strings.Cut(messages[0].Content, "Current date and time: ")
		stamp, _, _ := strings.Cut(clock, " ")
		at, err := time.Parse(time.RFC3339, stamp)
		if !ok || err != nil || at.Before(before) || at.After(after) || strings.Count(messages[0].Content, "Current date and time:") != 1 {
			t.Fatalf("missing or stale request clock: %q", messages[0].Content)
		}
	}
}
