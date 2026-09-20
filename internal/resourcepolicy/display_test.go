package resourcepolicy

import (
	"slices"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/attachment"
	"golang.org/x/net/html"
)

func TestDisplayBlocksImages(t *testing.T) {
	input := `<p>Hello <strong>world</strong></p><img src="https://tracker.example/pixel?id=secret" alt="Chart"><img src="//tracker.example/another"><img src="/uploads/unknown"><img src="cid:unknown">`
	got := (Policy{}).PrepareDisplay(input, nil)
	if got.BlockedImages != 4 || !slices.Equal(got.BlockedDomains, []string{"tracker.example"}) {
		t.Fatalf("unexpected blocked resources: %#v", got)
	}
	if !strings.Contains(got.HTML, `<strong>world</strong>`) || !strings.Contains(got.HTML, `[Image blocked: Chart]`) {
		t.Fatalf("missing content: %s", got.HTML)
	}
	if strings.Contains(got.HTML, "secret") || strings.Contains(got.HTML, "tracker.example") {
		t.Fatalf("blocked URL leaked into display HTML: %s", got.HTML)
	}
	assertDisplayResources(t, got.HTML, nil)
}

func TestDisplayAllowedImages(t *testing.T) {
	p, err := New(Config{Mode: Allowlist, AllowedDomains: []string{"images.example"}})
	if err != nil {
		t.Fatal(err)
	}
	trusted := "http://desk.example/uploads/known?sig=abc&exp=123"
	input := `<img src="https://images.example/logo" srcset="https://tracker.example/pixel 2x" onerror="alert(1)"><img src="` + trusted + `"><img src="http://desk.example/uploads/unknown"><img src="https://sub.images.example/pixel">`
	got := p.PrepareDisplayWithImages(input, []string{trusted}, func(source string) string {
		if p.AllowsURL(source) {
			return "/api/v1/conversations/c/messages/m/images/hash"
		}
		return ""
	})
	if got.BlockedImages != 2 {
		t.Fatalf("unexpected blocked images: %#v", got)
	}
	assertDisplayResources(t, got.HTML, []string{"/api/v1/conversations/c/messages/m/images/hash", trusted})
	if strings.Count(got.HTML, `referrerpolicy="no-referrer"`) != 2 {
		t.Fatalf("missing image referrer policy: %s", got.HTML)
	}
	blocked := (Policy{}).PrepareDisplay(`<img src="https://images.example/logo"><img src="`+trusted+`">`, []string{trusted})
	if blocked.BlockedImages != 1 {
		t.Fatalf("block_all must still permit verified local attachments: %#v", blocked)
	}
}

func TestDisplayPreservesBasicFormatting(t *testing.T) {
	input := `<table><tr><td colspan="2" style="color:#123456;background-color:white;font-size:14px;font-weight:bold;text-align:center;padding:12px;background-image:url(https://tracker.example/x);position:fixed;display:none;font-family:remote">Hello <em>there</em></td></tr></table><a href="https://example.com" ping="https://tracker.example/ping">Visit</a>`
	got := (Policy{}).PrepareDisplay(input, nil)
	for _, wanted := range []string{`<table>`, `colspan="2"`, "color: #123456", "font-size: 14px", "padding: 12px", "<em>there</em>", `href="https://example.com"`, "noreferrer", "noopener"} {
		if !strings.Contains(got.HTML, wanted) {
			t.Errorf("missing %q in %s", wanted, got.HTML)
		}
	}
	for _, forbidden := range []string{"tracker.example", "position", "display:", "font-family", "background-image", "ping="} {
		if strings.Contains(got.HTML, forbidden) {
			t.Errorf("found %q in %s", forbidden, got.HTML)
		}
	}
}

