package telegram

import "testing"

func TestFormatHTML(t *testing.T) {
	for _, tc := range []struct{ name, input, html, plain string }{
		{"empty", "", "", ""},
		{"paragraphs", "<p>Hello <strong>there</strong></p><p>Next</p>", "Hello <strong>there</strong>\nNext", "Hello there\nNext"},
		{"all supported tags", "<b>b</b><i>i</i><em>e</em><u>u</u><s>s</s><strike>s</strike><del>d</del><code>c</code><pre>p</pre><blockquote>q</blockquote>", "<b>b</b><i>i</i><em>e</em><u>u</u><s>s</s><strike>s</strike><del>d</del><code>c</code><pre>p</pre><blockquote>q</blockquote>", "bieussdcpq"},
		{"untrusted HTML", "<script>alert(1)</script><style>body{}</style><p onclick='run()'>A &amp; B &lt; C 😀</p>", "A &amp; B &lt; C 😀", "A & B < C 😀"},
		{"safe links", `<a class="x" href="https://example.com/?a=1&amp;b=2" onclick="run()">site</a>`, `<a href="https://example.com/?a=1&amp;b=2">site</a>`, "site"},
		{"unsafe links", `<a href="javascript:run()">click</a><a href="/%zz">bad</a><a title="x">plain</a>`, "clickbadplain", "clickbadplain"},
		{"lists", "<ul><li>One</li> <li>Two</li></ul><ol> <li>First</li> <li>Second</li></ol>", "• One\n \n• Two\n \n1. First\n \n2. Second", "• One\n \n• Two\n \n1. First\n \n2. Second"},
		{"heading and breaks", "<h2>Title</h2><div>Line<br>Next</div>", "<b>Title</b>\nLine\nNext", "Title\nLine\nNext"},
		{"nested list", "<ol><li>First<ul><li>Nested</li></ul></li><li>Second</li></ol>", "1. First\n• Nested\n2. Second", "1. First\n• Nested\n2. Second"},
		{"mail link", `<a href="mailto:a@example.com">Email</a>`, `<a href="mailto:a@example.com">Email</a>`, "Email"},
		{"telegram link", `<a href="tg://user?id=123">User</a>`, `<a href="tg://user?id=123">User</a>`, "User"},
		{"http link", `<a href="http://example.com">Site</a>`, `<a href="http://example.com">Site</a>`, "Site"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			formatted, plain := FormatHTML(tc.input)
			if formatted != tc.html || plain != tc.plain {
				t.Fatalf("formatted=%q plain=%q", formatted, plain)
			}
		})
	}
}
