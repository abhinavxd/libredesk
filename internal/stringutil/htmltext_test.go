package stringutil

import "testing"

func TestHTML2Text(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{
			name: "accented entities and formatting",
			html: `<div dir="ltr">Ol&aacute;, tudo bem? Segue o relat&oacute;rio.<br><br>Abra&ccedil;os,<br><b>Jo&atilde;o</b></div>`,
			want: "Olá, tudo bem? Segue o relatório.\n\nAbraços,\nJoão.",
		},
		{
			name: "links dropped",
			html: `<p>Hello,</p><p>See the <a href="https://example.com/inv/42?a=1&amp;b=2">invoice</a> or mail <a href="mailto:billing@example.com">billing</a>.</p>`,
			want: "Hello,\n\nSee the invoice or mail billing.",
		},
		{
			name: "lists",
			html: `<ul><li>First</li><li>Second</li></ul><ol><li>One</li><li>Two</li></ol>`,
			want: "First\nSecond\n\nOne\nTwo",
		},
		{
			name: "table flattened",
			html: `<table><tr><th>Plan</th><th>Price</th></tr><tr><td>Pro</td><td>&euro;29</td></tr></table>`,
			want: "Plan Price Pro €29",
		},
		{
			name: "entities and nbsp",
			html: `<p>&nbsp;spaced&nbsp;&amp; entities &lt;x&gt; &quot;q&quot;</p>`,
			want: "spaced & entities <x> \"q\"",
		},
		{
			name: "multilingual with hidden preheader",
			html: `<div style="display:none">preheader hidden</div><div>visible 您好 مرحبا שלום Grüße</div>`,
			want: "preheader hidden\nvisible 您好 مرحبا שלום Grüße",
		},
		{
			name: "script and style stripped",
			html: `<p>a<br>b</p><script>alert(1)</script><style>.c{}</style><p>after</p>`,
			want: "a\nb\n\nafter",
		},
		{
			name: "images dropped",
			html: `<img src="cid:logo" alt="Logo"><img src="https://x/y.png">text after images`,
			want: "text after images",
		},
		{
			name: "malformed nesting",
			html: `<div>unclosed <b>bold <i>both</b> italic?</i> end`,
			want: "unclosed bold both. italic? end",
		},
		{
			name: "linked image keeps alt",
			html: `<a href="https://x.io"><img src="cid:1" alt="Banner"></a>`,
			want: "Banner",
		},
		{
			name: "blockquote flattened",
			html: `<blockquote>quoted line</blockquote>after quote`,
			want: "quoted line\n\nafter quote",
		},
		{
			name: "plain text passthrough",
			html: `plain text, no html at all`,
			want: "plain text, no html at all",
		},
		{
			name: "empty input",
			html: ``,
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HTML2Text(tt.html); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
