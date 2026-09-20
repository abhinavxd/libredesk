package resourcepolicy

import (
	"bytes"
	"html"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/abhinavxd/libredesk/internal/attachment"
	"github.com/microcosm-cc/bluemonday"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type Display struct {
	HTML           string   `json:"html"`
	CanAllow       bool     `json:"can_allow"`
	Sender         string   `json:"sender,omitempty"`
	SenderTrusted  bool     `json:"sender_trusted"`
	BlockedImages  int      `json:"blocked_images"`
	BlockedDomains []string `json:"blocked_domains"`
}

var displaySanitizer = newDisplaySanitizer()

func (p Policy) PrepareDisplay(content string, trustedImageURLs []string) Display {
	return p.PrepareDisplayWithImages(content, trustedImageURLs, nil)
}

func (p Policy) PrepareDisplayWithImages(content string, trustedImageURLs []string, resolve func(string) string) Display {
	return p.prepareDisplay(content, trustedImageURLs, resolve, nil)
}

func (p Policy) prepareDisplay(content string, trustedImageURLs []string, resolve func(string) string, aliases map[string]string) Display {
	out := Display{BlockedDomains: []string{}}
	if len(content) > 2*1024*1024 {
		out.HTML = html.EscapeString(content)
		return out
	}
	context := &xhtml.Node{Type: xhtml.ElementNode, Data: "div", DataAtom: atom.Div}
	nodes, err := xhtml.ParseFragment(strings.NewReader(content), context)
	if err != nil {
		out.HTML = html.EscapeString(content)
		return out
	}
	trusted := make(map[string]bool, len(trustedImageURLs))
	for _, raw := range trustedImageURLs {
		u, err := url.Parse(raw)
		if err == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Host != "" && u.User == nil && !strings.ContainsAny(raw, "\\\r\n\t") {
			trusted[raw] = true
		}
	}
	var walk func(*xhtml.Node)
	walk = func(n *xhtml.Node) {
		if n.Type == xhtml.ElementNode && (n.Namespace != "" || slices.Contains([]string{"script", "style", "iframe", "object", "embed", "link", "meta", "base", "form", "input", "button", "video", "audio", "source", "track", "template", "noscript"}, n.Data)) {
			n.Type, n.Data, n.Attr = xhtml.TextNode, "", nil
			for n.FirstChild != nil {
				n.RemoveChild(n.FirstChild)
			}
			return
		}
		if n.Type == xhtml.ElementNode && n.Data == "img" {
			var src, alt string
			for _, attr := range n.Attr {
				if attr.Namespace == "" && attr.Key == "src" && src == "" {
					src = attr.Val
				}
				if attr.Namespace == "" && attr.Key == "alt" {
					alt = attr.Val
				}
			}
			resolved := ""
			if alias := aliases[imageReference(src)]; trusted[alias] {
				src = alias
			}
			if trusted[src] {
				resolved = src
			} else if p.PermitsFetching() && ValidImageURL(src) && resolve != nil {
				resolved = resolve(src)
			}
			if resolved != "" {
				attrs := []xhtml.Attribute{{Key: "src", Val: resolved}, {Key: "alt", Val: alt}, {Key: "referrerpolicy", Val: "no-referrer"}}
				for _, attr := range n.Attr {
					if attr.Namespace == "" && (attr.Key == "width" || attr.Key == "height" || attr.Key == "style") {
						attrs = append(attrs, attr)
					}
				}
				n.Attr = attrs
			} else {
				out.BlockedImages++
				if u, err := url.Parse(src); err == nil {
					host := strings.ToLower(u.Hostname())
					if validHost(host) {
						out.BlockedDomains = append(out.BlockedDomains, host)
					}
				}
				n.Data, n.DataAtom, n.Attr = "span", atom.Span, nil
				label := "[Image blocked]"
				if alt != "" {
					label = "[Image blocked: " + alt + "]"
				}
				n.AppendChild(&xhtml.Node{Type: xhtml.TextNode, Data: label})
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	var buf bytes.Buffer
	for _, node := range nodes {
		walk(node)
		if err := xhtml.Render(&buf, node); err != nil {
			out.HTML = html.EscapeString(content)
			return out
		}
	}
	out.HTML = displaySanitizer.Sanitize(buf.String())
	slices.Sort(out.BlockedDomains)
	out.BlockedDomains = slices.Compact(out.BlockedDomains)
	return out
}

func newDisplaySanitizer() *bluemonday.Policy {
	p := bluemonday.NewPolicy()
	elements := []string{"a", "abbr", "b", "blockquote", "br", "caption", "code", "dd", "del", "div", "dl", "dt", "em", "h1", "h2", "h3", "h4", "h5", "h6", "hr", "i", "img", "li", "ol", "p", "pre", "s", "small", "span", "strong", "sub", "sup", "table", "tbody", "td", "th", "thead", "tfoot", "tr", "u", "ul"}
	p.AllowElements(elements...)
	p.AllowAttrs("href").OnElements("a")
	p.AllowAttrs("title").OnElements("a", "abbr")
	p.AllowURLSchemes("http", "https", "mailto")
	p.RequireParseableURLs(true)
	p.AllowRelativeURLs(true)
	p.RequireNoReferrerOnLinks(true)
	p.AddTargetBlankToFullyQualifiedLinks(true)
	p.AllowAttrs("src", "alt").OnElements("img")
	p.AllowAttrs("referrerpolicy").Matching(regexp.MustCompile(`^no-referrer$`)).OnElements("img")
	p.AllowAttrs("width", "height").Matching(regexp.MustCompile(`^[0-9]{1,4}$`)).OnElements("img")
	p.AllowAttrs("colspan", "rowspan").Matching(regexp.MustCompile(`^[1-9][0-9]{0,2}$`)).OnElements("td", "th")
	p.AllowAttrs("dir").Matching(regexp.MustCompile(`^(ltr|rtl|auto)$`)).OnElements(elements...)
	p.AllowStyles("color", "background-color").Matching(regexp.MustCompile(`(?i)^(#[0-9a-f]{3}|#[0-9a-f]{6}|black|white|red|green|blue|gray|grey|silver|maroon|purple|fuchsia|lime|olive|yellow|navy|teal|aqua|transparent)$`)).OnElements(elements...)
	p.AllowStyles("font-size").Matching(regexp.MustCompile(`^([1-9]|[1-6][0-9]|7[0-2])(px|pt)$`)).OnElements(elements...)
	p.AllowStyles("font-weight").MatchingEnum("normal", "bold", "400", "500", "600", "700").OnElements(elements...)
	p.AllowStyles("font-style").MatchingEnum("normal", "italic").OnElements(elements...)
	p.AllowStyles("text-align").MatchingEnum("left", "right", "center", "justify").OnElements(elements...)
	p.AllowStyles("text-decoration").MatchingEnum("none", "underline", "line-through").OnElements(elements...)
	p.AllowStyles("margin", "padding", "margin-top", "margin-bottom", "margin-left", "margin-right", "padding-top", "padding-bottom", "padding-left", "padding-right").Matching(regexp.MustCompile(`^(0|[1-9][0-9]?(px|pt))$`)).OnElements(elements...)
	return p
}

func PrepareContent(content, contentType string) Display {
	if contentType == "text" {
		return Display{HTML: html.EscapeString(content), BlockedDomains: []string{}}
	}
	return (Policy{}).PrepareDisplay(content, nil)
}

// PrepareContentWithAttachments permits only images backed by the supplied,
// authorized attachments. Callers must generate their URLs before calling it.
// It resolves CIDs and refreshes old upload URLs without changing stored content.
func PrepareContentWithAttachments(content, contentType string, attachments attachment.Attachments) Display {
	if contentType == "text" {
		return PrepareContent(content, contentType)
	}
	trusted := make([]string, 0, len(attachments))
	aliases := make(map[string]string)
	for _, att := range attachments {
		if !IsDisplayImage(att.ContentType) {
			continue
		}
		u, err := url.Parse(att.URL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil {
			continue
		}
		// Attachment URLs are generated by the media manager, never taken from HTML.
		if u.Path != "/uploads/"+att.UUID || att.UUID == "" {
			continue
		}
		trusted = append(trusted, att.URL)
		aliases[imageReference(att.URL)] = att.URL
		aliases[u.Path] = att.URL
		if att.ContentID != "" {
			aliases["cid:"+att.ContentID] = att.URL
		}
	}
	return (Policy{}).prepareDisplay(content, trusted, nil, aliases)
}

func imageReference(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	// CIDs are opaque identifiers; do not normalize their contents.
	if u.Scheme == "cid" {
		return raw
	}
	u.RawQuery, u.Fragment, u.RawFragment = "", "", ""
	u.ForceQuery = false
	return u.String()
}

// IsDisplayImage reports whether a media type can be displayed as a passive image.
func IsDisplayImage(contentType string) bool {
	switch contentType {
	case "image/png", "image/jpeg", "image/gif", "image/webp", "image/avif":
		return true
	}
	return false
}
