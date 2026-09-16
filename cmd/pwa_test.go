package main

import (
	"strings"
	"testing"

	"github.com/knadh/stuffbin"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func TestServePWAFiles(t *testing.T) {
	fs, err := stuffbin.NewLocalFS("/",
		"../frontend/public/manifest.webmanifest:frontend/dist/main/manifest.webmanifest",
		"../frontend/public/sw.js:frontend/dist/main/sw.js",
	)
	if err != nil {
		t.Fatalf("creating test filesystem: %v", err)
	}

	tests := []struct {
		name                 string
		handler              fastglue.FastRequestHandler
		contentType          string
		wantServiceWorkerHdr bool
	}{
		{"manifest", serveManifest, "application/manifest+json", false},
		{"service worker", serveServiceWorker, "application/javascript", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &fasthttp.RequestCtx{}
			req := &fastglue.Request{RequestCtx: ctx, Context: &App{fs: fs}}
			if err := tc.handler(req); err != nil {
				t.Fatalf("serving file: %v", err)
			}
			if got := string(ctx.Response.Header.ContentType()); got != tc.contentType {
				t.Errorf("content type = %q, want %q", got, tc.contentType)
			}
			if got := string(ctx.Response.Header.Peek("Cache-Control")); got != "no-cache" {
				t.Errorf("cache control = %q, want no-cache", got)
			}
			if tc.wantServiceWorkerHdr && string(ctx.Response.Header.Peek("Service-Worker-Allowed")) != "/" {
				t.Error("service worker scope header is missing")
			}
			if strings.TrimSpace(string(ctx.Response.Body())) == "" {
				t.Error("response body is empty")
			}
		})
	}
}
