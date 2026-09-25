package s3

import (
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/media/models"
	"github.com/google/uuid"
)

func TestPutStoresSafeContentHeaders(t *testing.T) {
	tests := []struct {
		contentType string
		wantType    string
		disposition string
	}{
		{"image/svg+xml", "image/svg+xml", "attachment"},
		{"IMAGE/SVG+XML; x", "image/svg+xml", "attachment"},
		{`image/svg+xml; charset="unterminated`, "image/svg+xml", "attachment"},
		{"image/svg+xml; charset=utf-8; charset=ascii", "application/octet-stream", "attachment"},
		{"text/plain,image/svg+xml", "application/octet-stream", "attachment"},
		{"image/x+xml", "image/x+xml", "attachment"},
		{"text/html", "text/html", "attachment"},
		{"", "application/octet-stream", "attachment"},
		{"image/png", "image/png", "inline"},
		{"video/mp4", "video/mp4", "inline"},
		{"application/pdf", "application/pdf", "inline"},
	}
	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			body := `<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`
			requests := make(chan http.Header, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPut || r.URL.Path != "/test-bucket/uploads/file" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				got, err := io.ReadAll(r.Body)
				if err != nil || string(got) != body {
					t.Errorf("uploaded body = %q, error = %v", got, err)
				}
				requests <- r.Header.Clone()
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			store, err := New(Opt{
				URL:        server.URL,
				PublicURL:  "https://cdn.example.com",
				AccessKey:  uuid.NewString(),
				SecretKey:  uuid.NewString(),
				Region:     "test-region",
				Bucket:     "test-bucket",
				BucketPath: "uploads",
				BucketType: "private",
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := store.Put("file", tt.contentType, strings.NewReader(body)); err != nil {
				t.Fatal(err)
			}

			headers := <-requests
			if got := headers.Get("Content-Type"); got != tt.wantType {
				t.Errorf("Content-Type = %q, want %q", got, tt.wantType)
			}
			if got := headers.Get("Content-Disposition"); got != tt.disposition {
				t.Errorf("Content-Disposition = %q, want %q", got, tt.disposition)
			}
		})
	}
}

func TestGetURLPreservesPublicURLs(t *testing.T) {
	tests := []struct {
		name        string
		bucketType  string
		publicURL   string
		disposition string
		wantHost    string
		wantPath    string
		wantSigned  bool
	}{
		{"private download", "private", "", models.DispositionAttachment, "storage.example.com", "/test-bucket/uploads/existing-file", true},
		{"private inline", "private", "", models.DispositionInline, "storage.example.com", "/test-bucket/uploads/existing-file", true},
		{"private custom download", "private", "https://cdn.example.com", models.DispositionAttachment, "cdn.example.com", "/uploads/existing-file", false},
		{"private custom inline", "private", "https://cdn.example.com", models.DispositionInline, "cdn.example.com", "/uploads/existing-file", false},
		{"public download", "public", "", models.DispositionAttachment, "storage.example.com", "/test-bucket/uploads/existing-file", false},
		{"public inline", "public", "", models.DispositionInline, "storage.example.com", "/test-bucket/uploads/existing-file", false},
		{"public custom download", "public", "https://cdn.example.com", models.DispositionAttachment, "cdn.example.com", "/uploads/existing-file", false},
		{"public custom inline", "public", "https://cdn.example.com", models.DispositionInline, "cdn.example.com", "/uploads/existing-file", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := New(Opt{
				URL:        "https://storage.example.com",
				PublicURL:  tt.publicURL,
				AccessKey:  uuid.NewString(),
				SecretKey:  uuid.NewString(),
				Region:     "test-region",
				Bucket:     "test-bucket",
				BucketPath: "uploads",
				BucketType: tt.bucketType,
			})
			if err != nil {
				t.Fatal(err)
			}
			filename := "customer attachment.svg"
			u, err := url.Parse(store.GetURL("existing-file", tt.disposition, filename))
			if err != nil {
				t.Fatal(err)
			}

			if u.Host != tt.wantHost || u.Path != tt.wantPath {
				t.Errorf("URL target = %s%s, want %s%s", u.Host, u.Path, tt.wantHost, tt.wantPath)
			}
			query := u.Query()
			if got := query.Get("X-Amz-Signature") != ""; got != tt.wantSigned {
				t.Errorf("signed URL = %v, want %v", got, tt.wantSigned)
			}
			if tt.wantSigned {
				got, params, err := mime.ParseMediaType(query.Get("response-content-disposition"))
				if err != nil || got != tt.disposition || params["filename"] != filename {
					t.Errorf("download disposition = %q, filename = %q, error = %v", got, params["filename"], err)
				}
			} else if u.RawQuery != "" {
				t.Error("public URL has unexpected query parameters")
			}
		})
	}
}
