package conversation

import (
	"strings"
	"testing"
)

func TestAbsolutizeConversationReferenceLinks(t *testing.T) {
	content := `<p><a href="/inboxes/all/conversation/abc" class="ld-conversation-reference">#108</a> <a href="/help">Help</a> <a href="https://old.example.com/inboxes/all/conversation/def">#109</a></p>`
	want := `<p><a href="https://desk.example.com/inboxes/all/conversation/abc" class="ld-conversation-reference">#108</a> <a href="/help">Help</a> <a href="https://old.example.com/inboxes/all/conversation/def">#109</a></p>`

	if got := absolutizeConversationReferenceLinks(content, "https://desk.example.com"); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestSanitizeNotificationHTML(t *testing.T) {
	content := `<p><strong>Allowed</strong><img src="x" onerror="alert(1)"><script>alert(2)</script><a href="/inboxes/all/conversation/abc">#108</a></p>`
	got := string(sanitizeNotificationHTML(content, "https://desk.example.com"))

	for _, unsafe := range []string{"onerror", "<script", "alert(1)", "alert(2)"} {
		if strings.Contains(got, unsafe) {
			t.Fatalf("unsafe content %q remained in %q", unsafe, got)
		}
	}
	for _, safe := range []string{"<strong>Allowed</strong>", `href="https://desk.example.com/inboxes/all/conversation/abc"`} {
		if !strings.Contains(got, safe) {
			t.Fatalf("safe content %q missing from %q", safe, got)
		}
	}
}
