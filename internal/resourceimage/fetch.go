package resourceimage

import (
	"bytes"
	"context"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"net/http"
	"time"

	"github.com/abhinavxd/libredesk/internal/resourcepolicy"
	"github.com/abhinavxd/libredesk/internal/ssrf"
	"github.com/abhinavxd/ssrfguard"
	_ "golang.org/x/image/webp"
)

const (
	MaxDownloadBytes = 5 << 20
	MaxImageBytes    = 8 << 20
	maxPixels        = 8_000_000
)

var ErrImage = errors.New("external image unavailable or unsupported")

type Fetcher struct {
	client  *http.Client
	slots   chan struct{}
	pending chan struct{}
}

func NewFetcher() *Fetcher {
	transport := ssrf.NewTransport(ssrfguard.New().Control, 5*time.Second)
	transport.Proxy = nil
	transport.DisableCompression = true
	transport.MaxResponseHeaderBytes = 16 << 10
	transport.ResponseHeaderTimeout = 5 * time.Second
	transport.TLSHandshakeTimeout = 5 * time.Second
	return &Fetcher{
		client: &http.Client{
			Transport:     transport,
			Timeout:       10 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
		slots:   make(chan struct{}, 4),
		pending: make(chan struct{}, 64),
	}
}

func (f *Fetcher) Fetch(ctx context.Context, raw string) ([]byte, error) {
	return f.FetchAuthorized(ctx, raw, nil)
}

func (f *Fetcher) FetchAuthorized(ctx context.Context, raw string, authorize func() error) ([]byte, error) {
	if !resourcepolicy.ValidImageURL(raw) {
		return nil, ErrImage
	}
	select {
	case f.pending <- struct{}{}:
		defer func() { <-f.pending }()
	default:
		return nil, ErrImage
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	select {
	case f.slots <- struct{}{}:
		defer func() { <-f.slots }()
	case <-ctx.Done():
		return nil, ErrImage
	}
	if authorize != nil {
		if err := authorize(); err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return nil, ErrImage
	}
	req.Header.Set("User-Agent", "LibreDesk-Image-Proxy")
	req.Header.Set("Accept", "image/png,image/jpeg,image/gif,image/webp")
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, ErrImage
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || resp.ContentLength > MaxDownloadBytes || resp.Header.Get("Content-Encoding") != "" {
		return nil, ErrImage
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxDownloadBytes+1))
	if err != nil || len(body) > MaxDownloadBytes {
		return nil, ErrImage
	}
	return normalize(body)
}

func normalize(body []byte) ([]byte, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(body))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > maxPixels/cfg.Height {
		return nil, ErrImage
	}
	decoded, _, err := image.Decode(bytes.NewReader(body))
	if err != nil {
		return nil, ErrImage
	}
	var output boundedBuffer
	if err := png.Encode(&output, decoded); err != nil {
		return nil, ErrImage
	}
	return output.Bytes(), nil
}

type boundedBuffer struct{ bytes.Buffer }

func (b *boundedBuffer) Write(p []byte) (int, error) {
	if b.Len()+len(p) > MaxImageBytes {
		return 0, ErrImage
	}
	return b.Buffer.Write(p)
}
