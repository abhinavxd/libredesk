package ai

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/ai/models"
)

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
