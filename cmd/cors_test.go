package main

import (
	"testing"

	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/v2"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func newCORSTestGlue(t *testing.T, origins any) func(*fasthttp.RequestCtx) {
	t.Helper()
	k := koanf.New(".")
	if err := k.Load(confmap.Provider(map[string]any{"app.server.cors_allowed_origins": origins}, "."), nil); err != nil {
		t.Fatal(err)
	}
	g := fastglue.NewGlue()
	g.GET("/api/v1/ping", func(r *fastglue.Request) error { return r.SendEnvelope("pong") })
	initCORS(g, k)
	return g.Handler()
}

func doCORSRequest(h func(*fasthttp.RequestCtx), method, origin string) *fasthttp.RequestCtx {
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod(method)
	ctx.Request.SetRequestURI("/api/v1/ping")
	ctx.Request.Header.Set(fasthttp.HeaderOrigin, origin)
	h(ctx)
	return ctx
}

func TestCORS(t *testing.T) {
	for name, origins := range map[string]any{
		"toml list":  []any{"https://admin.example.com/"},
		"env string": "https://other.example.com, https://admin.example.com",
	} {
		h := newCORSTestGlue(t, origins)

		ctx := doCORSRequest(h, fasthttp.MethodGet, "https://admin.example.com")
		if got := string(ctx.Response.Header.Peek(fasthttp.HeaderAccessControlAllowOrigin)); got != "https://admin.example.com" {
			t.Errorf("%s: GET from allowed origin: Allow-Origin = %q", name, got)
		}
		if got := ctx.Response.Header.Peek(fasthttp.HeaderAccessControlAllowCredentials); len(got) != 0 {
			t.Errorf("%s: credentials must never be allowed, got %q", name, got)
		}

		ctx = doCORSRequest(h, fasthttp.MethodOptions, "https://admin.example.com")
		if ctx.Response.StatusCode() != fasthttp.StatusNoContent || len(ctx.Response.Header.Peek(fasthttp.HeaderAccessControlAllowHeaders)) == 0 {
			t.Errorf("%s: preflight from allowed origin: status %d, headers %s", name, ctx.Response.StatusCode(), ctx.Response.Header.String())
		}

		ctx = doCORSRequest(h, fasthttp.MethodGet, "https://evil.example.com")
		if got := ctx.Response.Header.Peek(fasthttp.HeaderAccessControlAllowOrigin); len(got) != 0 {
			t.Errorf("%s: GET from other origin should get no Allow-Origin, got %q", name, got)
		}
	}

	// Not configured: no CORS headers at all.
	ctx := doCORSRequest(newCORSTestGlue(t, ""), fasthttp.MethodGet, "https://admin.example.com")
	if got := ctx.Response.Header.Peek(fasthttp.HeaderAccessControlAllowOrigin); len(got) != 0 {
		t.Errorf("unconfigured: expected no Allow-Origin, got %q", got)
	}
}
