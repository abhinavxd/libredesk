package ai

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"

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

func (m *Manager) CreateKnowledgeBaseItem(title, content, source string, enabled bool) (models.KnowledgeBaseItem, error) {
	if source == "" {
		source = models.KnowledgeSourceManual
	}
	var item models.KnowledgeBaseItem
	if err := m.q.InsertKnowledgeBaseItem.Get(&item, models.KnowledgeTypeSnippet, title, content, enabled, source); err != nil {
		m.lo.Error("error creating knowledge base item", "error", err)
		return item, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	m.reindexSnippet(item)
	return item, nil
}

func (m *Manager) UpdateKnowledgeBaseItem(id int, title, content string, enabled bool) (models.KnowledgeBaseItem, error) {
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

func (m *Manager) DeleteKnowledgeBaseItem(id int) error {
	if _, err := m.q.DeleteKnowledgeBaseItem.Exec(id); err != nil {
		m.lo.Error("error deleting knowledge base item", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if err := m.RemoveEmbeddings(models.SourceSnippet, id); err != nil {
		m.lo.Error("error removing snippet embeddings", "error", err)
	}
	return nil
}

// reindexSnippet embeds an enabled snippet (or drops its vectors when disabled) in the background.
func (m *Manager) reindexSnippet(item models.KnowledgeBaseItem) {
	go m.reindexSnippetSync(item)
}

func (m *Manager) reindexSnippetSync(item models.KnowledgeBaseItem) {
	cfg, err := m.getRawProviderConfig(models.ProviderTypeEmbedding)
	if err != nil {
		return
	}
	m.reindexSnippetWith(item, cfg.Model, cfg.Dimensions)
}

// reindexSnippetWith embeds an enabled snippet (or drops its vectors when disabled) and records the
// fingerprint on success. On failure the stored fingerprint stays stale, so reconcile retries later.
func (m *Manager) reindexSnippetWith(item models.KnowledgeBaseItem, model string, dimensions int) {
	if !item.Enabled {
		if err := m.RemoveEmbeddings(models.SourceSnippet, item.ID); err != nil {
			m.lo.Error("error removing snippet embeddings", "error", err)
			return
		}
		m.setSnippetFingerprint(item.ID, "")
		return
	}
	if err := m.Reindex(models.SourceSnippet, item.ID, item.Title, item.Content); err != nil {
		m.lo.Error("error indexing snippet", "error", err, "id", item.ID)
		return
	}
	m.setSnippetFingerprint(item.ID, snippetFingerprint(item, model, dimensions))
}

func (m *Manager) setSnippetFingerprint(id int, fingerprint string) {
	if _, err := m.q.SetKnowledgeBaseFingerprint.Exec(id, fingerprint); err != nil {
		m.lo.Error("error setting snippet embedded fingerprint", "error", err, "id", id)
	}
}

// ReindexAll triggers a reconcile so snippets are re-embedded against the current model, e.g. after the embedding model changed.
func (m *Manager) ReindexAll() {
	go m.reconcile()
}

// snippetFingerprint signs the content and embedding model a snippet was last embedded against; any
// change (edited content, switched model, changed dimensions) yields a new value and triggers reindex.
func snippetFingerprint(item models.KnowledgeBaseItem, model string, dimensions int) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s\x00%s\x00%s\x00%d", item.Title, item.Content, model, dimensions)
	return hex.EncodeToString(h.Sum(nil))
}
