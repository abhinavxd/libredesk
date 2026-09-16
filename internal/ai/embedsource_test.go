package ai

import (
	"testing"
)

func TestItemFingerprint(t *testing.T) {
	item := embedItem{Title: "Refunds", Content: "We refund within 30 days."}
	base := itemFingerprint(item, "https://api.openai.com/v1", "text-embedding-3-small", 1536)

	if itemFingerprint(item, "https://api.openai.com/v1", "text-embedding-3-small", 1536) != base {
		t.Fatal("fingerprint must be stable for identical inputs")
	}

	// Every field that makes stored vectors comparable must change the fingerprint, or a reindex is skipped when it shouldn't be.
	cases := map[string]string{
		"base URL changed":   itemFingerprint(item, "https://llm.internal/v1", "text-embedding-3-small", 1536),
		"model changed":      itemFingerprint(item, "https://api.openai.com/v1", "text-embedding-3-large", 1536),
		"dimensions changed": itemFingerprint(item, "https://api.openai.com/v1", "text-embedding-3-small", 3072),
		"title changed":      itemFingerprint(embedItem{Title: "Returns", Content: item.Content}, "https://api.openai.com/v1", "text-embedding-3-small", 1536),
		"content changed":    itemFingerprint(embedItem{Title: item.Title, Content: "We refund within 14 days."}, "https://api.openai.com/v1", "text-embedding-3-small", 1536),
	}
	for name, fp := range cases {
		if fp == base {
			t.Errorf("%s: fingerprint should differ from the baseline", name)
		}
	}

	// Eligibility and id are lifecycle state, not embedded content; they must not force a reindex.
	varied := embedItem{ID: 42, Title: item.Title, Content: item.Content, Fingerprint: "stale", Eligible: true}
	if itemFingerprint(varied, "https://api.openai.com/v1", "text-embedding-3-small", 1536) != base {
		t.Error("fingerprint must depend only on title, content and provider identity")
	}
}
