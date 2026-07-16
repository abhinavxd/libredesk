package ai

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	aimodels "github.com/abhinavxd/libredesk/internal/ai/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/stringutil"
)

const (
	urlImportTimeout  = 20 * time.Second
	urlImportMaxBytes = 3 << 20
)

// ImportKnowledgeBaseFromURL fetches a page and stores its readable content as a knowledge base item.
func (m *Manager) ImportKnowledgeBaseFromURL(ctx context.Context, rawURL string) (aimodels.KnowledgeBaseItem, error) {
	var item aimodels.KnowledgeBaseItem
	rawURL = strings.TrimSpace(rawURL)
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return item, envelope.NewError(envelope.InputError, m.i18n.T("ai.urlImport.invalidUrl"), nil)
	}

	body, contentType, err := m.fetchURL(ctx, u.String())
	if err != nil {
		m.lo.Error("error fetching knowledge base import url", "host", u.Hostname(), "error", err)
		return item, envelope.NewError(envelope.InputError, m.i18n.T("ai.urlImport.fetchFailed"), nil)
	}
	if !isHTMLContentType(contentType) {
		return item, envelope.NewError(envelope.InputError, m.i18n.T("ai.urlImport.unsupportedContentType"), nil)
	}

	title, content := stringutil.ExtractReadableContent(body)
	if content == "" {
		return item, envelope.NewError(envelope.InputError, m.i18n.T("ai.urlImport.noContent"), nil)
	}
	if title == "" {
		title = u.Host + u.Path
	}
	return m.CreateKnowledgeBaseItem(title, content, aimodels.KnowledgeSourceURL, u.String(), true)
}

func (m *Manager) fetchURL(ctx context.Context, pageURL string) (string, string, error) {
	ctx, cancel := context.WithTimeout(ctx, urlImportTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", "libredesk")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, urlImportMaxBytes))
	if err != nil {
		return "", "", err
	}
	return string(body), resp.Header.Get("Content-Type"), nil
}

func isHTMLContentType(contentType string) bool {
	if contentType == "" {
		return true
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}
	return mediaType == "text/html" || mediaType == "application/xhtml+xml"
}
