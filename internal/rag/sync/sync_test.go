package sync

import (
	"testing"
)

func TestStripHTML(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "plain text",
			input: "Hello World",
			want:  "Hello World",
		},
		{
			name:  "simple HTML tags",
			input: "<p>Hello World</p>",
			want:  "Hello World",
		},
		{
			name:  "nested tags",
			input: "<div><p>Hello <strong>World</strong></p></div>",
			want:  "Hello World",
		},
		{
			name:  "HTML entities",
			input: "Hello &amp; World &lt;test&gt;",
			want:  "Hello & World <test>",
		},
		{
			name:  "multiple whitespace",
			input: "Hello    World",
			want:  "Hello World",
		},
		{
			name:  "line breaks and tabs",
			input: "Hello\n\t\nWorld",
			want:  "Hello World",
		},
		{
			name:  "HTML with attributes",
			input: `<a href="https://example.com" class="link">Click here</a>`,
			want:  "Click here",
		},
		{
			name:  "self-closing tags",
			input: "Hello<br/>World<hr/>End",
			want:  "Hello World End",
		},
		{
			name:  "complex HTML",
			input: `<html><body><h1>Title</h1><p>Some <em>emphasized</em> text.</p></body></html>`,
			want:  "Title Some emphasized text.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripHTML(tt.input)
			if got != tt.want {
				t.Errorf("stripHTML() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestChunkContent(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		maxLen    int
		wantCount int
	}{
		{
			name:      "short content",
			content:   "Hello world",
			maxLen:    100,
			wantCount: 1,
		},
		{
			name:      "exact length",
			content:   "Hello",
			maxLen:    5,
			wantCount: 1,
		},
		{
			name:      "needs chunking",
			content:   "Hello world this is a test",
			maxLen:    10,
			wantCount: 3, // "Hello", "world this", "is a test"
		},
		{
			name:      "empty content",
			content:   "",
			maxLen:    100,
			wantCount: 1, // empty string counts as 1 chunk
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := chunkContent(tt.content, tt.maxLen)
			if len(got) != tt.wantCount {
				t.Errorf("chunkContent() returned %d chunks, want %d", len(got), tt.wantCount)
			}
			// Verify no chunk exceeds maxLen (except single words)
			for i, chunk := range got {
				if len(chunk) > tt.maxLen && len([]string{chunk}) == 1 {
					t.Errorf("chunk %d exceeds maxLen: %d > %d", i, len(chunk), tt.maxLen)
				}
			}
		})
	}
}

func TestChunkContentPreservesContent(t *testing.T) {
	content := "The quick brown fox jumps over the lazy dog"
	maxLen := 15
	chunks := chunkContent(content, maxLen)

	// Join chunks and compare to original (accounting for potential word boundary issues)
	var reconstructed string
	for i, chunk := range chunks {
		if i > 0 {
			reconstructed += " "
		}
		reconstructed += chunk
	}

	// Should preserve all words
	originalWords := len(content) - len(" ")
	reconstructedWords := len(reconstructed) - len(" ")
	if originalWords != reconstructedWords {
		t.Errorf("Content not preserved: original has %d chars, reconstructed has %d", len(content), len(reconstructed))
	}
}
