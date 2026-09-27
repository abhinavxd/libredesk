package telegram

import "testing"

func TestIncomingFormatting(t *testing.T) {
	for _, tc := range []struct {
		name, text, want string
		entities         []Entity
	}{
		{"plain escaping", "<a>&", "&lt;a&gt;&amp;", nil},
		{"UTF16 emoji", "😀Hello", "😀<b>Hello</b>", []Entity{{Type: "bold", Offset: 2, Length: 5}}},
		{"nested", "abcdef", "<b>a<i>bc</i>def</b>", []Entity{{Type: "italic", Offset: 1, Length: 2}, {Type: "bold", Length: 6}}},
		{"adjacent", "ab", "<b>a</b><i>b</i>", []Entity{{Type: "bold", Length: 1}, {Type: "italic", Offset: 1, Length: 1}}},
		{"crossing", "abcd", "<b>abc</b>d", []Entity{{Type: "bold", Length: 3}, {Type: "italic", Offset: 2, Length: 2}}},
		{"invalid boundaries", "😀abc", "😀abc", []Entity{{Type: "bold", Offset: 1, Length: 1}, {Type: "bold", Offset: -1, Length: 2}, {Type: "bold", Length: 0}, {Type: "bold", Length: 99}}},
		{"unknown", "ab", "ab", []Entity{{Type: "unknown", Length: 2}}},
		{"safe link", "link", `<a href="https://example.com/?a=1&amp;b=2">link</a>`, []Entity{{Type: "text_link", Length: 4, URL: "https://example.com/?a=1&b=2"}}},
		{"unsafe link", "link", "link", []Entity{{Type: "text_link", Length: 4, URL: "javascript:alert(1)"}}},
		{"invalid link", "link", "link", []Entity{{Type: "text_link", Length: 4, URL: "%"}}},
		{"url", "https://example.com", `<a href="https://example.com">https://example.com</a>`, []Entity{{Type: "url", Length: 19}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := FormatEntities(tc.text, tc.entities); got != tc.want {
				t.Fatalf("got=%q want=%q", got, tc.want)
			}
		})
	}
	for kind, tag := range map[string]string{"underline": "u", "strikethrough": "s", "code": "code", "pre": "pre", "blockquote": "blockquote", "expandable_blockquote": "blockquote"} {
		if got := FormatEntities("x", []Entity{{Type: kind, Length: 1}}); got != "<"+tag+">x</"+tag+">" {
			t.Errorf("%s: %s", kind, got)
		}
	}
	for _, tc := range []struct {
		message       Message
		content, kind string
	}{
		{Message{Text: "Hello"}, "Hello", "text"},
		{Message{Text: "Hello", Entities: []Entity{{Type: "bold", Length: 5}}}, "<b>Hello</b>", "html"},
		{Message{Caption: "Photo", CaptionEntities: []Entity{{Type: "italic", Length: 5}}}, "<i>Photo</i>", "html"},
	} {
		if content, kind := tc.message.FormattedContent(); content != tc.content || kind != tc.kind {
			t.Errorf("%q %q", content, kind)
		}
	}
}
