package ai

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/abhinavxd/libredesk/internal/ai/models"
)

func TestDiffTagIndex(t *testing.T) {
	indexed := map[int]indexedChunk{
		1: {sourceType: models.SourceTag, sourceID: 1, chunkText: "billing", vec: []float32{1, 0}},
		2: {sourceType: models.SourceTag, sourceID: 2, chunkText: "old-name", vec: []float32{0, 1}},
		3: {sourceType: models.SourceTag, sourceID: 3, chunkText: "deleted", vec: []float32{1, 1}},
		4: {sourceType: models.SourceTag, sourceID: 4, chunkText: "blanked", vec: []float32{1, 1}},
	}
	tags := []models.TagRef{
		{ID: 1, Name: "billing"},
		{ID: 2, Name: "new-name"},
		{ID: 4, Name: "  "},
		{ID: 6, Name: "kyc"},
	}

	stale, removed := diffTagIndex(tags, indexed)

	wantStale := []models.TagRef{{ID: 2, Name: "new-name"}, {ID: 6, Name: "kyc"}}
	if !reflect.DeepEqual(stale, wantStale) {
		t.Errorf("stale = %v, want %v", stale, wantStale)
	}
	if want := []int{3, 4}; !reflect.DeepEqual(removed, want) {
		t.Errorf("removed = %v, want %v", removed, want)
	}
}

func TestIndexSearchFiltersBySourceType(t *testing.T) {
	ix := newEmbeddingIndex()
	ix.replaceAll([]indexedChunk{
		{sourceType: models.SourceSnippet, sourceID: 1, chunkText: "snippet", vec: []float32{1, 0}, norm: 1},
		{sourceType: models.SourceTag, sourceID: 2, chunkText: "tag", vec: []float32{1, 0}, norm: 1},
	})

	results, _ := ix.search([]float32{1, 0}, 10, models.SourceTag)
	if len(results) != 1 || results[0].ChunkText != "tag" {
		t.Fatalf("results = %v, want only the tag chunk", results)
	}
}

func TestIndexReplaceSourceIDs(t *testing.T) {
	ix := newEmbeddingIndex()
	ix.replaceAll([]indexedChunk{
		{sourceType: models.SourceSnippet, sourceID: 1, chunkText: "snippet"},
		{sourceType: models.SourceTag, sourceID: 1, chunkText: "keep"},
		{sourceType: models.SourceTag, sourceID: 2, chunkText: "drop"},
		{sourceType: models.SourceTag, sourceID: 3, chunkText: "rename"},
	})

	ix.replaceSourceIDs(models.SourceTag, []int{2, 3}, []indexedChunk{
		{sourceType: models.SourceTag, sourceID: 3, chunkText: "renamed"},
	})

	got := map[string]string{}
	for _, c := range ix.chunks {
		got[fmt.Sprintf("%s:%d", c.sourceType, c.sourceID)] = c.chunkText
	}
	want := map[string]string{"snippet:1": "snippet", "tag:1": "keep", "tag:3": "renamed"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("chunks = %v, want %v", got, want)
	}
}
