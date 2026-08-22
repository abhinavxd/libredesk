package ai

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/abhinavxd/libredesk/internal/ai/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
)

// embedItem is one indexable row, normalized across the content types that can be embedded.
type embedItem struct {
	ID          int
	Title       string
	Content     string
	Fingerprint string
	Eligible    bool
}

// genKey identifies a reindex generation counter for one row of one source type.
type genKey struct {
	sourceType string
	id         int
}

// embedSource is one kind of embeddable content; it supplies the rows and owns their fingerprint column.
type embedSource interface {
	sourceType() string
	list() ([]embedItem, error)
	get(id int) (embedItem, error)
	exists(id int) (bool, error)
	setFingerprint(id int, fingerprint string)
	// deleteOrphans drops stored vectors whose row is gone and returns the affected ids.
	deleteOrphans() ([]int, error)
}

// reindexItem embeds an item in the background. The generation is taken now so a later
// update always supersedes this job, however long the provider takes.
func (m *Manager) reindexItem(src embedSource, item embedItem) {
	gen := m.nextGen(src.sourceType(), item.ID)
	m.runEmbedJob(func(baseURL, model string, dimensions int) {
		m.reindexItemWith(m.ctx, src, item, baseURL, model, dimensions, gen)
	})
}

// reindexItemByID fetches the item, then embeds it in the background. The generation is
// taken at enqueue time so jobs commit in the order the edits were made.
func (m *Manager) reindexItemByID(src embedSource, id int) {
	gen := m.nextGen(src.sourceType(), id)
	m.runEmbedJob(func(baseURL, model string, dimensions int) {
		item, err := src.get(id)
		if err != nil {
			return
		}
		m.reindexItemWith(m.ctx, src, item, baseURL, model, dimensions, gen)
	})
}

// reindexItemWith embeds an eligible item (or drops its vectors when ineligible), gated by gen so an older job can't overwrite a newer one.
func (m *Manager) reindexItemWith(ctx context.Context, src embedSource, item embedItem, baseURL, model string, dimensions int, gen uint64) {
	m.lo.Debug("reindex item", "source_type", src.sourceType(), "id", item.ID, "eligible", item.Eligible, "model", model, "dimensions", dimensions)
	if !item.Eligible {
		m.reindexMu.Lock()
		defer m.reindexMu.Unlock()
		if !m.canCommit(src, item.ID, gen) {
			return
		}
		if err := m.removeEmbeddings(src.sourceType(), item.ID); err != nil {
			m.lo.Error("error removing embeddings", "error", err, "source_type", src.sourceType(), "id", item.ID)
			return
		}
		src.setFingerprint(item.ID, "")
		m.lo.Debug("reindex removed embeddings", "source_type", src.sourceType(), "id", item.ID)
		return
	}

	// Unchanged content on the same model is already embedded; don't pay the provider again.
	fingerprint := itemFingerprint(item, baseURL, model, dimensions)
	if item.Fingerprint == fingerprint {
		m.lo.Debug("reindex skipped, fingerprint unchanged", "source_type", src.sourceType(), "id", item.ID)
		return
	}

	indexed, err := m.embedSource(ctx, src.sourceType(), item.ID, item.Title, item.Content)
	if err != nil {
		m.lo.Error("error indexing content", "error", err, "source_type", src.sourceType(), "id", item.ID)
		return
	}

	m.reindexMu.Lock()
	defer m.reindexMu.Unlock()
	if !m.canCommit(src, item.ID, gen) {
		return
	}
	if err := m.commitEmbeddings(src.sourceType(), []int{item.ID}, indexed); err != nil {
		m.lo.Error("error indexing content", "error", err, "source_type", src.sourceType(), "id", item.ID)
		return
	}
	src.setFingerprint(item.ID, fingerprint)
	m.lo.Debug("reindex embedded", "source_type", src.sourceType(), "id", item.ID, "chunks", len(indexed))
}

