package stringutil

import "testing"

func TestHTML2WhatsApp(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{"plain paragraph", `<p>Hello there</p>`, "Hello there"},
		{"bold", `<p>Hello <strong>there</strong></p>`, "Hello *there*"},
		{"bold b tag", `<p>Hello <b>there</b></p>`, "Hello *there*"},
		{"italic", `<p>Hello <em>there</em> and <i>you</i></p>`, "Hello _there_ and _you_"},
		{"strikethrough", `<p><s>old</s> <del>gone</del> <strike>past</strike></p>`, "~old~ ~gone~ ~past~"},
		{"inline code", `<p>Run <code>make build</code> now</p>`, "Run `make build` now"},
		{"nested marks", `<p><strong><em>both</em></strong></p>`, "*_both_*"},
		{"underline dropped", `<p><u>plain</u></p>`, "plain"},
		{"markers hug the text", `<p>a<strong> spaced </strong>b</p>`, "a *spaced* b"},
		{"empty mark emits nothing", `<p>a<strong> </strong>b</p>`, "a b"},
		{"mark split across line break", `<p><strong>one<br>two</strong></p>`, "*one*\n*two*"},
		{"paragraphs separated by blank line", `<p>First</p><p>Second</p>`, "First\n\nSecond"},
		{"empty paragraphs collapse", `<p>First</p><p></p><p></p><p>Second</p>`, "First\n\nSecond"},
		{"line break", `<p>one<br>two</p>`, "one\ntwo"},
		{"bulleted list", `<ul><li><p>First</p></li><li><p>Second</p></li></ul>`, "- First\n- Second"},
		{"numbered list", `<ol><li><p>One</p></li><li><p>Two</p></li></ol>`, "1. One\n2. Two"},
		{"list between paragraphs", `<p>Steps:</p><ol><li><p>One</p></li><li><p>Two</p></li></ol><p>Done.</p>`, "Steps:\n\n1. One\n2. Two\n\nDone."},
		{"list item with marks", `<ul><li><p><strong>Bold</strong> item</p></li></ul>`, "- *Bold* item"},
		{"nested list indented", `<ul><li><p>Top</p><ul><li><p>Child</p></li></ul></li></ul>`, "- Top\n  - Child"},
		{"quote", `<blockquote><p>quoted line</p></blockquote>`, "> quoted line"},
		{"quote multiple lines", `<blockquote><p>a</p><p>b</p></blockquote>`, "> a\n>\n> b"},
		{"quote with list", `<blockquote><ul><li><p>a</p></li><li><p>b</p></li></ul></blockquote>`, "> - a\n> - b"},
		{"ordered list start attribute", `<ol start="3"><li><p>c</p></li><li><p>d</p></li></ol>`, "3. c\n4. d"},
		{"list item with two paragraphs", `<ul><li><p>a</p><p>b</p></li></ul>`, "- a\n  b"},
		{"nested ordered under numbered", `<ol><li><p>Top</p><ol><li><p>Child</p></li></ol></li></ol>`, "1. Top\n   1. Child"},
		{"code block keeps indentation", "<pre><code>func x() {\n    return\n}</code></pre>", "```\nfunc x() {\n    return\n}\n```"},
		{"code block between paragraphs", "<p>Run:</p><pre><code>make</code></pre><p>Then.</p>", "Run:\n\n```\nmake\n```\n\nThen."},
		{"already marked text is not doubled", `<p><strong><b>x</b></strong></p>`, "*x*"},
		{"nbsp treated as space", `<p>a&nbsp;<strong>&nbsp;b&nbsp;</strong>c</p>`, "a *b* c"},
		{"mailto link with address as text", `<p><a href="mailto:a@example.com">a@example.com</a></p>`, "a@example.com"},
		{"link with empty text uses url", `<p><a href="https://example.com"></a></p>`, "https://example.com"},
		{"bold inside link", `<p><a href="https://example.com/x"><strong>Docs</strong></a></p>`, "*Docs* (https://example.com/x)"},
		{"leading and trailing whitespace trimmed", "\n  <p> hello </p>\n  ", "hello"},
		{"div blocks", `<div>a</div><div>b</div>`, "a\n\nb"},
		{"table with thead tbody", `<table><thead><tr><th>A</th></tr></thead><tbody><tr><td>1</td></tr></tbody></table>`, "A\n1"},
		{"collapsible details", `<details><summary>More</summary><div>hidden text</div></details>`, "More\n\nhidden text"},
		{"windows newlines in text", "<p>a\r\nb</p>", "a b"},
		{"tiptap trailing break", `<p>hello<br class="ProseMirror-trailingBreak"></p><p>world</p>`, "hello\n\nworld"},
		{"tiptap hard break at paragraph end", `<p>hello<br></p>`, "hello"},
		{"tiptap callout", `<div data-type="callout" class="callout"><p>Heads up</p></div><p>after</p>`, "Heads up\n\nafter"},
		{"tiptap youtube iframe dropped", `<div data-youtube-video><iframe src="https://youtube.com/embed/x"></iframe></div><p>watch</p>`, "watch"},
		{"tiptap styled link class kept out", `<p><a class="btn" href="https://example.com/go" target="_blank" rel="noopener">Go</a></p>`, "Go (https://example.com/go)"},
		{"text without paragraph", `just text <strong>here</strong>`, "just text *here*"},
		{"literal asterisks untouched", `<p>5 * 3 = 15</p>`, "5 * 3 = 15"},
		{"marker around punctuation", `<p><strong>Note:</strong> read this.</p>`, "*Note:* read this."},
		{"bold wrapping a paragraph", `<strong><p>x</p></strong>`, "*x*"},
		{"empty list", `<ul></ul><p>a</p>`, "a"},
		{"list item with only nested list", `<ul><li><ul><li><p>deep</p></li></ul></li></ul>`, "- - deep"},
		{"quote inside list item", `<ul><li><p>a</p><blockquote><p>q</p></blockquote></li></ul>`, "- a\n  > q"},
		{"link in list item", `<ul><li><p><a href="https://example.com/a">A</a></p></li></ul>`, "- A (https://example.com/a)"},
		{"code block with blank line inside", "<pre><code>a\n\nb</code></pre>", "```\na\n\nb\n```"},
		{"code block with html entities", "<pre><code>if a &lt; b &amp;&amp; c</code></pre>", "```\nif a < b && c\n```"},
		{"inline code with spaces inside", "<p>use <code> ls -la </code> now</p>", "use `ls -la` now"},
		{"strike inside bold inside italic", `<p><em><strong><s>x</s></strong></em></p>`, "_*~x~*_"},
		{"consecutive bold words", `<p><strong>a</strong> <strong>b</strong></p>`, "*a* *b*"},
		{"long whitespace between blocks", "<p>a</p>\n\n\n   \n<p>b</p>", "a\n\nb"},
		{"heading levels", `<h1>One</h1><h3>Three</h3>`, "*One*\n\n*Three*"},
		{"table cell with line break", `<table><tr><td>a<br>b</td><td>c</td></tr></table>`, "a b | c"},
		{"table with empty row skipped", `<table><tr><td></td><td></td></tr><tr><td>x</td></tr></table>`, "x"},
		{"anchor link dropped", `<p><a href="#top">Top</a></p>`, "Top"},
		{"tel link with number as text", `<p><a href="tel:+911234">+911234</a></p>`, "+911234"},
		{"cid inline image inside paragraph", `<p>Before</p><p><img src="cid:ldsk-abc"></p><p>After</p>`, "Before\n\nAfter"},
		{"unclosed tags", `<p>a<strong>b<p>c`, "a*b*\n\n*c*"},
		{"only whitespace", "  \n\t ", ""},
		{"only empty paragraphs", `<p></p><p></p>`, ""},
		{"code block", "<pre><code class=\"language-go\">x := 1\ny := 2</code></pre>", "```\nx := 1\ny := 2\n```"},
		{"code block keeps literal markers", "<pre><code>*not bold*</code></pre>", "```\n*not bold*\n```"},
		{"heading is bold", `<h2>Title</h2><p>Body</p>`, "*Title*\n\nBody"},
		{"link with different text", `<p>See <a href="https://example.com/x">the docs</a></p>`, "See the docs (https://example.com/x)"},
		{"link with url as text", `<p><a href="https://example.com">https://example.com</a></p>`, "https://example.com"},
		{"image dropped", `<p>Look <img src="cid:ldsk-1" alt="pic"> here</p>`, "Look here"},
		{"entities decoded", `<p>Tom &amp; Jerry &lt;3 &eacute;</p>`, "Tom & Jerry <3 é"},
		{"whitespace collapsed", "<p>a\n   b\t c</p>", "a b c"},
		{"table rows", `<table><tr><th>Plan</th><th>Price</th></tr><tr><td>Pro</td><td>29</td></tr></table>`, "Plan | Price\nPro | 29"},
		{"horizontal rule", `<p>a</p><hr><p>b</p>`, "a\n\n---\n\nb"},
		{"mention span is text", `<p>Hi <span data-type="mention" data-id="7">@Ana</span></p>`, "Hi @Ana"},
		{"style and script dropped", `<style>p{}</style><script>x()</script><p>ok</p>`, "ok"},
		{"empty", ``, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HTML2WhatsApp(tt.html); got != tt.want {
				t.Errorf("\nhtml: %s\ngot:  %q\nwant: %q", tt.html, got, tt.want)
			}
		})
	}
}
