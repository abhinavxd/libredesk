package stringutil

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/zerodha/logf"
	"golang.org/x/net/html"
)

var (
	sentenceRegex     = regexp.MustCompile(`[.!?]+[\s]+`)
	headingInnerRegex = regexp.MustCompile(`(?is)<h[1-6][^>]*>(.*?)</h[1-6]>`)
)

type ChunkConfig struct {
	MaxTokens      int
	MinTokens      int
	OverlapTokens  int
	TokenizerFunc  func(string) int
	PreserveBlocks []string
	Logger         *logf.Logger
}

type htmlBoundary struct {
	Type     string
	Content  string
	Priority int
	Tokens   int
}

// DefaultChunkConfig returns a ChunkConfig with sensible defaults for HTML chunking.
func DefaultChunkConfig() ChunkConfig {
	return ChunkConfig{
		MaxTokens:      2000,
		MinTokens:      400,
		OverlapTokens:  75,
		TokenizerFunc:  defaultTokenizer,
		PreserveBlocks: []string{"pre", "code", "table"},
		Logger:         nil,
	}
}

func (c *ChunkConfig) validate() error {
	if c.MaxTokens <= c.MinTokens {
		return fmt.Errorf("MaxTokens must be greater than MinTokens")
	}
	if c.OverlapTokens >= c.MinTokens {
		return fmt.Errorf("OverlapTokens must be less than MinTokens")
	}
	if c.TokenizerFunc == nil {
		c.TokenizerFunc = defaultTokenizer
	}
	return nil
}

