package ai

import (
	"context"
	"slices"
	"strings"

	"github.com/abhinavxd/libredesk/internal/ai/models"
)

// reconcileTags embeds tags missing from the index, re-embeds renamed ones and drops the vectors of deleted tags.
func (m *Manager) reconcileTags(ctx context.Context) {
	tags, err := m.getTags()
	if err != nil {
		return
	}

	// Before the boot load finishes every tag reads as unindexed, which would re-embed the whole list.
	select {
	case <-m.indexReady:
	case <-ctx.Done():
		return
	}

	gen := m.tagGen.Load()
	stale, removed := diffTagIndex(tags, m.index.chunksBySourceID(models.SourceTag))
	if len(stale) == 0 && len(removed) == 0 {
		return
	}

	touched := slices.Clone(removed)
	chunks := make([]indexedChunk, 0, len(stale))
	if len(stale) > 0 {
		names := tagNames(stale)
		vecs, err := m.GetEmbeddingsBatch(ctx, names)
		if err != nil {
			m.lo.Error("error embedding tags", "error", err, "count", len(names))
			return
		}
		for i, t := range stale {
			touched = append(touched, t.ID)
			chunks = append(chunks, indexedChunk{
				sourceType: models.SourceTag,
				sourceID:   t.ID,
				chunkText:  t.Name,
				vec:        vecs[i],
				norm:       norm(vecs[i]),
			})
		}
	}

	m.reindexMu.Lock()
	defer m.reindexMu.Unlock()
	if m.tagGen.Load() != gen {
		return
	}
	if err := m.commitEmbeddings(models.SourceTag, touched, chunks); err != nil {
		return
	}
	m.lo.Info("reconciled tag embeddings", "embedded", len(chunks), "removed", len(removed), "total", len(tags))
}

// purgeTagEmbeddings drops every tag vector so the next reconcile re-embeds the list with the current provider.
func (m *Manager) purgeTagEmbeddings() {
	m.reindexMu.Lock()
	defer m.reindexMu.Unlock()
	m.tagGen.Add(1)
	// Clearing the index even on a failed delete keeps this self-healing: the next reconcile re-embeds every tag and overwrites the rows left behind.
	m.index.removeSourceType(models.SourceTag)
	if _, err := m.q.DeleteEmbeddingsBySourceType.Exec(models.SourceTag); err != nil {
		m.lo.Error("error deleting tag embeddings", "error", err)
	}
}

// tagCandidates returns the current names of the tags most similar to text, most similar first; a name only the index still knows is dropped.
func (m *Manager) tagCandidates(ctx context.Context, text string, k int, tags []models.TagRef) ([]string, error) {
	results, err := m.searchSource(ctx, text, k, models.SourceTag)
	if err != nil {
		return nil, err
	}
	byID := make(map[int]string, len(tags))
	for _, t := range tags {
		byID[t.ID] = t.Name
	}
	names := make([]string, 0, len(results))
	for _, r := range results {
		if name, ok := byID[r.SourceID]; ok {
			names = append(names, name)
		}
	}
	return names, nil
}

func (m *Manager) getTags() ([]models.TagRef, error) {
	tags := make([]models.TagRef, 0)
	if err := m.q.GetTags.Select(&tags); err != nil {
		m.lo.Error("error fetching tags", "error", err)
		return nil, err
	}
	return tags, nil
}

func tagNames(tags []models.TagRef) []string {
	names := make([]string, 0, len(tags))
	for _, t := range tags {
		names = append(names, t.Name)
	}
	return names
}

// diffTagIndex reports which tags need embedding and which indexed source ids no longer belong; a blank name is dropped, not embedded.
func diffTagIndex(tags []models.TagRef, indexed map[int]indexedChunk) (stale []models.TagRef, removed []int) {
	live := make(map[int]bool, len(tags))
	for _, t := range tags {
		if strings.TrimSpace(t.Name) == "" {
			continue
		}
		live[t.ID] = true
		if c, ok := indexed[t.ID]; ok && c.chunkText == t.Name {
			continue
		}
		stale = append(stale, t)
	}
	for id := range indexed {
		if !live[id] {
			removed = append(removed, id)
		}
	}
	slices.Sort(removed)
	return stale, removed
}
