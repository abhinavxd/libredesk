package stringutil

import (
	"strings"
	"testing"
)

func TestExtractReadableContent(t *testing.T) {
	tests := []struct {
		name        string
		html        string
		wantTitle   string
		wantContent string // substring that must appear in the returned markdown
		wantEmpty   bool   // true when both return values should be empty
	}{
		{
			name: "article with title and body",
			html: `<!DOCTYPE html><html><head><title>How to write tests</title></head><body>
				<article>
					<h1>How to write tests</h1>
					<p>Writing tests is important because they catch regressions early.
					Every public function deserves at least one test that covers the happy path
					and one that covers an error or edge case. This makes the codebase easier
					to maintain over time and gives reviewers confidence when merging changes.</p>
				</article>
			</body></html>`,
			wantTitle:   "How to write tests",
			wantContent: "tests",
		},
		{
			name:      "empty string returns empty",
			html:      "",
			wantEmpty: true,
		},
		{
			name:      "html with no article content returns empty",
			html:      `<html><head><title>Nav only</title></head><body><nav><a>Home</a></nav></body></html>`,
			wantEmpty: true,
		},
		{
			name: "content below default threshold is still extracted",
			// The function lowers CharThreshold to 100 so short help center pages are not dropped.
			html: `<!DOCTYPE html><html><head><title>Short page</title></head><body>
				<article>
					<p>This is a short but meaningful support article that helps users configure their inbox settings correctly.</p>
				</article>
			</body></html>`,
			wantTitle:   "Short page",
			wantContent: "inbox",
		},
		{
			name: "title is extracted from head tag",
			html: `<!DOCTYPE html><html><head><title>  Trimmed Title  </title></head><body>
				<article>
					<p>LibreDesk is an open-source customer support desk that handles email, live chat,
					and WhatsApp conversations in a single shared inbox. Teams can assign conversations,
					add private notes, and track SLA targets without switching between tools.</p>
				</article>
			</body></html>`,
			wantTitle:   "Trimmed Title",
			wantContent: "inbox",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			title, content := ExtractReadableContent(tc.html)
			if tc.wantEmpty {
				if title != "" || content != "" {
					t.Errorf("expected empty result, got title=%q content=%q", title, content)
				}
				return
			}
			if tc.wantTitle != "" && !strings.Contains(title, tc.wantTitle) {
				t.Errorf("title %q does not contain %q", title, tc.wantTitle)
			}
			if tc.wantContent != "" && !strings.Contains(content, tc.wantContent) {
				t.Errorf("content %q does not contain %q", content, tc.wantContent)
			}
			if content != strings.TrimSpace(content) {
				t.Error("content should have leading/trailing whitespace trimmed")
			}
		})
	}
}
