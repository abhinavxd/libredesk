package conversation

import "strings"

func absolutizeConversationReferenceLinks(content, rootURL string) string {
	if rootURL == "" {
		return content
	}
	return strings.ReplaceAll(
		content,
		`href="/inboxes/all/conversation/`,
		`href="`+strings.TrimRight(rootURL, "/")+`/inboxes/all/conversation/`,
	)
}
