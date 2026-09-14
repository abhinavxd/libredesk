package conversation

import (
	htmltemplate "html/template"
	"strings"

	"github.com/microcosm-cc/bluemonday"
)

var notificationHTMLPolicy = bluemonday.UGCPolicy()

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

func sanitizeNotificationHTML(content, rootURL string) htmltemplate.HTML {
	content = notificationHTMLPolicy.Sanitize(content)
	return htmltemplate.HTML(absolutizeConversationReferenceLinks(content, rootURL))
}
