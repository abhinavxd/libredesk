package telegram

import (
	"html"
	"net/url"
	"strconv"
	"strings"

	nethtml "golang.org/x/net/html"
)

func FormatHTML(content string) (string, string) {
	root, _ := nethtml.Parse(strings.NewReader(content))
	var formatted, plain strings.Builder
	var walk func(*nethtml.Node)
	writeText := func(text string) { formatted.WriteString(html.EscapeString(text)); plain.WriteString(text) }
	walk = func(node *nethtml.Node) {
		if node.Type == nethtml.TextNode {
			writeText(node.Data)
			return
		}
		tag := ""
		block := false
		prefix := ""
		if node.Type == nethtml.ElementNode {
			switch node.Data {
			case "script", "style", "head":
				return
			case "br":
				writeText("\n")
				return
			case "p", "div", "ul", "ol":
				block = true
			case "li":
				block = true
				prefix = "• "
				if node.Parent != nil && node.Parent.Data == "ol" {
					index := 1
					for previous := node.PrevSibling; previous != nil; previous = previous.PrevSibling {
						if previous.Type == nethtml.ElementNode && previous.Data == "li" {
							index++
						}
					}
					prefix = strconv.Itoa(index) + ". "
				}
			case "b", "strong", "i", "em", "u", "s", "strike", "del", "code", "pre", "blockquote":
				tag = node.Data
			case "h1", "h2", "h3", "h4", "h5", "h6":
				tag = "b"
				block = true
			case "a":
				for _, attr := range node.Attr {
					if attr.Key != "href" {
						continue
					}
					target, err := url.Parse(attr.Val)
					if err == nil && (target.Scheme == "https" || target.Scheme == "http" || target.Scheme == "mailto" || target.Scheme == "tg") {
						formatted.WriteString(`<a href="` + html.EscapeString(attr.Val) + `">`)
						tag = "a"
					}
					break
				}
			}
		}
		if block && plain.Len() > 0 && !strings.HasSuffix(plain.String(), "\n") {
			writeText("\n")
		}
		if tag != "" && tag != "a" {
			formatted.WriteString("<" + tag + ">")
		}
		writeText(prefix)
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
		if tag != "" {
			formatted.WriteString("</" + tag + ">")
		}
		if block && plain.Len() > 0 && !strings.HasSuffix(plain.String(), "\n") {
			writeText("\n")
		}
	}
	walk(root)
	return strings.TrimSpace(formatted.String()), strings.TrimSpace(plain.String())
}
