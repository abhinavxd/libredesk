package stringutil

import (
	"strings"

	readability "github.com/mackee/go-readability"
)

// ExtractReadableContent returns the title and readable content (as Markdown) of an HTML page.
// Returns empty content when the page has no article-like content to extract.
func ExtractReadableContent(htmlDoc string) (string, string) {
	// Default char threshold (500) leaves Root nil on short support pages; lower it so they
	// still get extracted.
	opts := readability.DefaultOptions()
	opts.CharThreshold = 100
	art, err := readability.Extract(htmlDoc, opts)
	if err != nil || art.Root == nil {
		return "", ""
	}
	return art.Title, strings.TrimSpace(readability.ToMarkdown(art.Root))
}
