package ai

import "testing"

func TestStripCodeFence(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "html fence unwrapped",
			in:   "```html\n<p>Hello <a href=\"https://example.com\">link</a></p>\n```",
			want: `<p>Hello <a href="https://example.com">link</a></p>`,
		},
		{
			name: "bare fence unwrapped",
			in:   "```\n<p>Hi</p>\n```",
			want: "<p>Hi</p>",
		},
		{
			name: "fence with surrounding whitespace",
			in:   "  ```html\n<p>Hi</p>\n```  ",
			want: "<p>Hi</p>",
		},
		{
			name: "no fence untouched",
			in:   "<p>Hi</p>",
			want: "<p>Hi</p>",
		},
		{
			name: "fence in the middle untouched",
			in:   "Use ```code``` like this",
			want: "Use ```code``` like this",
		},
		{
			name: "inner fences untouched",
			in:   "```html\n<p>a</p>\n```\ntext\n```html\n<p>b</p>\n```",
			want: "```html\n<p>a</p>\n```\ntext\n```html\n<p>b</p>\n```",
		},
		{
			name: "single line fence untouched",
			in:   "``````",
			want: "``````",
		},
		{
			name: "content on opening fence line untouched",
			in:   "```<p>Hello,</p>\n<p>Thanks.</p>\n```",
			want: "```<p>Hello,</p>\n<p>Thanks.</p>\n```",
		},
		{
			name: "c++ fence unwrapped",
			in:   "```c++\nint main() {}\n```",
			want: "int main() {}",
		},
		{
			name: "text/plain fence unwrapped",
			in:   "```text/plain\nHello there.\n```",
			want: "Hello there.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stripCodeFence(tt.in); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEnsureHTMLFragment(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "html untouched",
			in:   "<p>Hello <a href=\"https://example.com\">link</a></p>",
			want: "<p>Hello <a href=\"https://example.com\">link</a></p>",
		},
		{
			name: "plain text escaped with line breaks",
			in:   "Hello,\nHow are you? 1 < 2",
			want: "Hello,<br>How are you? 1 &lt; 2",
		},
		{
			name: "angle-bracketed email is not html",
			in:   "Contact <customer@example.com>",
			want: "Contact &lt;customer@example.com&gt;",
		},
		{
			name: "self-closing tag untouched",
			in:   "line one<br/>line two",
			want: "line one<br/>line two",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ensureHTMLFragment(tt.in); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
