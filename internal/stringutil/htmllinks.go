package stringutil

import (
	"strings"

	"github.com/jaytaylor/html2text"
	"golang.org/x/net/html"
)

// HTML2TextMarkdownLinks converts HTML to text keeping links as markdown "[text](url)", the form a
// model copies back into its reply and that Markdown2HTML turns into an anchor again.
func HTML2TextMarkdownLinks(htmlContent string) string {
	if doc, err := html.Parse(strings.NewReader(htmlContent)); err == nil {
		inlineAnchorMarkdown(doc)
		var b strings.Builder
		if err := html.Render(&b, doc); err == nil {
			htmlContent = b.String()
		}
	}
	return htmlToText(htmlContent, html2text.Options{})
}

func inlineAnchorMarkdown(n *html.Node) {
	var next *html.Node
	for child := n.FirstChild; child != nil; child = next {
		next = child.NextSibling
		if child.Type == html.ElementNode && child.Data == "a" {
			link := markdownLink(nodeText(child), strings.TrimSpace(attrValue(child, "href")))
			if link != "" {
				n.InsertBefore(&html.Node{Type: html.TextNode, Data: link}, child)
				n.RemoveChild(child)
				continue
			}
		}
		inlineAnchorMarkdown(child)
	}
}

// markdownLink returns "" when the anchor is not worth rewriting, leaving it to the text conversion.
func markdownLink(text, href string) string {
	if href == "" || strings.HasPrefix(href, "#") || strings.HasPrefix(href, "cid:") {
		return ""
	}
	text = strings.Join(strings.Fields(text), " ")
	if text == "" || text == href {
		return href
	}
	if strings.ContainsAny(href, " ()") {
		href = "<" + href + ">"
	}
	return "[" + strings.NewReplacer("[", `\[`, "]", `\]`).Replace(text) + "](" + href + ")"
}

func nodeText(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			b.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}
