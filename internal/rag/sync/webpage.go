package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/rag"
	"github.com/abhinavxd/libredesk/internal/rag/models"
	"github.com/zerodha/logf"
)

// WebpageSyncer syncs web pages to the RAG knowledge base.
type WebpageSyncer struct {
	rag    *rag.Manager
	lo     *logf.Logger
	client *http.Client
}

// NewWebpageSyncer creates a new webpage syncer.
func NewWebpageSyncer(ragMgr *rag.Manager, lo *logf.Logger) *WebpageSyncer {
	return &WebpageSyncer{
		rag: ragMgr,
		lo:  lo,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Sync syncs all configured web pages to the RAG documents table.
func (s *WebpageSyncer) Sync(sourceID int, config models.WebpageConfig) error {
	s.lo.Info("starting webpage sync", "source_id", sourceID, "urls", len(config.URLs))

	syncedRefs := make(map[string]bool)

	for _, url := range config.URLs {
		url = strings.TrimSpace(url)
		if url == "" {
			continue
		}

		sourceRef := fmt.Sprintf("webpage_%s", hashURL(url))
		syncedRefs[sourceRef] = true

		// Fetch and parse webpage
		title, content, err := s.fetchWebpage(url)
		if err != nil {
			s.lo.Error("error fetching webpage", "url", url, "error", err)
			continue
		}

		if strings.TrimSpace(content) == "" {
			s.lo.Warn("empty content from webpage", "url", url)
			continue
		}

		// Chunk long content
		chunks := chunkContent(content, 2000)

		for i, chunk := range chunks {
			chunkRef := sourceRef
			chunkTitle := title
			if len(chunks) > 1 {
				chunkRef = fmt.Sprintf("%s_chunk%d", sourceRef, i)
				chunkTitle = fmt.Sprintf("%s (Part %d)", title, i+1)
			}

			metadata, _ := json.Marshal(map[string]interface{}{
				"url":        url,
				"chunk":      i,
				"total":      len(chunks),
				"fetched_at": time.Now(),
			})

			if err := s.rag.AddDocument(sourceID, chunkRef, chunkTitle, chunk, metadata); err != nil {
				s.lo.Error("error syncing webpage chunk", "url", url, "chunk", i, "error", err)
				continue
			}
		}

		s.lo.Debug("synced webpage", "url", url, "chunks", len(chunks))
	}

	s.lo.Info("webpage sync complete", "source_id", sourceID, "synced", len(syncedRefs))
	return nil
}

// fetchWebpage fetches a URL and extracts title and text content.
func (s *WebpageSyncer) fetchWebpage(url string) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", "LibreDesk-RAG/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("fetching URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
	if err != nil {
		return "", "", fmt.Errorf("reading body: %w", err)
	}

	html := string(body)
	title := extractTitle(html)
	content := extractTextContent(html)

	return title, content, nil
}

// extractTitle extracts the page title from HTML.
func extractTitle(html string) string {
	re := regexp.MustCompile(`(?i)<title[^>]*>([^<]+)</title>`)
	matches := re.FindStringSubmatch(html)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return "Untitled Page"
}

// extractTextContent extracts readable text from HTML.
func extractTextContent(html string) string {
	// Remove script, style, and other non-content tags
	patterns := []string{
		`(?is)<script[^>]*>.*?</script>`,
		`(?is)<style[^>]*>.*?</style>`,
		`(?is)<noscript[^>]*>.*?</noscript>`,
		`(?is)<header[^>]*>.*?</header>`,
		`(?is)<footer[^>]*>.*?</footer>`,
		`(?is)<nav[^>]*>.*?</nav>`,
		`(?is)<!--.*?-->`,
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		html = re.ReplaceAllString(html, " ")
	}

	// Try to find main content areas
	mainContent := html
	for _, selector := range []string{`<main[^>]*>`, `<article[^>]*>`, `class="content"`, `id="content"`} {
		if strings.Contains(html, selector) {
			// Simple extraction - could be improved with proper HTML parsing
			break
		}
	}

	// Strip remaining HTML tags
	return stripHTML(mainContent)
}

// chunkContent splits content into chunks of approximately maxLen characters.
func chunkContent(content string, maxLen int) []string {
	if len(content) <= maxLen {
		return []string{content}
	}

	var chunks []string
	words := strings.Fields(content)
	var current strings.Builder

	for _, word := range words {
		if current.Len()+len(word)+1 > maxLen && current.Len() > 0 {
			chunks = append(chunks, strings.TrimSpace(current.String()))
			current.Reset()
		}
		if current.Len() > 0 {
			current.WriteString(" ")
		}
		current.WriteString(word)
	}

	if current.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(current.String()))
	}

	return chunks
}

// hashURL creates a short hash for a URL to use as source_ref.
func hashURL(url string) string {
	hash := rag.HashContent(url)
	if len(hash) > 16 {
		return hash[:16]
	}
	return hash
}
