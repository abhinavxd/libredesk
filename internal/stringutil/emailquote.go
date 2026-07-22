package stringutil

import (
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// Marker sets mirror the frontend's .hide-quoted-text selectors in
// frontend/shared-ui/assets/styles/main.scss - keep the two in sync.
var (
	// Outlook/Hotmail markers: the quoted history follows as trailing siblings, so the marker AND
	// everything after it at the same level is quote.
	quoteMarkerIDs     = map[string]bool{"divRplyFwdMsg": true, "appendonsend": true, "OLK_SRC_BODY_SECTION": true}
	quoteMarkerClasses = []string{"OutlookMessageHeader"}
	// Containers that wrap the whole quote.
	quoteContainerClasses = []string{"yahoo_quoted", "gmail_quote_container", "protonmail_quote"}

	attributionRegex     = regexp.MustCompile(`(?i)^on\b.{0,200}\bwrote:$`)
	originalMessageRegex = regexp.MustCompile(`(?i)^-{2,}\s*original message\s*-{2,}$`)
)

// HTML2TextNoQuotes converts email HTML to plain text with quoted reply chains removed, returning "" when the message was quote-only.
func HTML2TextNoQuotes(htmlContent string) string {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return TrimPlainTextQuotes(HTML2Text(htmlContent))
	}
	pruneQuotedNodes(doc)
	var b strings.Builder
	if err := html.Render(&b, doc); err != nil {
		return TrimPlainTextQuotes(HTML2Text(htmlContent))
	}
	return TrimPlainTextQuotes(HTML2Text(b.String()))
}

// TrimPlainTextQuotes strips a trailing quoted-reply block (">" lines, "On ... wrote:" and "Original Message" markers) from plain text.
func TrimPlainTextQuotes(text string) string {
	lines := strings.Split(text, "\n")
	end := len(lines)
	for end > 0 {
		l := strings.TrimSpace(lines[end-1])
		if l == "" || strings.HasPrefix(l, ">") || attributionRegex.MatchString(l) || originalMessageRegex.MatchString(l) {
			end--
			continue
		}
		break
	}
	return strings.TrimSpace(strings.Join(lines[:end], "\n"))
}

func pruneQuotedNodes(n *html.Node) {
	var next *html.Node
	for child := n.FirstChild; child != nil; child = next {
		next = child.NextSibling
		if isQuoteNode(child) {
			n.RemoveChild(child)
			continue
		}
		if isQuoteMarkerNode(child) {
			for sib := child; sib != nil; {
				after := sib.NextSibling
				n.RemoveChild(sib)
				sib = after
			}
			return
		}
		pruneQuotedNodes(child)
	}
}

func isQuoteNode(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}
	if n.Data == "blockquote" {
		return true
	}
	class := attrValue(n, "class")
	for _, c := range quoteContainerClasses {
		if strings.Contains(class, c) {
			return true
		}
	}
	return false
}

func isQuoteMarkerNode(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}
	if quoteMarkerIDs[attrValue(n, "id")] {
		return true
	}
	class := attrValue(n, "class")
	for _, c := range quoteMarkerClasses {
		if strings.Contains(class, c) {
			return true
		}
	}
	return false
}

func attrValue(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}