func TestDisplayHostileHTML(t *testing.T) {
	for _, input := range []string{
		`<script src="https://evil.example/x">alert(1)</script><p onclick="alert(1)">OK</p>`,
		`<style>@import 'https://evil.example/style'; @font-face{font-family:x;src:url(https://evil.example/font)}</style><p style="background:url(https://evil.example/x)">OK</p>`,
		`<link rel="preload" href="https://evil.example/x"><base href="https://evil.example/"><meta http-equiv="refresh" content="0;url=https://evil.example">`,
		`<iframe srcdoc="<img src='https://evil.example/x'>" src="https://evil.example/"></iframe><object data="https://evil.example/x"></object><embed src="https://evil.example/x">`,
		`<form action="https://evil.example/x"><input type="image" src="https://evil.example/x"><button formaction="https://evil.example/y">Submit</button></form>`,
		`<video poster="https://evil.example/x" src="https://evil.example/y"><source src="https://evil.example/z"></video><audio src="https://evil.example/a"></audio>`,
		`<picture><source srcset="https://evil.example/x 1x"><img src="https://evil.example/y" lowsrc="https://evil.example/z"></picture>`,
		`<svg><image href="https://evil.example/x"/><foreignObject><img src="https://evil.example/y"></foreignObject></svg>`,
		`<math><mtext><table><mglyph><style><!--</style><img title="--><img src=https://evil.example/x onerror=alert(1)>">`,
		`<noscript><img src="https://evil.example/x"></noscript><template><img src="https://evil.example/y"></template>`,
		`<img src="https://evil.example/x" alt="&lt;img src=x onerror=alert(1)&gt;"><img src="javascript:alert(1)">`,
		`<img src="https://evil.example/x" src="https://other.example/y"><img/src=//evil.example/z>`,
		`<a href="jav&#x61;script:alert(1)">x</a><a href="data:text/html,x">y</a>`,
		`<p style="color:expression(alert(1));padding:var(--evil);font-size:calc(1px);background-color:u\72l(https://evil.example/x)">x</p>`,
		`<link rel="dns-prefetch" href="//evil.example"><link rel="preconnect" href="https://evil.example"><link rel="prefetch" href="https://evil.example/x">`,
		`<p style="list-style-image:url(https://evil.example/list);cursor:url(https://evil.example/cursor),auto;content:url(https://evil.example/content);border-image:url(https://evil.example/border);filter:url(https://evil.example/filter);mask-image:url(https://evil.example/mask)">Text</p>`,
		`<style>@media (prefers-color-scheme: dark) {p {background-image:url(https://evil.example/dark)}}</style><p style="background-image:image-set(url(https://evil.example/a) 1x,url(https://evil.example/b) 2x)">Text</p>`,
		`<svg><use href="https://evil.example/icons#x"/></svg><video><track src="https://evil.example/subtitles"></video>`,
		`<table background="https://evil.example/x"><tr><td background="https://evil.example/y">x</td></tr></table>`,
	} {
		for _, mode := range []string{BlockAll, Allowlist, LoadOnReceipt} {
			t.Run(mode+"/"+input, func(t *testing.T) {
				policy, err := New(Config{Mode: mode})
				if err != nil {
					t.Fatal(err)
				}
				gateway := "https://desk.example/uploads/approved"
				got := policy.PrepareDisplayWithImages(input, nil, func(string) string { return gateway })
				images := []string{gateway}
				if mode == BlockAll {
					images = nil
				}
				assertDisplayResources(t, got.HTML, images)
				if again := policy.PrepareDisplay(got.HTML, images); again.HTML != got.HTML {
					t.Fatalf("display changes after reparsing:\n%s\n%s", got.HTML, again.HTML)
				}
			})
		}
	}
}

func TestDisplayFallsBackToEscapedText(t *testing.T) {
	for _, input := range []string{
		strings.Repeat("<div>", 600) + `<img src="https://evil.example/x">`,
		strings.Repeat("x", 2*1024*1024) + `<img src="https://evil.example/x">`,
	} {
		got := (Policy{}).PrepareDisplay(input, nil)
		if strings.Contains(got.HTML, "<") || !strings.Contains(got.HTML, "&lt;img") {
			t.Fatal("oversized or deeply nested HTML was not escaped")
		}
	}
}

