package main

import (
	"bytes"
	"os"
	"testing"

	"github.com/knadh/stuffbin"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// Exercise the registered routes: testing only a file handler misses a missing route.
func TestFaviconRoutes(t *testing.T) {
	fs, err := stuffbin.NewLocalFS("/",
		"../frontend/public/favicon.svg:frontend/dist/main/favicon.svg",
		"../frontend/public/favicon.ico:frontend/dist/main/favicon.ico",
	)
	if err != nil {
		t.Fatal(err)
	}
	g := fastglue.New()
	g.SetContext(&App{fs: fs})
	initHandlers(g, nil)
	for _, tc := range []struct{ path, contentType string }{
		{"/favicon.svg", "image/svg+xml"},
		{"/favicon.ico", "image/x-icon"},
	} {
		want, err := os.ReadFile("../frontend/public" + tc.path)
		if err != nil {
			t.Fatal(err)
		}
		for _, method := range []string{"GET", "HEAD"} {
			t.Run(method+tc.path, func(t *testing.T) {
				ctx := &fasthttp.RequestCtx{}
				ctx.Request.SetRequestURI(tc.path)
				ctx.Request.Header.SetMethod(method)
				g.Handler()(ctx)
				if ctx.Response.StatusCode() != 200 {
					t.Fatalf("status = %d", ctx.Response.StatusCode())
				}
				if got := string(ctx.Response.Header.ContentType()); got != tc.contentType {
					t.Fatalf("content type = %q, want %q", got, tc.contentType)
				}
				if method == "GET" && !bytes.Equal(ctx.Response.Body(), want) {
					t.Fatal("response does not match the bundled favicon")
				}
			})
		}
	}
}
