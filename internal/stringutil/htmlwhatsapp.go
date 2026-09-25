package stringutil

import (
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

const (
	waBold       = "*"
	waItalic     = "_"
	waStrike     = "~"
	waCode       = "`"
	waCodeFence  = "```"
	waQuote      = "> "
	waBullet     = "- "
	waRule       = "---"
	waCellJoiner = " | "
)

var (
	waInlineSpaces = regexp.MustCompile(`[\s\x{00a0}]+`)
	waBlankLines   = regexp.MustCompile(`\n{3,}`)
)

type waRenderer struct {
	inPre bool
}

func (r *waRenderer) render(n *html.Node) string {
	switch n.Type {
	case html.TextNode:
		if r.inPre {
			return strings.ReplaceAll(n.Data, "\r", "")
		}
		return waInlineSpaces.ReplaceAllString(n.Data, " ")
	case html.DocumentNode:
		return r.children(n)
	case html.ElementNode:
		return r.element(n)
	}
	return ""
}

func (r *waRenderer) element(n *html.Node) string {
	switch n.DataAtom {
	case atom.Script, atom.Style, atom.Head, atom.Title, atom.Img, atom.Iframe, atom.Video, atom.Audio:
		return ""
	case atom.Br:
		return "\n"
	case atom.Hr:
		return waBlock(waRule)
	case atom.Strong, atom.B:
		return waWrap(r.children(n), waBold)
	case atom.Em, atom.I:
		return waWrap(r.children(n), waItalic)
	case atom.S, atom.Del, atom.Strike:
		return waWrap(r.children(n), waStrike)
	case atom.Code:
		if r.inPre {
			return r.children(n)
		}
		return waWrap(r.children(n), waCode)
	case atom.Pre:
		r.inPre = true
		inner := r.children(n)
		r.inPre = false
		return waBlock(waCodeFence + "\n" + strings.Trim(inner, "\n") + "\n" + waCodeFence)
	case atom.A:
		return r.link(n)
	case atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
		return waBlock(waWrap(r.children(n), waBold))
	case atom.Blockquote:
		return waBlock(waPrefixLines(waNormalize(r.children(n)), waQuote))
	case atom.Ul, atom.Ol:
		return waBlock(r.list(n))
	case atom.Table:
		return waBlock(r.table(n))
	case atom.P, atom.Div, atom.Section, atom.Article, atom.Header, atom.Footer, atom.Aside, atom.Main, atom.Nav,
		atom.Figure, atom.Figcaption, atom.Details, atom.Summary, atom.Li, atom.Tr, atom.Dd, atom.Dt, atom.Dl:
		return waBlock(r.children(n))
	}
	return r.children(n)
}

// children joins child output, dropping a chunk's leading spaces after a space or newline so tags never double them.
func (r *waRenderer) children(n *html.Node) string {
	var b strings.Builder
	var last byte
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		chunk := r.render(c)
		if !r.inPre && (last == ' ' || last == '\n') {
			chunk = strings.TrimLeft(chunk, " ")
		}
		if chunk == "" {
			continue
		}
		b.WriteString(chunk)
		last = chunk[len(chunk)-1]
	}
	return b.String()
}

func (r *waRenderer) link(n *html.Node) string {
	text := strings.TrimSpace(waNormalize(r.children(n)))
	href := strings.TrimSpace(attrValue(n, "href"))
	switch {
	case href == "" || strings.HasPrefix(href, "#") || strings.HasPrefix(href, "javascript:"):
		return text
	case text == "":
		return strings.TrimPrefix(href, "mailto:")
	case text == href || text == strings.TrimPrefix(href, "mailto:") || text == strings.TrimPrefix(href, "tel:"):
		return text
	}
	return text + " (" + href + ")"
}

func (r *waRenderer) list(n *html.Node) string {
	ordered := n.DataAtom == atom.Ol
	index := 1
	if start, err := strconv.Atoi(attrValue(n, "start")); err == nil && ordered {
		index = start
	}
	var items []string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode || c.DataAtom != atom.Li {
			continue
		}
		marker := waBullet
		if ordered {
			marker = strconv.Itoa(index) + ". "
		}
		index++
		inner := strings.ReplaceAll(waNormalize(r.children(c)), "\n\n", "\n")
		lines := strings.Split(inner, "\n")
		indent := strings.Repeat(" ", len(marker))
		for i, line := range lines {
			if i == 0 {
				lines[i] = marker + line
			} else {
				lines[i] = indent + line
			}
		}
		items = append(items, strings.Join(lines, "\n"))
	}
	return strings.Join(items, "\n")
}

func (r *waRenderer) table(n *html.Node) string {
	var rows []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type != html.ElementNode {
				continue
			}
			switch c.DataAtom {
			case atom.Tr:
				if row := r.tableRow(c); row != "" {
					rows = append(rows, row)
				}
			case atom.Thead, atom.Tbody, atom.Tfoot:
				walk(c)
			}
		}
	}
	walk(n)
	return strings.Join(rows, "\n")
}

func (r *waRenderer) tableRow(tr *html.Node) string {
	var cells []string
	for c := tr.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode || (c.DataAtom != atom.Td && c.DataAtom != atom.Th) {
			continue
		}
		cell := strings.ReplaceAll(waNormalize(r.children(c)), "\n", " ")
		cells = append(cells, cell)
	}
	row := strings.Join(cells, waCellJoiner)
	if strings.TrimSpace(strings.ReplaceAll(row, "|", "")) == "" {
		return ""
	}
	return row
}

// HTML2WhatsApp converts HTML to WhatsApp's text formatting (*bold*, _italic_, ~strike~, `code`, lists, quotes).
func HTML2WhatsApp(htmlContent string) string {
	if strings.TrimSpace(htmlContent) == "" {
		return ""
	}
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return HTML2Text(htmlContent)
	}
	r := &waRenderer{}
	return waNormalize(r.render(doc))
}

func waBlock(s string) string {
	return "\n\n" + strings.Trim(s, " \t") + "\n\n"
}

// waWrap puts a formatting marker around each non-blank line so markers hug the text, which WhatsApp requires.
func waWrap(inner, marker string) string {
	lines := strings.Split(inner, "\n")
	for i, line := range lines {
		core := strings.TrimSpace(line)
		if core == "" {
			continue
		}
		if len(core) > 2*len(marker) && strings.HasPrefix(core, marker) && strings.HasSuffix(core, marker) {
			continue
		}
		lead := line[:strings.Index(line, core)]
		trail := line[len(lead)+len(core):]
		lines[i] = lead + marker + core + marker + trail
	}
	return strings.Join(lines, "\n")
}

func waPrefixLines(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if line == "" {
			lines[i] = strings.TrimRight(prefix, " ")
			continue
		}
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}

// waNormalize strips trailing whitespace per line and collapses runs of blank lines. Leading indentation is kept.
func waNormalize(s string) string {
	lines := strings.Split(strings.ReplaceAll(s, "\r", ""), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return strings.TrimSpace(waBlankLines.ReplaceAllString(strings.Join(lines, "\n"), "\n\n"))
}
