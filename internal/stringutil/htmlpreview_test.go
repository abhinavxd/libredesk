package stringutil

import "testing"

func TestHTMLPreviewText(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{"inline formatting keeps the sentence intact", `<p>Pay <strong>online</strong> &amp; <em>confirm</em>.</p>`, "Pay online & confirm."},
		{"headings are left out", `<h2>Allow cookies</h2><p>Open your browser settings.</p><h3>Next</h3><p>Reload.</p>`, "Open your browser settings. Reload."},
		{"inline code stays attached to punctuation", `<p>Click <code>Save</code>.</p><pre><code>go build</code></pre><p>Done</p>`, "Click Save. go build Done"},
		{"blocks and line breaks are separated", `<p>One</p><ul><li>Two</li><li>Three</li></ul><p>Four<br>Five</p>`, "One Two Three Four Five"},
		{"scripts and styles are dropped", `<p>Before</p><script>alert(1)</script><style>.c{}</style><p>After</p>`, "Before After"},
		{"plain text passes through", `No markup here`, "No markup here"},
		{"only headings", `<h1>Title</h1>`, ""},
		{"empty", ``, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HTMLPreviewText(tt.html); got != tt.want {
				t.Errorf("HTMLPreviewText() = %q, want %q", got, tt.want)
			}
		})
	}
}
