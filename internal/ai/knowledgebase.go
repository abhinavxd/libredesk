package ai

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/abhinavxd/libredesk/internal/ai/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
)

// GetKnowledgeBaseItems returns all snippet knowledge base items.
func (m *Manager) GetKnowledgeBaseItems() ([]models.KnowledgeBaseItem, error) {
	items := make([]models.KnowledgeBaseItem, 0)
	if err := m.q.GetKnowledgeBaseItems.Select(&items); err != nil {
		m.lo.Error("error fetching knowledge base items", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return items, nil
}

func (m *Manager) GetKnowledgeBaseItem(id int) (models.KnowledgeBaseItem, error) {
	var item models.KnowledgeBaseItem
	if err := m.q.GetKnowledgeBaseItem.Get(&item, id); err != nil {
		if err == sql.ErrNoRows {
			return item, envelope.NewError(envelope.NotFoundError, m.i18n.T("globals.messages.notFound"), nil)
		}
		m.lo.Error("error fetching knowledge base item", "error", err)
		return item, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return item, nil
}

func (m *Manager) CreateKnowledgeBaseItem(title, content, source, sourceURL string, enabled bool) (models.KnowledgeBaseItem, error) {
	if strings.TrimSpace(content) == "" {
		return models.KnowledgeBaseItem{}, envelope.NewError(envelope.InputError, m.i18n.Ts("globals.messages.empty", "name", m.i18n.T("globals.terms.content")), nil)
	}
	if source == "" {
		source = models.KnowledgeSourceManual
	}
	var item models.KnowledgeBaseItem
	if err := m.q.InsertKnowledgeBaseItem.Get(&item, models.KnowledgeTypeSnippet, title, content, enabled, source, sourceURL); err != nil {
		m.lo.Error("error creating knowledge base item", "error", err)
		return item, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	m.reindexSnippet(item)
	return item, nil
}

func (m *Manager) UpdateKnowledgeBaseItem(id int, title, content string, enabled bool) (models.KnowledgeBaseItem, error) {
	if strings.TrimSpace(content) == "" {
		return models.KnowledgeBaseItem{}, envelope.NewError(envelope.InputError, m.i18n.Ts("globals.messages.empty", "name", m.i18n.T("globals.terms.content")), nil)
	}
	var item models.KnowledgeBaseItem
	if err := m.q.UpdateKnowledgeBaseItem.Get(&item, id, title, content, enabled); err != nil {
		if err == sql.ErrNoRows {
			return item, envelope.NewError(envelope.NotFoundError, m.i18n.T("globals.messages.notFound"), nil)
		}
		m.lo.Error("error updating knowledge base item", "error", err)
		return item, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	m.reindexSnippet(item)
	return item, nil
}

// DeleteKnowledgeBaseItem removes the snippet and its embeddings in one transaction.
func (m *Manager) DeleteKnowledgeBaseItem(id int) error {
	m.reindexMu.Lock()
	defer m.reindexMu.Unlock()

	// Supersede any in-flight reindex for this snippet so a slower embed can't re-insert its vectors
	// after the delete commits.
	m.dropSnippetGen(id)

	tx, err := m.db.BeginTxx(context.Background(), &sql.TxOptions{})
	if err != nil {
		m.lo.Error("error beginning knowledge base delete transaction", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.Stmtx(m.q.DeleteKnowledgeBaseItem).Exec(id); err != nil {
		m.lo.Error("error deleting knowledge base item", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if _, err := tx.Stmtx(m.q.DeleteEmbeddingsBySource).Exec(models.SourceSnippet, id); err != nil {
		m.lo.Error("error deleting snippet embeddings", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if err := tx.Commit(); err != nil {
		m.lo.Error("error committing knowledge base delete transaction", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	m.index.removeSource(models.SourceSnippet, id)
	return nil
}

// reindexSnippet embeds an enabled snippet (or drops its vectors when disabled) in the background.
func (m *Manager) reindexSnippet(item models.KnowledgeBaseItem) {
	gen := m.nextSnippetGen(item.ID)
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.embedSem <- struct{}{}
		defer func() { <-m.embedSem }()
		m.reindexSnippetSync(m.ctx, item, gen)
	}()
}

func (m *Manager) reindexSnippetSync(ctx context.Context, item models.KnowledgeBaseItem, gen uint64) {
	cfg, err := m.getRawProviderConfig(models.ProviderTypeEmbedding)
	if err != nil {
		return
	}
	m.reindexSnippetWith(ctx, item, cfg.BaseURL, cfg.Model, cfg.Dimensions, gen)
}

// reindexSnippetWith embeds an enabled snippet (or drops its vectors when disabled), gated by gen so an older job can't overwrite a newer one.
func (m *Manager) reindexSnippetWith(ctx context.Context, item models.KnowledgeBaseItem, baseURL, model string, dimensions int, gen uint64) {
	if !item.Enabled {
		m.reindexMu.Lock()
		defer m.reindexMu.Unlock()
		if !m.canCommitSnippet(item.ID, gen) {
			return
		}
		if err := m.removeEmbeddings(models.SourceSnippet, item.ID); err != nil {
			m.lo.Error("error removing snippet embeddings", "error", err)
			return
		}
		m.setSnippetFingerprint(item.ID, "")
		return
	}

	indexed, err := m.embedSource(ctx, models.SourceSnippet, item.ID, item.Title, item.Content)
	if err != nil {
		m.lo.Error("error indexing snippet", "error", err, "id", item.ID)
		return
	}

	m.reindexMu.Lock()
	defer m.reindexMu.Unlock()
	if !m.canCommitSnippet(item.ID, gen) {
		return
	}
	if err := m.commitEmbeddings(models.SourceSnippet, item.ID, indexed); err != nil {
		m.lo.Error("error indexing snippet", "error", err, "id", item.ID)
		return
	}
	m.setSnippetFingerprint(item.ID, snippetFingerprint(item, baseURL, model, dimensions))
}

// nextSnippetGen bumps and returns the reindex generation for a snippet; a job only commits if its gen is still the latest.
func (m *Manager) nextSnippetGen(id int) uint64 {
	m.snippetGenMu.Lock()
	defer m.snippetGenMu.Unlock()
	m.snippetGen[id]++
	return m.snippetGen[id]
}

func (m *Manager) isLatestSnippetGen(id int, gen uint64) bool {
	m.snippetGenMu.Lock()
	defer m.snippetGenMu.Unlock()
	return m.snippetGen[id] == gen
}

// canCommitSnippet reports whether a reindex commit may proceed: gen is still the latest and the row still exists (a delete between snapshot and commit would otherwise be resurrected). Caller must hold reindexMu.
func (m *Manager) canCommitSnippet(id int, gen uint64) bool {
	if !m.isLatestSnippetGen(id, gen) {
		return false
	}
	var exists bool
	if err := m.q.KnowledgeBaseItemExists.Get(&exists, id); err != nil {
		m.lo.Error("error checking knowledge base item existence", "error", err, "id", id)
		return false
	}
	if !exists {
		m.dropSnippetGen(id)
		return false
	}
	return true
}

func (m *Manager) dropSnippetGen(id int) {
	m.snippetGenMu.Lock()
	defer m.snippetGenMu.Unlock()
	delete(m.snippetGen, id)
}

func (m *Manager) setSnippetFingerprint(id int, fingerprint string) {
	if _, err := m.q.SetKnowledgeBaseFingerprint.Exec(id, fingerprint); err != nil {
		m.lo.Error("error setting snippet embedded fingerprint", "error", err, "id", id)
	}
}

// ReindexAll triggers a reconcile so snippets are re-embedded against the current model, e.g. after the embedding model changed.
func (m *Manager) ReindexAll() {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.reconcile(m.ctx)
	}()
}

// snippetFingerprint signs the content and the full embedding provider identity (base URL, model, dimensions); base URL is included so re-pointing the provider triggers reindex even when the model name is unchanged.
func snippetFingerprint(item models.KnowledgeBaseItem, baseURL, model string, dimensions int) string {
	sum := sha256.Sum256(fmt.Appendf(nil, "%s\x00%s\x00%s\x00%s\x00%d", item.Title, item.Content, baseURL, model, dimensions))
	return hex.EncodeToString(sum[:])
}
