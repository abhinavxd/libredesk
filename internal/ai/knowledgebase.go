package ai

import (
	"database/sql"

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

func (m *Manager) CreateKnowledgeBaseItem(title, content string, enabled bool) (models.KnowledgeBaseItem, error) {
	var item models.KnowledgeBaseItem
	if err := m.q.InsertKnowledgeBaseItem.Get(&item, models.KnowledgeTypeSnippet, title, content, enabled); err != nil {
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
	if !item.Enabled {
		if err := m.RemoveEmbeddings(models.SourceSnippet, item.ID); err != nil {
			m.lo.Error("error removing snippet embeddings", "error", err)
		}
		return
	}
	if err := m.Reindex(models.SourceSnippet, item.ID, item.Title, item.Content); err != nil {
		m.lo.Error("error indexing snippet", "error", err, "id", item.ID)
	}
}

// ReindexAll re-embeds every knowledge base snippet in the background, e.g. after the embedding model changed.
func (m *Manager) ReindexAll() {
	go func() {
		items, err := m.GetKnowledgeBaseItems()
		if err != nil {
			m.lo.Error("error fetching knowledge base items for reindex", "error", err)
			return
		}
		for _, item := range items {
			m.reindexSnippetSync(item)
		}
	}()
}
