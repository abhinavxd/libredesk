package telegram

import (
	"cmp"
	"html"
	"net/url"
	"slices"
	"strings"
	"unicode/utf16"
)

type Entity struct {
	Type   string `json:"type"`
	Offset int    `json:"offset"`
	Length int    `json:"length"`
	URL    string `json:"url"`
}

type textEntity struct {
	start, end  int
	open, close string
}

func (m Message) FormattedContent() (string, string) {
	text, entities := m.Text, m.Entities
	if text == "" {
		text, entities = m.Caption, m.CaptionEntities
	}
	if len(entities) == 0 {
		return m.Content(), "text"
	}
	return FormatEntities(text, entities), "html"
}

func FormatEntities(text string, entities []Entity) string {
	positions := map[int]int{0: 0}
	offset := 0
	for index, r := range text {
		positions[offset] = index
		offset += utf16.RuneLen(r)
	}
	positions[offset] = len(text)
	spans := make([]textEntity, 0, len(entities))
	for _, e := range entities {
		start, validStart := positions[e.Offset]
		end, validEnd := positions[e.Offset+e.Length]
		if !validStart || !validEnd || e.Length <= 0 {
			continue
		}
		open, close := "", ""
		switch e.Type {
		case "bold":
			open, close = "<b>", "</b>"
		case "italic":
			open, close = "<i>", "</i>"
		case "underline":
			open, close = "<u>", "</u>"
		case "strikethrough":
			open, close = "<s>", "</s>"
		case "code":
			open, close = "<code>", "</code>"
		case "pre":
			open, close = "<pre>", "</pre>"
		case "blockquote", "expandable_blockquote":
			open, close = "<blockquote>", "</blockquote>"
		case "text_link", "url":
			target := e.URL
			if e.Type == "url" {
				target = text[start:end]
			}
			parsed, err := url.Parse(target)
			if err == nil && (parsed.Scheme == "https" || parsed.Scheme == "http" || parsed.Scheme == "tg" || parsed.Scheme == "mailto") {
				open, close = `<a href="`+html.EscapeString(target)+`">`, "</a>"
			}
		}
		if open != "" {
			spans = append(spans, textEntity{start: start, end: end, open: open, close: close})
		}
	}
	slices.SortStableFunc(spans, func(a, b textEntity) int { return cmp.Or(cmp.Compare(a.start, b.start), cmp.Compare(b.end, a.end)) })
	var result strings.Builder
	stack := []textEntity{}
	cursor := 0
	for _, span := range spans {
		for len(stack) > 0 && stack[len(stack)-1].end <= span.start {
			parent := stack[len(stack)-1]
			result.WriteString(html.EscapeString(text[cursor:parent.end]))
			result.WriteString(parent.close)
			cursor = parent.end
			stack = stack[:len(stack)-1]
		}
		if span.start < cursor || (len(stack) > 0 && span.end > stack[len(stack)-1].end) {
			continue
		}
		result.WriteString(html.EscapeString(text[cursor:span.start]))
		result.WriteString(span.open)
		cursor = span.start
		stack = append(stack, span)
	}
	for len(stack) > 0 {
		span := stack[len(stack)-1]
		result.WriteString(html.EscapeString(text[cursor:span.end]))
		result.WriteString(span.close)
		cursor = span.end
		stack = stack[:len(stack)-1]
	}
	result.WriteString(html.EscapeString(text[cursor:]))
	return result.String()
}
