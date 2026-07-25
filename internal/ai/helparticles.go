package ai

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"

	"github.com/abhinavxd/libredesk/internal/ai/models"
)

// ReindexHelpArticle re-syncs a help article's embeddings in the background, indexing or removing
// based on its current status and AI flag.
func (m *Manager) ReindexHelpArticle(id int) {
	go func() {
		var item models.HelpArticleItem
		if err := m.q.GetEmbeddableHelpArticle.Get(&item, id); err != nil {
			if err != sql.ErrNoRows {
				m.lo.Error("error fetching help article for reindex", "error", err, "id", id)
			}
			return
		}
		cfg, err := m.getRawProviderConfig(models.ProviderTypeEmbedding)
		if err != nil {
			return
		}
		m.reindexHelpArticleWith(item, cfg.Model, cfg.Dimensions)
	}()
}

// RemoveHelpArticleEmbeddings drops a help article's vectors from the DB and memory.
func (m *Manager) RemoveHelpArticleEmbeddings(id int) error {
	return m.RemoveEmbeddings(models.SourceHelpArticle, id)
}

// reindexHelpArticleWith embeds an eligible article (or drops its vectors when ineligible) and records
// the fingerprint on success. On failure the stored fingerprint stays stale, so reconcile retries later.
func (m *Manager) reindexHelpArticleWith(item models.HelpArticleItem, model string, dimensions int) {
	if !helpArticleEligible(item) {
		if err := m.RemoveEmbeddings(models.SourceHelpArticle, item.ID); err != nil {
			m.lo.Error("error removing help article embeddings", "error", err, "id", item.ID)
			return
		}
		m.setHelpArticleFingerprint(item.ID, "")
		return
	}
	if err := m.Reindex(models.SourceHelpArticle, item.ID, item.Title, item.Content); err != nil {
		m.lo.Error("error indexing help article", "error", err, "id", item.ID)
		return
	}
	m.setHelpArticleFingerprint(item.ID, helpArticleFingerprint(item, model, dimensions))
}

func (m *Manager) setHelpArticleFingerprint(id int, fingerprint string) {
	if _, err := m.q.SetHelpArticleFingerprint.Exec(id, fingerprint); err != nil {
		m.lo.Error("error setting help article embedded fingerprint", "error", err, "id", id)
	}
}

// reconcileHelpArticles re-embeds articles whose fingerprint is stale and sweeps embeddings orphaned
// by cascade deletes (collection/help center removal).
func (m *Manager) reconcileHelpArticles(model string, dimensions int) {
	items := make([]models.HelpArticleItem, 0)
	if err := m.q.GetEmbeddableHelpArticles.Select(&items); err != nil {
		m.lo.Error("error fetching help articles for reconcile", "error", err)
		return
	}

	reindexed := 0
	for _, item := range items {
		if !helpArticleEligible(item) {
			if item.EmbeddedFingerprint != "" {
				m.reindexHelpArticleWith(item, model, dimensions)
			}
			continue
		}
		if item.EmbeddedFingerprint == helpArticleFingerprint(item, model, dimensions) {
			continue
		}
		m.reindexHelpArticleWith(item, model, dimensions)
		reindexed++
	}
	if reindexed > 0 {
		m.lo.Info("reconciled help article embeddings", "reindexed", reindexed)
	}

	var orphanIDs []int
	if err := m.q.DeleteOrphanArticleVectors.Select(&orphanIDs); err != nil {
		m.lo.Error("error sweeping orphan help article embeddings", "error", err)
		return
	}
	seen := make(map[int]struct{}, len(orphanIDs))
	for _, id := range orphanIDs {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		m.index.removeSource(models.SourceHelpArticle, id)
	}
	if len(seen) > 0 {
		m.lo.Info("swept orphan help article embeddings", "articles", len(seen))
	}
}

// helpArticleEligible reports whether an article belongs in the knowledge base.
func helpArticleEligible(item models.HelpArticleItem) bool {
	return item.Status == "published" && item.AIEnabled
}

// helpArticleFingerprint signs the content and embedding model an article was last embedded against.
func helpArticleFingerprint(item models.HelpArticleItem, model string, dimensions int) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s\x00%s\x00%s\x00%d", item.Title, item.Content, model, dimensions)
	return hex.EncodeToString(h.Sum(nil))
}
