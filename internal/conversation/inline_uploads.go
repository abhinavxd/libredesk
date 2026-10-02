package conversation

import (
	"html"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/jakedolan443/fernmail/internal/stringutil"
)

// normalizeInlineUploads recognizes explicit local references and authenticated
// legacy storage URLs. An arbitrary remote image containing a UUID is not an
// upload claim. Local/CID claims are validated atomically when saved or sent.
func (m *Manager) normalizeInlineUploads(content string, conversationID, userID int) (string, []string) {
	claims := []string{}
	seen := map[string]bool{}
	rewritten := imgSrcPattern.ReplaceAllStringFunc(content, func(match string) string {
		raw := imgSrcPattern.FindStringSubmatch(match)[1]
		var candidate string
		explicit := false
		if strings.HasPrefix(raw, "cid:ldsk-") {
			candidate = strings.TrimPrefix(raw, "cid:ldsk-")
			explicit = true
		} else if strings.HasPrefix(raw, "cid:") {
			return match
		} else {
			parsed, err := url.Parse(html.UnescapeString(raw))
			if err != nil {
				return match
			}
			if strings.HasPrefix(parsed.Path, "/uploads/") {
				candidate = strings.TrimPrefix(parsed.Path, "/uploads/")
				explicit = true
			} else {
				candidate = stringutil.ExtractUUID(parsed.Path)
			}
		}
		parsed, err := uuid.Parse(candidate)
		if err != nil {
			return match
		}
		candidate = parsed.String()
		if !explicit {
			if _, err := m.mediaStore.GetDraftInlineMedia(candidate, conversationID, userID); err != nil {
				return match
			}
		}
		if !seen[candidate] {
			seen[candidate] = true
			claims = append(claims, candidate)
		}
		return strings.Replace(match, raw, "cid:"+inlineContentID(candidate), 1)
	})
	return rewritten, claims
}
