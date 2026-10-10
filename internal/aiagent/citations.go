package aiagent

import (
	"encoding/json"
	"strings"

	aimodels "github.com/abhinavxd/libredesk/internal/ai/models"
)

func splitArticleSources(answer string, hits []aimodels.SearchResult, enabled bool) (string, []int) {
	allowed := map[int]bool{}
	if enabled {
		for _, hit := range hits {
			if hit.SourceType == aimodels.SourceHelpArticle && hit.SourceID > 0 {
				allowed[hit.SourceID] = true
			}
		}
	}
	var articleIDs []int
	var body strings.Builder
	for {
		before, payload, found := strings.Cut(answer, sourcesMarker)
		if !found {
			body.WriteString(answer)
			break
		}
		body.WriteString(before)
		payload = strings.TrimLeft(payload, " \t\r\n")
		decoder := json.NewDecoder(strings.NewReader(payload))
		var ids []int
		err := decoder.Decode(&ids)
		end := int(decoder.InputOffset())
		if end == 0 {
			end = len(payload)
			if strings.HasPrefix(payload, "[") {
				if close := strings.IndexByte(payload, ']'); close >= 0 {
					end = close + 1
				}
			} else if newline := strings.IndexByte(payload, '\n'); newline >= 0 {
				end = newline
			}
			for _, marker := range []string{confirmMarker, suggestionsMarker, sourcesMarker} {
				if next := strings.Index(payload, marker); next >= 0 {
					end = min(end, next)
				}
			}
		}
		if err == nil {
			for _, id := range ids {
				if allowed[id] {
					articleIDs = append(articleIDs, id)
					delete(allowed, id)
				}
			}
		}
		answer = strings.TrimLeft(payload[end:], " \t\r")
		if strings.HasSuffix(strings.TrimRight(before, " \t\r"), "\n") {
			answer = strings.TrimPrefix(answer, "\n")
		}
	}
	return strings.TrimSpace(body.String()), articleIDs
}

func replyMeta(suggestions []string, articleIDs []int) map[string]any {
	meta := suggestedRepliesMeta(suggestions)
	if len(articleIDs) > 0 {
		meta["ai_article_ids"] = articleIDs
	}
	return meta
}
