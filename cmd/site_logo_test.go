package main

import (
	"bytes"
	"mime/multipart"
	"testing"

	"github.com/jakedolan443/fernmail/internal/testutil"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
)

func TestIsValidLogoURL(t *testing.T) {
	for u, want := range map[string]bool{
		"": true,
		"/uploads/550e8400-e29b-41d4-a716-446655440000":            true,
		"https://cdn.example.com/logo.png":                         true,
		"/uploads/550e8400-e29b-41d4-a716-446655440000/../secrets": false,
		"/uploads/not-a-uuid":                                      false,
		"/images/fern.svg":                                         false,
		"javascript:alert(1)":                                      false,
		"logo.png":                                                 false,
	} {
		if got := isValidLogoURL(u); got != want {
			t.Errorf("isValidLogoURL(%q) = %v, want %v", u, got, want)
		}
	}
}

func TestSiteLogoContentTypeSniffsRasterImages(t *testing.T) {
	for name, tc := range map[string]struct {
		head []byte
		want bool
	}{
		"png":   {[]byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR"), true},
		"jpeg":  {[]byte("\xff\xd8\xff\xe0\x00\x10JFIF"), true},
		"webp":  {[]byte("RIFF\x24\x00\x00\x00WEBPVP8 "), true},
		"ico":   {[]byte("\x00\x00\x01\x00\x01\x00\x10\x10"), true},
		"svg":   {[]byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`), false},
		"gif":   {[]byte("GIF89a\x01\x00\x01\x00"), false},
		"html":  {[]byte("<!doctype html><title>x</title>"), false},
		"empty": {nil, false},
	} {
		if _, got := siteLogoContentType(tc.head); got != tc.want {
			t.Errorf("%s: accepted = %v, want %v", name, got, tc.want)
		}
	}
}

func TestBrandingConstantsFallBackToFernmail(t *testing.T) {
	keys := []string{"app.root_url", "app.site_name", "app.logo_url", "upload.provider"}
	t.Cleanup(func() {
		for _, k := range keys {
			ko.Delete(k)
		}
	})
	for _, tc := range []struct {
		name, title, logo                string
		wantTitle, wantLogo, wantFavicon string
	}{
		{"unset", "  ", "", "Fernmail", "", "/favicon.svg"},
		{"uploaded logo", " Acme ", "/uploads/550e8400-e29b-41d4-a716-446655440000", "Acme",
			"https://mail.example.com/uploads/550e8400-e29b-41d4-a716-446655440000",
			"https://mail.example.com/uploads/550e8400-e29b-41d4-a716-446655440000"},
		{"hosted logo", "Acme", "https://cdn.example.com/logo.png", "Acme", "https://cdn.example.com/logo.png", "https://cdn.example.com/logo.png"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range map[string]string{"app.root_url": "https://mail.example.com/", "app.site_name": tc.title, "app.logo_url": tc.logo, "upload.provider": "fs"} {
				if err := ko.Set(k, v); err != nil {
					t.Fatal(err)
				}
			}
			c := initConstants()
			if c.SiteName != tc.wantTitle || c.LogoURL != tc.wantLogo || c.FaviconURL != tc.wantFavicon {
				t.Fatalf("got title %q logo %q favicon %q", c.SiteName, c.LogoURL, c.FaviconURL)
			}
		})
	}
}

// The rejections run before any storage access, so they need no database.
func TestUploadSiteLogoRejectsUnsafeFiles(t *testing.T) {
	lo := logf.New(logf.Opts{})
	app := &App{i18n: testutil.NewI18n(t), lo: &lo}
	for _, tc := range []struct {
		name    string
		content []byte
		want    int
	}{
		{"svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`), fasthttp.StatusBadRequest},
		{"empty", nil, fasthttp.StatusBadRequest},
		{"oversized png", append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, maxSiteLogoBytes)...), fasthttp.StatusRequestEntityTooLarge},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var body bytes.Buffer
			w := multipart.NewWriter(&body)
			part, err := w.CreateFormFile("files", "logo")
			if err != nil {
				t.Fatal(err)
			}
			part.Write(tc.content)
			w.Close()

			ctx := &fasthttp.RequestCtx{}
			ctx.Request.Header.SetMethod(fasthttp.MethodPost)
			ctx.Request.Header.SetContentType(w.FormDataContentType())
			ctx.Request.SetBody(body.Bytes())
			if err := handleUploadSiteLogo(&fastglue.Request{RequestCtx: ctx, Context: app}); err != nil {
				t.Fatal(err)
			}
			if got := ctx.Response.StatusCode(); got != tc.want {
				t.Fatalf("status = %d, want %d: %s", got, tc.want, ctx.Response.Body())
			}
		})
	}
}
