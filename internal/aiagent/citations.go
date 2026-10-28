package aiagent

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	aimodels "github.com/abhinavxd/libredesk/internal/ai/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"golang.org/x/net/html"
)

var inlineCitationRegexp = regexp.MustCompile(`\[\[cite:([^\]]*)\]\]`)

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
	cited := map[int]bool{}
	answer = inlineCitationRegexp.ReplaceAllStringFunc(answer, func(marker string) string {
		id, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(marker, "[[cite:"), "]]"))
		if err != nil || !allowed[id] {
			return ""
		}
		if !cited[id] {
			articleIDs = append(articleIDs, id)
			cited[id] = true
		}
		return marker
	})
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
				if allowed[id] && !cited[id] {
					articleIDs = append(articleIDs, id)
					cited[id] = true
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

func articleCitationHTML(text string, articleIDs []int) string {
	allowed := make(map[int]bool, len(articleIDs))
	for _, id := range articleIDs {
		allowed[id] = true
	}
	tokens := html.NewTokenizer(strings.NewReader(stringutil.Markdown2HTML(text)))
	var content strings.Builder
	for {
		kind := tokens.Next()
		if kind == html.ErrorToken {
			break
		}
		raw := string(tokens.Raw())
		raw = inlineCitationRegexp.ReplaceAllStringFunc(raw, func(marker string) string {
			id, _ := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(marker, "[[cite:"), "]]"))
			if kind != html.TextToken || !allowed[id] {
				return ""
			}
			return "<!--ld-cite:" + strconv.Itoa(id) + "-->"
		})
		content.WriteString(raw)
	}
	return content.String()
}

func replyMeta(suggestions []string, articleIDs []int) map[string]any {
	meta := suggestedRepliesMeta(suggestions)
	if len(articleIDs) > 0 {
		meta["ai_article_ids"] = articleIDs
	}
	return meta
}

func numberedCitationText(text string, articleIDs []int) string {
	numbers := make(map[int]int, len(articleIDs))
	for i, id := range articleIDs {
		numbers[id] = i + 1
	}
	return inlineCitationRegexp.ReplaceAllStringFunc(text, func(marker string) string {
		id, _ := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(marker, "[[cite:"), "]]"))
		if number := numbers[id]; number > 0 {
			return "(" + strconv.Itoa(number) + ")"
		}
		return ""
	})
}
