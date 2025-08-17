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

type HTMLChunk struct {
	Text         string
	OriginalHTML string
	ChunkIndex   int
	TotalChunks  int
	HasHeading   bool
	HasCode      bool
	HasTable     bool
	Metadata     map[string]any
}

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

// DefaultChunkConfig returns a ChunkConfig with sensible default values for HTML chunking.
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

// validate checks that the ChunkConfig has valid settings and sets defaults for missing values.
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

// defaultTokenizer estimates token count from text using a conservative rune-based approach.
// It works reliably across different AI providers and languages.
func defaultTokenizer(text string) int {
	// Rune-based tokenizer for plain text (HTML already stripped by callers)
	// Works reliably across all AI providers and languages

	// Count Unicode runes (actual characters) not bytes
	textRunes := utf8.RuneCountInString(text)

	// Conservative ratio: ~0.4 tokens per rune (pessimistic across all providers)
	return int(float64(textRunes) * 0.4)
}

var (
	headingRegex  = regexp.MustCompile(`(?i)<h[1-6][^>]*>`)
	codeRegex     = regexp.MustCompile(`(?i)<(pre|code)[^>]*>`)
	tableRegex    = regexp.MustCompile(`(?i)<table[^>]*>`)
	sentenceRegex = regexp.MustCompile(`[.!?]+[\s]+`)
)

// ChunkHTMLContent splits HTML content into semantically meaningful chunks suitable for AI processing.
// It preserves HTML structure boundaries while respecting token limits and creating overlapping chunks for better context continuity.
func ChunkHTMLContent(title, htmlContent string, config ...ChunkConfig) ([]HTMLChunk, error) {
	cfg := DefaultChunkConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	if strings.TrimSpace(htmlContent) == "" {
		return []HTMLChunk{{
			Text:         buildEmbeddingText(title, ""),
			OriginalHTML: htmlContent,
			ChunkIndex:   0,
			TotalChunks:  1,
			Metadata:     map[string]any{"empty": true},
		}}, nil
	}

	// First we create all HTML boundaries
	boundaries, err := parseHTMLBoundaries(htmlContent, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Now create chunks from the boundaries while making sure to respect token limits, create overlapping chunks, all heading elements
	chunks := createChunks(boundaries, cfg)
	result := make([]HTMLChunk, len(chunks))

	for i, chunk := range chunks {
		cleanText := HTML2Text(chunk.Content)
		result[i] = HTMLChunk{
			Text:         buildEmbeddingText(title, cleanText),
			OriginalHTML: chunk.Content,
			ChunkIndex:   i,
			TotalChunks:  len(chunks),
			HasHeading:   headingRegex.MatchString(chunk.Content),
			HasCode:      codeRegex.MatchString(chunk.Content),
			HasTable:     tableRegex.MatchString(chunk.Content),
			Metadata: map[string]any{
				"tokens":   chunk.Tokens,
				"priority": chunk.Priority,
			},
		}
	}

	return result, nil
}

// isBlockElement determines if a tag represents a block-level element.
func isBlockElement(tag string) bool {
	blockElements := map[string]bool{
		// Headings
		"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
		// Paragraphs and divisions
		"p": true, "div": true, "section": true, "article": true, "aside": true,
		"header": true, "footer": true, "main": true, "nav": true,
		// Lists
		"ul": true, "ol": true, "li": true, "dl": true, "dt": true, "dd": true,
		// Tables
		"table": true, "thead": true, "tbody": true, "tfoot": true, "tr": true, "td": true, "th": true,
		// Form elements
		"form": true, "fieldset": true, "legend": true,
		// Other block elements
		"blockquote": true, "pre": true, "code": true, "figure": true, "figcaption": true,
		"address": true, "details": true, "summary": true, "hr": true,
	}
	return blockElements[tag]
}

// parseHTMLBoundaries extracts block-level HTML elements and creates boundaries for chunking.
func parseHTMLBoundaries(htmlContent string, cfg ChunkConfig) ([]htmlBoundary, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, err
	}

	var boundaries []htmlBoundary

	var extract func(*html.Node, int) int
	extract = func(n *html.Node, startPos int) int {
		if n.Type == html.ElementNode {
			// Get tag name
			tag := strings.ToLower(n.Data)

			// Render the HTML element
			var content strings.Builder
			html.Render(&content, n)
			contentStr := content.String()

			// Skip empty elements
			cleanText := HTML2Text(contentStr)
			if strings.TrimSpace(cleanText) == "" {
				return startPos
			}

			// Create boundary for ALL block-level elements, not just high-priority ones
			if isBlockElement(tag) {
				boundaries = append(boundaries, htmlBoundary{
					Type:     tag,
					Content:  contentStr,
					Priority: getPriority(tag),
					Tokens:   cfg.TokenizerFunc(cleanText),
				})
				return startPos + len(contentStr)
			}
		}

		// Recursively process children to find boundaries / block elements
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			startPos = extract(c, startPos)
		}
		return startPos
	}

	extract(doc, 0)

	// Merge adjacent text nodes and small boundaries
	return mergeBoundaries(boundaries, cfg), nil
}

// getPriority assigns priority levels to HTML tags for chunking decisions.
// Lower numbers indicate higher priority.
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

// isPreservedBlock checks if a block type should be preserved based on the configuration.
func isPreservedBlock(blockType string, preserveBlocks []string) bool {
	return slices.Contains(preserveBlocks, blockType)
}

