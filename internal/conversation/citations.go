package conversation

import (
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	hcmodels "github.com/abhinavxd/libredesk/internal/helpcenter/models"
)

var articleCitationRegexp = regexp.MustCompile(`<!--ld-cite:(\d+)-->`)

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
	message.Content = chatArticleReferencesHTML(message.Content, m.lookupArticleReferences(message.Meta)[0])
}

func (m *Manager) RenderMessagesArticleReferences(messages []models.Message) {
	for i, references := range m.lookupArticleReferences(messageMetas(messages)...) {
		messages[i].Content = chatArticleReferencesHTML(messages[i].Content, references)
	}
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

func chatArticleReferencesHTML(content string, references []hcmodels.ArticleReference) string {
	byID := make(map[int]int, len(references))
	for i, reference := range references {
		byID[reference.ID] = i
	}
	used := map[int]bool{}
	content = articleCitationRegexp.ReplaceAllStringFunc(content, func(marker string) string {
		id, _ := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(marker, "<!--ld-cite:"), "-->"))
		i, ok := byID[id]
		if !ok {
			return ""
		}
		used[id] = true
		return articleReferenceLinkHTML(references[i], i+1)
	})
	var remaining strings.Builder
	for i, reference := range references {
		if !used[reference.ID] {
			if remaining.Len() > 0 {
				remaining.WriteByte(' ')
			}
			remaining.WriteString(articleReferenceLinkHTML(reference, i+1))
		}
	}
	if remaining.Len() > 0 {
		content += "<p>" + remaining.String() + "</p>"
	}
	return content
}

func articleReferenceLinkHTML(reference hcmodels.ArticleReference, number int) string {
	return fmt.Sprintf(`<sup><a class="ld-article-citation" href="%s" title="%s" aria-label="%s" target="_blank" rel="noopener noreferrer">(%d)</a></sup>`, html.EscapeString(reference.URL), html.EscapeString(reference.Title), html.EscapeString(reference.Title), number)
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