func assertDisplayResources(t *testing.T, content string, images []string) {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if slices.Contains([]string{"script", "style", "iframe", "object", "embed", "link", "meta", "base", "svg", "math", "form", "input", "video", "audio", "source", "template"}, n.Data) && n.Type == html.ElementNode {
			t.Errorf("unsafe element %s in %s", n.Data, content)
		}
		for _, attr := range n.Attr {
			if attr.Key == "src" && (n.Data != "img" || !slices.Contains(images, attr.Val)) {
				t.Errorf("unexpected resource %q in %s", attr.Val, content)
			}
			if strings.HasPrefix(attr.Key, "on") || slices.Contains([]string{"srcset", "background", "poster", "ping", "srcdoc", "action", "formaction", "data", "xlink:href"}, attr.Key) {
				t.Errorf("unsafe attribute %s in %s", attr.Key, content)
			}
			if attr.Key == "href" && !strings.HasPrefix(attr.Val, "https://") && !strings.HasPrefix(attr.Val, "http://") && !strings.HasPrefix(attr.Val, "mailto:") {
				t.Errorf("unsafe link %q", attr.Val)
			}
			if attr.Key == "style" && strings.ContainsAny(attr.Val, "(\\@") {
				t.Errorf("unsafe style %q", attr.Val)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
}

func TestDisplayDoesNotResolveImagesInsideRemovedContent(t *testing.T) {
	policy, _ := New(Default())
	for _, tag := range []string{"template", "object", "video", "audio", "form", "svg", "math"} {
		t.Run(tag, func(t *testing.T) {
			input := "<" + tag + `><img src="https://evil.example/hidden"></` + tag + ">"
			policy.PrepareDisplayWithImages(input, nil, func(source string) string {
				t.Errorf("removed content reached image resolver: %s", source)
				return ""
			})
		})
	}
}

func TestContentWithAttachments(t *testing.T) {
	const id = "12345678-1234-4234-9234-123456789abc"
	const path = "/uploads/" + id
	const signed = "https://desk.example" + path + "?sig=fresh&exp=123"
	attachments := attachment.Attachments{{UUID: id, ContentID: "ldsk-" + id, URL: signed, ContentType: "image/png"}}
	for _, source := range []string{"cid:ldsk-" + id, path, path + "?sig=expired", "https://desk.example" + path + "?sig=expired", signed} {
		t.Run(source, func(t *testing.T) {
			got := PrepareContentWithAttachments(`<p>Screenshot:</p><img src="`+source+`">`, "html", attachments)
			if got.BlockedImages != 0 || strings.Count(got.HTML, "<img ") != 1 {
				t.Fatalf("local image blocked: %#v", got)
			}
			assertDisplayResources(t, got.HTML, []string{signed})
		})
	}
	for _, source := range []string{"https://tracker.example" + path, "//tracker.example" + path, "/uploads/unattached", "cid:unattached", "data:image/png;base64,abc"} {
		got := PrepareContentWithAttachments(`<img src="`+source+`">`, "html", attachments)
		if got.BlockedImages != 1 || strings.Contains(got.HTML, "<img") {
			t.Fatalf("untrusted image accepted: %#v", got)
		}
	}
	for _, kind := range []string{"image/svg+xml", "text/html"} {
		unsafe := append(attachment.Attachments(nil), attachments...)
		unsafe[0].ContentType = kind
		got := PrepareContentWithAttachments(`<img src="cid:ldsk-`+id+`">`, "html", unsafe)
		if got.BlockedImages != 1 {
			t.Fatalf("active attachment accepted: %#v", got)
		}
	}
	got := PrepareContentWithAttachments(`<img src="`+signed+`">`, "text", attachments)
	if strings.Contains(got.HTML, "<img") || got.BlockedImages != 0 {
		t.Fatalf("plain text interpreted as HTML: %#v", got)
	}
}
