package conversation

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	hcmodels "github.com/abhinavxd/libredesk/internal/helpcenter/models"
)

func (m *Manager) SetArticleReferenceStore(store articleReferenceStore) {
	m.articleReferenceStore = store
}

func (m *Manager) GetArticleReferences(ids []int) ([]hcmodels.ArticleReference, error) {
	if m.articleReferenceStore == nil || len(ids) == 0 {
		return nil, nil
	}
	references, err := m.articleReferenceStore.GetArticleReferences(ids)
	if err != nil {
		return nil, err
	}
	rootURL, err := m.settingsStore.GetAppRootURL()
	if err != nil {
		return nil, err
	}
	resolved := make([]hcmodels.ArticleReference, 0, len(references))
	for _, reference := range references {
		reference.URL = reference.PublicURL(rootURL)
		if reference.URL != "" {
			resolved = append(resolved, reference)
		}
	}
	return resolved, nil
}

func (m *Manager) RenderArticleReferences(message *models.Message) {
	message.Content += m.chatArticleReferencesHTML(m.lookupArticleReferences(message.Meta)[0])
}

func (m *Manager) RenderMessagesArticleReferences(messages []models.Message) {
	for i, references := range m.lookupArticleReferences(messageMetas(messages)...) {
		messages[i].Content += m.chatArticleReferencesHTML(references)
	}
}

func (m *Manager) chatArticleReferencesHTML(references []hcmodels.ArticleReference) string {
	return articleReferencesHTML(references, m.i18n.Tc("globals.terms.articleReference", 2 /* n */))
}

func (m *Manager) emailArticleReferences(message *models.Message) string {
	return emailArticleReferencesHTML(m.lookupArticleReferences(message.Meta)[0])
}

// lookupArticleReferences returns each message's references, resolved with one query for all of them.
func (m *Manager) lookupArticleReferences(metas ...json.RawMessage) [][]hcmodels.ArticleReference {
	out := make([][]hcmodels.ArticleReference, len(metas))
	idsByMessage := make([][]int, len(metas))
	var ids []int
	for i, meta := range metas {
		idsByMessage[i] = messageArticleIDs(meta)
		ids = append(ids, idsByMessage[i]...)
	}
	if len(ids) == 0 {
		return out
	}
	references, err := m.GetArticleReferences(ids)
	if err != nil {
		m.lo.Error("error rendering article references", "error", err)
		return out
	}
	byID := make(map[int]hcmodels.ArticleReference, len(references))
	for _, reference := range references {
		byID[reference.ID] = reference
	}
	for i, messageIDs := range idsByMessage {
		for _, id := range messageIDs {
			if reference, ok := byID[id]; ok {
				out[i] = append(out[i], reference)
			}
		}
	}
	return out
}

func messageMetas(messages []models.Message) []json.RawMessage {
	metas := make([]json.RawMessage, len(messages))
	for i := range messages {
		metas[i] = messages[i].Meta
	}
	return metas
}

func messageArticleIDs(meta json.RawMessage) []int {
	var data struct {
		AssistantID int   `json:"ai_assistant_id"`
		ArticleIDs  []int `json:"ai_article_ids"`
	}
	if json.Unmarshal(meta, &data) != nil || data.AssistantID <= 0 {
		return nil
	}
	ids := make([]int, 0, len(data.ArticleIDs))
	seen := map[int]bool{}
	for _, id := range data.ArticleIDs {
		if id > 0 && !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	return ids
}

func articleReferencesHTML(references []hcmodels.ArticleReference, label string) string {
	if len(references) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("<p>")
	b.WriteString(html.EscapeString(label))
	b.WriteString("</p><ul>")
	for _, reference := range references {
		b.WriteString(`<li><a href="`)
		b.WriteString(html.EscapeString(reference.URL))
		b.WriteString(`" target="_blank" rel="noopener noreferrer">`)
		b.WriteString(html.EscapeString(reference.Title))
		b.WriteString("</a></li>")
	}
	b.WriteString("</ul>")
	return b.String()
}

func emailArticleReferencesHTML(references []hcmodels.ArticleReference) string {
	if len(references) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("<p>")
	for i, reference := range references {
		if i > 0 {
			b.WriteByte(' ')
		}
		fmt.Fprintf(&b, `<a href="%s" target="_blank" rel="noopener noreferrer">(%d)</a>`, html.EscapeString(reference.URL), i+1)
	}
	b.WriteString("</p>")
	return b.String()
}