// mergeBoundaries combines adjacent HTML boundaries based on priority and token limits.
func mergeBoundaries(boundaries []htmlBoundary, cfg ChunkConfig) []htmlBoundary {
	if len(boundaries) == 0 {
		return boundaries
	}

	var merged []htmlBoundary
	current := boundaries[0]

	for i := 1; i < len(boundaries); i++ {
		next := boundaries[i]

		// Don't merge across high-priority boundaries (h1, h2)
		if next.Priority == 1 {
			merged = append(merged, current)
			current = next
			continue
		}

		// Don't merge if current is h1/h2 and has sufficient content
		if current.Priority == 1 && current.Tokens >= cfg.MinTokens {
			merged = append(merged, current)
			current = next
			continue
		}

		// For preserved blocks, don't merge with other content
		if isPreservedBlock(current.Type, cfg.PreserveBlocks) || isPreservedBlock(next.Type, cfg.PreserveBlocks) {
			merged = append(merged, current)
			current = next
			continue
		}

		// Merge adjacent boundaries if:
		// 1. Combined tokens are under MinTokens, OR
		// 2. Both are low-priority (p, div, li) and combined tokens < maxTokens
		combinedTokens := current.Tokens + next.Tokens
		shouldMerge := false

		if combinedTokens < cfg.MinTokens {
			shouldMerge = true
		} else if current.Priority >= 3 && next.Priority >= 3 && combinedTokens < cfg.MaxTokens {
			// Merge small adjacent low-priority elements (paragraphs, divs, etc.)
			shouldMerge = true
		}

		if shouldMerge {
			current.Content += next.Content
			current.Tokens = combinedTokens
			// Keep the higher priority (lower number)
			current.Priority = min(current.Priority, next.Priority)
		} else {
			merged = append(merged, current)
			current = next
		}
	}

	merged = append(merged, current)
	return merged
}

// truncateOversizedContent simply truncates content that exceeds max tokens.
// Logs a warning for admins to fix the content at the source.
func truncateOversizedContent(boundary htmlBoundary, cfg ChunkConfig) htmlBoundary {
	text := HTML2Text(boundary.Content)
	if cfg.TokenizerFunc(text) <= cfg.MaxTokens {
		return boundary
	}

	// Cut text to fit max tokens
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

	// If we can't truncate to fit, return empty
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

// createChunks groups HTML boundaries into final chunks respecting token limits and overlap.
func createChunks(boundaries []htmlBoundary, cfg ChunkConfig) []htmlBoundary {
	if len(boundaries) == 0 {
		return boundaries
	}

	var chunks []htmlBoundary
	var currentChunk htmlBoundary
	currentChunk.Priority = 10 // Start with lowest priority

	for _, boundary := range boundaries {
		// Check if we should start a new chunk
		shouldStartNewChunk := false

		// Always start new chunk on h1/h2 headings (high priority) if current chunk has sufficient content
		if boundary.Priority == 1 && currentChunk.Tokens >= cfg.MinTokens {
			shouldStartNewChunk = true
		}

		// Check if adding this boundary would exceed MaxTokens
		if currentChunk.Tokens+boundary.Tokens > cfg.MaxTokens {
			// Only create a chunk if we have some content
			if currentChunk.Content != "" {
				shouldStartNewChunk = true
			}
		}

		// If we need to start a new chunk, finalize the current one
		if shouldStartNewChunk && currentChunk.Content != "" {
			chunks = append(chunks, currentChunk)

			// Start new chunk with overlap if appropriate
			var overlapContent string
			if !isPreservedBlock(boundary.Type, cfg.PreserveBlocks) && len(chunks) > 0 {
				overlapContent = extractOverlap(currentChunk.Content, cfg)
			}

			currentChunk = htmlBoundary{
				Content:  overlapContent,
				Tokens:   cfg.TokenizerFunc(HTML2Text(overlapContent)),
				Priority: 10, // Reset to lowest priority
			}
		}

		// Add current boundary to the chunk
		currentChunk.Content += boundary.Content
		currentChunk.Tokens += boundary.Tokens

		// Update chunk priority to highest priority element it contains
		if boundary.Priority < currentChunk.Priority {
			currentChunk.Priority = boundary.Priority
		}

		// Handle case where single boundary exceeds MaxTokens
		if currentChunk.Tokens > cfg.MaxTokens && currentChunk.Content == boundary.Content {
			// Single boundary is too large, truncate it
			truncatedBoundary := truncateOversizedContent(boundary, cfg)
			chunks = append(chunks, truncatedBoundary)
			currentChunk = htmlBoundary{Priority: 10}
		}
	}

	// Add the final chunk if it has content
	if currentChunk.Content != "" {
		chunks = append(chunks, currentChunk)
	}

	return chunks
}

// extractOverlap creates overlapping content from the end of a chunk for context continuity.
func extractOverlap(content string, cfg ChunkConfig) string {
	cleanText := HTML2Text(content)
	sentences := sentenceRegex.Split(cleanText, -1)

	if len(sentences) <= 1 {
		return ""
	}

	// Take last N sentences that fit in overlap tokens
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

// buildEmbeddingText formats title and content text for AI embedding processing.
// It creates a structured format that helps AI models understand the semantic
// relationship between title and content. When title is empty, only the content
// is returned to avoid redundant "Title: " prefixes in embeddings.
func buildEmbeddingText(title, cleanText string) string {
	title = strings.TrimSpace(title)
	cleanText = strings.TrimSpace(cleanText)

	// If no title is provided, return content as-is to avoid empty "Title: " prefixes
	if title == "" {
		return cleanText
	}

	// If no content is provided, return only the title
	if cleanText == "" {
		return title
	}

	// Both title and content are present, use structured format
	return fmt.Sprintf("Title: %s\nContent: %s", title, cleanText)
}
