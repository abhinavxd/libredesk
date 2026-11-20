package stringutil

import (
	"strings"

	"golang.org/x/net/html"
)

// HTMLPreviewText returns the body text of HTML on one line, leaving out headings so a preview starts at the first sentence.
func HTMLPreviewText(htmlContent string) string {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return ""
	}
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
			return
		}
		block := false
		if n.Type == html.ElementNode {
			if nonContentElements[n.Data] || isHeadingElement(n.Data) {
				return
			}
			// code is inline here, a space around it would detach the punctuation that follows.
			block = (blockElements[n.Data] && n.Data != "code") || n.Data == "br"
		}
		// Text from neighboring blocks would otherwise run together as one word.
		if block {
			b.WriteByte(' ')
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
		if block {
			b.WriteByte(' ')
		}
	}
	walk(doc)
	return strings.Join(strings.Fields(b.String()), " ")
}

func isHeadingElement(tag string) bool {
	return len(tag) == 2 && tag[0] == 'h' && tag[1] >= '1' && tag[1] <= '6'
}
