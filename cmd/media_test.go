package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	mmodels "github.com/abhinavxd/libredesk/internal/media/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func TestPrepareImageUploadAllowsOversizedImageWithoutThumbnail(t *testing.T) {
	imageData := []byte{'G', 'I', 'F', '8', '9', 'a', 0x10, 0x27, 0x88, 0x13, 0x00, 0x00, 0x00}

	prepared, err := prepareImageUpload(bytes.NewReader(imageData))
	if err != nil {
		t.Fatal(err)
	}
	if prepared.thumbnail != nil {
		t.Fatal("expected thumbnail to be skipped")
	}
	if prepared.thumbnailErr == nil {
		t.Fatal("expected thumbnail error to be retained")
	}

	var meta struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	}
	if err := json.Unmarshal(prepared.meta, &meta); err != nil {
		t.Fatal(err)
	}
	if meta.Width != 10_000 || meta.Height != 5_000 {
		t.Fatalf("dimensions = %dx%d, want 10000x5000", meta.Width, meta.Height)
	}
}

func TestServeMediaFileSandboxesAllButPDF(t *testing.T) {
	const uuid = "0b7a3c1e-2f4d-4b8a-9c6e-1d2f3a4b5c6d"
	dir := t.TempDir()
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)
	if err := os.WriteFile(filepath.Join(dir, uuid), svg, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ko.Set("upload.fs.upload_path", dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ko.Delete("upload.fs.upload_path") })

	app := &App{}
	app.consts.Store(&constants{UploadProvider: "fs"})

	tests := []struct {
		name            string
		contentType     string
		download        bool
		wantType        string
		wantDisposition string
		wantSandbox     bool
	}{
		{"svg", "image/svg+xml", false, "image/svg+xml", "attachment", true},
		{"svg with bad param", "image/svg+xml; x", false, "image/svg+xml", "attachment", true},
		{"other xml image", "image/x+xml", false, "image/x+xml", "attachment", true},
		{"unparseable", "text/plain,image/svg+xml", false, "application/octet-stream", "attachment", true},
		{"png", "image/png", false, "image/png", "inline", true},
		{"png forced download", "image/png", true, "image/png", "attachment", true},
		{"pdf", "application/pdf", false, "application/pdf", "inline", false},
		{"pdf forced download", "application/pdf", true, "application/pdf", "attachment", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &fastglue.Request{RequestCtx: &fasthttp.RequestCtx{}, Context: app}
			if tt.download {
				req.RequestCtx.Request.SetRequestURI("/uploads/" + uuid + "?download=1")
			}
			media := &mmodels.Media{ContentType: tt.contentType, Filename: "file", Private: true}
			if err := serveMediaFile(req, app, uuid, media); err != nil {
				t.Fatal(err)
			}

			h := &req.RequestCtx.Response.Header
			if got := string(h.ContentType()); got != tt.wantType {
				t.Fatalf("Content-Type = %q, want %q", got, tt.wantType)
			}
			if got := string(h.Peek("Content-Disposition")); !strings.HasPrefix(got, tt.wantDisposition) {
				t.Fatalf("Content-Disposition = %q, want %s", got, tt.wantDisposition)
			}
			if got := string(h.Peek("Content-Security-Policy")) == "sandbox"; got != tt.wantSandbox {
				t.Fatalf("sandboxed = %v, want %v", got, tt.wantSandbox)
			}
		})
	}
}
