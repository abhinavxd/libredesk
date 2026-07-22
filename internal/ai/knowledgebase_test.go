package ai

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/ai/models"
)

func TestSnippetFingerprint(t *testing.T) {
	item := models.KnowledgeBaseItem{Title: "Refunds", Content: "We refund within 30 days."}
	base := snippetFingerprint(item, "https://api.openai.com/v1", "text-embedding-3-small", 1536)

	if snippetFingerprint(item, "https://api.openai.com/v1", "text-embedding-3-small", 1536) != base {
		t.Fatal("fingerprint must be stable for identical inputs")
	}

	// Every field that makes stored vectors comparable must change the fingerprint, or a reindex is skipped when it shouldn't be.
	cases := map[string]string{
		"base URL changed":   snippetFingerprint(item, "https://llm.internal/v1", "text-embedding-3-small", 1536),
		"model changed":      snippetFingerprint(item, "https://api.openai.com/v1", "text-embedding-3-large", 1536),
		"dimensions changed": snippetFingerprint(item, "https://api.openai.com/v1", "text-embedding-3-small", 3072),
		"title changed":      snippetFingerprint(models.KnowledgeBaseItem{Title: "Returns", Content: item.Content}, "https://api.openai.com/v1", "text-embedding-3-small", 1536),
		"content changed":    snippetFingerprint(models.KnowledgeBaseItem{Title: item.Title, Content: "We refund within 14 days."}, "https://api.openai.com/v1", "text-embedding-3-small", 1536),
	}
	for name, fp := range cases {
		if fp == base {
			t.Errorf("%s: fingerprint should differ from the baseline", name)
		}
	}
}