// reconcileSource re-embeds every row whose stored fingerprint no longer matches its content and the active model.
func (m *Manager) reconcileSource(ctx context.Context, src embedSource, baseURL, model string, dimensions int) {
	items, err := src.list()
	if err != nil {
		return
	}

	reindexed := 0
	for _, item := range items {
		if ctx.Err() != nil {
			return
		}
		if !item.Eligible {
			// An ineligible item should carry no embeddings; clean up any left behind.
			if item.Fingerprint != "" {
				m.reindexItemWith(ctx, src, item, baseURL, model, dimensions, m.nextGen(src.sourceType(), item.ID))
			}
			continue
		}
		if item.Fingerprint == itemFingerprint(item, baseURL, model, dimensions) {
			continue
		}
		m.reindexItemWith(ctx, src, item, baseURL, model, dimensions, m.nextGen(src.sourceType(), item.ID))
		reindexed++
	}
	if reindexed > 0 {
		m.lo.Info("reconciled embeddings", "source_type", src.sourceType(), "reindexed", reindexed)
	}

	m.sweepOrphans(src)
}

// sweepOrphans drops in-memory vectors for rows deleted outside the source's own delete path,
// e.g. by a cascade from a parent collection or help center.
func (m *Manager) sweepOrphans(src embedSource) {
	// Held across the sweep so an in-flight job can't pass canCommit before the sweep and
	// insert its vectors after it.
	m.reindexMu.Lock()
	defer m.reindexMu.Unlock()

	ids, err := src.deleteOrphans()
	if err != nil {
		m.lo.Error("error sweeping orphan embeddings", "error", err, "source_type", src.sourceType())
		return
	}
	seen := make(map[int]struct{}, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		m.dropGen(src.sourceType(), id)
		m.index.removeSource(src.sourceType(), id)
	}
	if len(seen) > 0 {
		m.lo.Info("swept orphan embeddings", "source_type", src.sourceType(), "count", len(seen))
	}
}

// runEmbedJob runs an embed job in the background, bounded by the concurrency cap and tracked
// by the wait group so shutdown drains in-flight work.
func (m *Manager) runEmbedJob(fn func(baseURL, model string, dimensions int)) {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		select {
		case m.embedSem <- struct{}{}:
		case <-m.ctx.Done():
			return
		}
		defer func() { <-m.embedSem }()

		cfg, err := m.getRawProviderConfig(models.ProviderTypeEmbedding)
		if err != nil {
			return
		}
		fn(cfg.BaseURL, cfg.Model, cfg.Dimensions)
	}()
}

// nextGen bumps and returns the reindex generation for a row; a job only commits if its gen is still the latest.
func (m *Manager) nextGen(sourceType string, id int) uint64 {
	k := genKey{sourceType, id}
	m.genMu.Lock()
	defer m.genMu.Unlock()
	m.gen[k]++
	return m.gen[k]
}

func (m *Manager) isLatestGen(sourceType string, id int, gen uint64) bool {
	m.genMu.Lock()
	defer m.genMu.Unlock()
	return m.gen[genKey{sourceType, id}] == gen
}

func (m *Manager) dropGen(sourceType string, id int) {
	m.genMu.Lock()
	defer m.genMu.Unlock()
	delete(m.gen, genKey{sourceType, id})
}

// canCommit reports whether a reindex commit may proceed: gen is still the latest and the row still
// exists (a delete between snapshot and commit would otherwise be resurrected). Caller must hold reindexMu.
func (m *Manager) canCommit(src embedSource, id int, gen uint64) bool {
	if !m.isLatestGen(src.sourceType(), id, gen) {
		return false
	}
	exists, err := src.exists(id)
	if err != nil {
		m.lo.Error("error checking row existence", "error", err, "source_type", src.sourceType(), "id", id)
		return false
	}
	if !exists {
		m.dropGen(src.sourceType(), id)
		return false
	}
	return true
}

// embedSources returns every content type the reconciler keeps embedded.
func (m *Manager) embedSources() []embedSource {
	return []embedSource{snippetSource{m}, helpArticleSource{m}}
}

// Base URL is signed too, so re-pointing the provider reindexes even when the model name is unchanged.
func itemFingerprint(item embedItem, baseURL, model string, dimensions int) string {
	sum := sha256.Sum256(fmt.Appendf(nil, "%s\x00%s\x00%s\x00%s\x00%d\x00%d", item.Title, item.Content, baseURL, model, dimensions, stringutil.ChunkerVersion))
	return hex.EncodeToString(sum[:])
}