// ChunkHTMLContent splits HTML into structure-aware chunks for embedding, prepending the title and section heading to each chunk's text.
func ChunkHTMLContent(title, htmlContent string, config ...ChunkConfig) ([]string, error) {
	cfg := DefaultChunkConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	if strings.TrimSpace(htmlContent) == "" {
		return []string{buildEmbeddingText(title, "", "")}, nil
	}

	boundaries, err := parseHTMLBoundaries(htmlContent, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Plain text (or any text not wrapped in a block element) yields no boundaries;
	// keep it as a single chunk so it still gets embedded.
	if len(boundaries) == 0 {
		if text := strings.TrimSpace(HTML2Text(htmlContent)); text != "" {
			return []string{buildEmbeddingText(title, "", text)}, nil
		}
	}

	chunks := createChunks(boundaries, cfg)
	result := make([]string, len(chunks))

	var lastHeading string
	for i, chunk := range chunks {
		if h := extractLeadingHeading(chunk.Content); h != "" {
			lastHeading = h
		}
		result[i] = buildEmbeddingText(title, lastHeading, HTML2Text(chunk.Content))
	}

	return result, nil
}

// defaultTokenizer estimates token count with a conservative rune-based ratio.
func defaultTokenizer(text string) int {
	return int(float64(utf8.RuneCountInString(text)) * 0.4)
}

func isBlockElement(tag string) bool {
	blockElements := map[string]bool{
		"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
		"p": true, "div": true, "section": true, "article": true, "aside": true,
		"header": true, "footer": true, "main": true, "nav": true,
		"ul": true, "ol": true, "li": true, "dl": true, "dt": true, "dd": true,
		"table": true, "thead": true, "tbody": true, "tfoot": true, "tr": true, "td": true, "th": true,
		"form": true, "fieldset": true, "legend": true,
		"blockquote": true, "pre": true, "code": true, "figure": true, "figcaption": true,
		"address": true, "details": true, "summary": true, "hr": true,
	}
	return blockElements[tag]
}

func parseHTMLBoundaries(htmlContent string, cfg ChunkConfig) ([]htmlBoundary, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, err
	}

	var boundaries []htmlBoundary

	var extract func(*html.Node)
	extract = func(n *html.Node) {
		if n.Type == html.ElementNode {
			tag := strings.ToLower(n.Data)

			var content strings.Builder
			html.Render(&content, n)
			contentStr := content.String()

			cleanText := HTML2Text(contentStr)
			if strings.TrimSpace(cleanText) == "" {
				return
			}

			if isBlockElement(tag) {
				tokens := cfg.TokenizerFunc(cleanText)
				// An oversized container taken as one atomic boundary would be truncated at
				// MaxTokens, silently dropping the rest; descend into its block children instead.
				if tokens <= cfg.MaxTokens || isPreservedBlock(tag, cfg.PreserveBlocks) || !splittableIntoBlocks(n) {
					boundaries = append(boundaries, htmlBoundary{
						Type:     tag,
						Content:  contentStr,
						Priority: getPriority(tag),
						Tokens:   tokens,
					})
					return
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}

	extract(doc)

	return mergeBoundaries(boundaries, cfg), nil
}

// getPriority ranks tags for chunking; lower is higher priority.
func getPriority(tag string) int {
	switch tag {
	case "h1", "h2":
		return 1
	case "h3", "h4", "h5", "h6", "pre", "code":
		return 2
	case "p", "table", "ul", "ol", "blockquote":
		return 3
	case "div", "section", "article", "figure":
		return 4
	default:
		return 5
	}
}

func isPreservedBlock(blockType string, preserveBlocks []string) bool {
	return slices.Contains(preserveBlocks, blockType)
}

// splittableIntoBlocks reports whether all of a node's visible content lives inside block-element
// children, so splitting the node into its children loses nothing.
func splittableIntoBlocks(n *html.Node) bool {
	hasBlock := false
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		switch c.Type {
		case html.TextNode:
			if strings.TrimSpace(c.Data) != "" {
				return false
			}
		case html.ElementNode:
			if isBlockElement(strings.ToLower(c.Data)) {
				hasBlock = true
				continue
			}
			var buf strings.Builder
			html.Render(&buf, c)
			if strings.TrimSpace(HTML2Text(buf.String())) != "" {
				return false
			}
		}
	}
	return hasBlock
}

func mergeBoundaries(boundaries []htmlBoundary, cfg ChunkConfig) []htmlBoundary {
	if len(boundaries) == 0 {
		return boundaries
	}

	var merged []htmlBoundary
	current := boundaries[0]

	for i := 1; i < len(boundaries); i++ {
		next := boundaries[i]

		if next.Priority == 1 {
			merged = append(merged, current)
			current = next
			continue
		}

		if current.Priority == 1 && current.Tokens >= cfg.MinTokens {
			merged = append(merged, current)
			current = next
			continue
		}

		if isPreservedBlock(current.Type, cfg.PreserveBlocks) || isPreservedBlock(next.Type, cfg.PreserveBlocks) {
			merged = append(merged, current)
			current = next
			continue
		}

		combinedTokens := current.Tokens + next.Tokens
		shouldMerge := false

		if combinedTokens < cfg.MinTokens {
			shouldMerge = true
		} else if current.Priority >= 3 && next.Priority >= 3 && combinedTokens < cfg.MaxTokens {
			shouldMerge = true
		}

		if shouldMerge {
			current.Content += next.Content
			current.Tokens = combinedTokens
			current.Priority = min(current.Priority, next.Priority)
		} else {
			merged = append(merged, current)
			current = next
		}
	}

	merged = append(merged, current)
	return merged
}

// truncateOversizedContent trims content past MaxTokens rune-by-rune, warning admins to fix the source.
func truncateOversizedContent(boundary htmlBoundary, cfg ChunkConfig) htmlBoundary {
	text := HTML2Text(boundary.Content)
	if cfg.TokenizerFunc(text) <= cfg.MaxTokens {
		return boundary
	}

	runes := []rune(text)
	for i := 1; i <= len(runes); i++ {
		truncated := string(runes[:len(runes)-i])
		if cfg.TokenizerFunc(truncated) <= cfg.MaxTokens {
			if cfg.Logger != nil {
				cfg.Logger.Warn("Content truncated: exceeded max_tokens",
					"type", boundary.Type,
					"original_tokens", boundary.Tokens,
					"truncated_tokens", cfg.TokenizerFunc(truncated))
			}
			boundary.Content = truncated
			boundary.Tokens = cfg.TokenizerFunc(truncated)
			return boundary
		}
	}

	if cfg.Logger != nil {
		cfg.Logger.Error("Content completely emptied: could not truncate to fit max_tokens",
			"type", boundary.Type,
			"original_tokens", boundary.Tokens,
			"max_tokens", cfg.MaxTokens)
	}
	boundary.Content = ""
	boundary.Tokens = 0
	return boundary
}

func createChunks(boundaries []htmlBoundary, cfg ChunkConfig) []htmlBoundary {
	if len(boundaries) == 0 {
		return boundaries
	}

	var chunks []htmlBoundary
	var currentChunk htmlBoundary
	currentChunk.Priority = 10

	for _, boundary := range boundaries {
		shouldStartNewChunk := false

		if boundary.Priority == 1 && currentChunk.Tokens >= cfg.MinTokens {
			shouldStartNewChunk = true
		}

		if currentChunk.Tokens+boundary.Tokens > cfg.MaxTokens {
			if currentChunk.Content != "" {
				shouldStartNewChunk = true
			}
		}

		if shouldStartNewChunk && currentChunk.Content != "" {
			chunks = append(chunks, currentChunk)

			var overlapContent string
			if !isPreservedBlock(boundary.Type, cfg.PreserveBlocks) {
				overlapContent = extractOverlap(currentChunk.Content, cfg)
			}

			currentChunk = htmlBoundary{
				Content:  overlapContent,
				Tokens:   cfg.TokenizerFunc(HTML2Text(overlapContent)),
				Priority: 10,
			}
		}

		currentChunk.Content += boundary.Content
		currentChunk.Tokens += boundary.Tokens

		if boundary.Priority < currentChunk.Priority {
			currentChunk.Priority = boundary.Priority
		}

		if currentChunk.Tokens > cfg.MaxTokens && currentChunk.Content == boundary.Content {
			truncatedBoundary := truncateOversizedContent(boundary, cfg)
			chunks = append(chunks, truncatedBoundary)
			currentChunk = htmlBoundary{Priority: 10}
		}
	}

	if currentChunk.Content != "" {
		chunks = append(chunks, currentChunk)
	}

	return chunks
}

// extractOverlap carries trailing whole sentences into the next chunk for context continuity.
func extractOverlap(content string, cfg ChunkConfig) string {
	cleanText := HTML2Text(content)
	sentences := sentenceRegex.Split(cleanText, -1)

	if len(sentences) <= 1 {
		return ""
	}

	var overlap []string
	tokens := 0
	for i := len(sentences) - 1; i >= 0 && tokens < cfg.OverlapTokens; i-- {
		sentence := strings.TrimSpace(sentences[i])
		if sentence == "" {
			continue
		}
		sentTokens := cfg.TokenizerFunc(sentence)
		if tokens+sentTokens <= cfg.OverlapTokens {
			overlap = append([]string{sentence}, overlap...)
			tokens += sentTokens
		} else {
			break
		}
	}

	if len(overlap) == 0 {
		return ""
	}

	return "<p>" + strings.Join(overlap, ". ") + ".</p>\n"
}

// extractLeadingHeading returns the plain text of the first heading in the chunk HTML, if any.
func extractLeadingHeading(htmlContent string) string {
	m := headingInnerRegex.FindStringSubmatch(htmlContent)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(HTML2Text(m[1]))
}

// buildEmbeddingText prepends the title and section heading (when present) to the chunk text.
func buildEmbeddingText(title, heading, cleanText string) string {
	title = strings.TrimSpace(title)
	heading = strings.TrimSpace(heading)
	cleanText = strings.TrimSpace(cleanText)

	if title == "" && heading == "" {
		return cleanText
	}
	if cleanText == "" {
		return title
	}

	var b strings.Builder
	if title != "" {
		fmt.Fprintf(&b, "Title: %s\n", title)
	}
	if heading != "" && !strings.EqualFold(heading, title) {
		fmt.Fprintf(&b, "Section: %s\n", heading)
	}
	fmt.Fprintf(&b, "Content: %s", cleanText)
	return b.String()
}
