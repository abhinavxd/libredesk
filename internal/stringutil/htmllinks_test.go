package stringutil

import (
	"testing"
)

func TestHTML2TextMarkdownLinks(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{
			name: "simple link with text",
			html: `<p>See the <a href="https://example.com">docs</a> for more.</p>`,
			want: "See the [docs](https://example.com) for more.",
		},
		{
			name: "link text equals href is not wrapped",
			html: `<p><a href="https://example.com">https://example.com</a></p>`,
			want: "https://example.com",
		},
		{
			name: "anchor with no href is left as plain text",
			html: `<p><a>no href here</a></p>`,
			want: "no href here",
		},
		{
			name: "fragment-only href is not wrapped",
			html: `<p><a href="#section">jump</a></p>`,
			want: "jump",
		},
		{
			name: "cid: href is not wrapped",
			html: `<p><a href="cid:image001@example.com">inline image</a></p>`,
			want: "inline image",
		},
		{
			name: "link text with square brackets is escaped",
			html: `<p><a href="https://example.com">[click here]</a></p>`,
			want: `[\[click here\]](https://example.com)`,
		},
		{
			name: "href with spaces is angle-bracket wrapped",
			html: `<p><a href="https://example.com/path with spaces">link</a></p>`,
			want: "[link](<https://example.com/path with spaces>)",
		},
		{
			name: "multiple links in one paragraph",
			html: `<p>Visit <a href="https://a.com">Site A</a> and <a href="https://b.com">Site B</a>.</p>`,
			want: "Visit [Site A](https://a.com) and [Site B](https://b.com).",
		},
		{
			name: "empty html returns empty string",
			html: "",
			want: "",
		},
		{
			name: "plain text with no links is unchanged",
			html: `<p>Hello world</p>`,
			want: "Hello world",
		},
		{
			name: "mailto link is preserved as markdown",
			html: `<p>Email <a href="mailto:support@example.com">support</a>.</p>`,
			want: "Email [support](mailto:support@example.com).",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := HTML2TextMarkdownLinks(tc.html)
			if got != tc.want {
				t.Errorf("HTML2TextMarkdownLinks(%q)\n got  %q\n want %q", tc.html, got, tc.want)
			}
		})
	}
}
