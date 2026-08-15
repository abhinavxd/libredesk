package stringutil

import (
	"strings"

	"golang.org/x/net/html"
)

// prepareHTMLForEmbedding rewrites structural markup (tables, link hrefs) into text that survives flattening.
func prepareHTMLForEmbedding(htmlContent string) string {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return htmlContent
	}
	inlineAnchorMarkdown(doc)
	flattenTables(doc)

	var b strings.Builder
	if err := html.Render(&b, doc); err != nil {
		return htmlContent
	}
	return b.String()
}

// flattenTables turns each table into a <pre> of one header-labelled line per row, which also keeps it unsplittable.
func flattenTables(n *html.Node) {
	var tables []*html.Node
	var collect func(*html.Node)
	collect = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "table" {
			tables = append(tables, node)
			return
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			collect(c)
		}
	}
	collect(n)

	for _, table := range tables {
		text := tableToText(table)
		if text == "" {
			continue
		}
		pre := &html.Node{Type: html.ElementNode, Data: "pre"}
		pre.AppendChild(&html.Node{Type: html.TextNode, Data: text})
		table.Parent.InsertBefore(pre, table)
		table.Parent.RemoveChild(table)
	}
}

func tableToText(table *html.Node) string {
	rows := collectRows(table)
	if len(rows) == 0 {
		return ""
	}

	var headers []string
	start := 0
	if isHeaderRow(rows[0]) {
		headers = rowCells(rows[0])
		start = 1
	}

	var lines []string
	for _, row := range rows[start:] {
		cells := rowCells(row)
		if len(cells) == 0 {
			continue
		}
		parts := make([]string, 0, len(cells))
		for i, cell := range cells {
			if i < len(headers) && headers[i] != "" {
				parts = append(parts, headers[i]+": "+cell)
				continue
			}
			parts = append(parts, cell)
		}
		lines = append(lines, strings.Join(parts, " | "))
	}
	if len(headers) > 0 && len(lines) == 0 {
		lines = append(lines, strings.Join(headers, " | "))
	}
	if caption := strings.Join(strings.Fields(captionText(table)), " "); caption != "" {
		lines = append([]string{caption}, lines...)
	}
	return strings.Join(lines, "\n")
}

func collectRows(table *html.Node) []*html.Node {
	var rows []*html.Node
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "tr" {
			rows = append(rows, node)
			return
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(table)
	return rows
}

func rowCells(row *html.Node) []string {
	var cells []string
	for c := row.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode || (c.Data != "td" && c.Data != "th") {
			continue
		}
		cells = append(cells, strings.Join(strings.Fields(nodeText(c)), " "))
	}
	return cells
}

func captionText(table *html.Node) string {
	for c := table.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "caption" {
			return nodeText(c)
		}
	}
	return ""
}

func isHeaderRow(row *html.Node) bool {
	seen := false
	for c := row.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode {
			continue
		}
		switch c.Data {
		case "th":
			seen = true
		case "td":
			return false
		}
	}
	return seen
}
