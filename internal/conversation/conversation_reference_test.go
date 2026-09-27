package conversation

import "testing"

func TestAbsolutizeConversationReferenceLinks(t *testing.T) {
	content := `<p><a href="/inboxes/all/conversation/abc" class="ld-conversation-reference">#108</a> <a href="/help">Help</a> <a href="https://old.example.com/inboxes/all/conversation/def">#109</a></p>`
	want := `<p><a href="https://desk.example.com/inboxes/all/conversation/abc" class="ld-conversation-reference">#108</a> <a href="/help">Help</a> <a href="https://old.example.com/inboxes/all/conversation/def">#109</a></p>`

	if got := absolutizeConversationReferenceLinks(content, "https://desk.example.com"); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
