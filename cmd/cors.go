package main

import (
	"strings"

	"github.com/knadh/koanf/v2"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// initCORS lets browser apps on the configured origins call the API (with an API key in the
// Authorization header) by adding CORS headers to responses and answering preflight requests.
// Credentials (cookies) are never allowed cross-origin, so sessions stay same-origin only.
//
// Configure with `app.server.cors_allowed_origins` in config.toml (a list), or the
// LIBREDESK_APP__SERVER__CORS_ALLOWED_ORIGINS environment variable (comma separated).
// Empty (the default) disables CORS entirely.
func initCORS(g *fastglue.Fastglue, ko *koanf.Koanf) {
	origins := ko.Strings("app.server.cors_allowed_origins")
	if len(origins) == 0 {
		// Environment variables always come through as a plain string.
		origins = strings.Split(ko.String("app.server.cors_allowed_origins"), ",")
	}

	allowed := map[string]bool{}
	for _, o := range origins {
		if o = strings.TrimRight(strings.TrimSpace(o), "/"); o != "" {
			allowed[o] = true
		}
	}
	if len(allowed) == 0 {
		return
	}

	// setHeaders adds the CORS headers if the request comes from an allowed origin.
	setHeaders := func(ctx *fasthttp.RequestCtx) bool {
		origin := string(ctx.Request.Header.Peek(fasthttp.HeaderOrigin))
		ctx.Response.Header.Add(fasthttp.HeaderVary, fasthttp.HeaderOrigin)
		if !allowed[origin] {
			return false
		}
		ctx.Response.Header.Set(fasthttp.HeaderAccessControlAllowOrigin, origin)
		return true
	}

	// Actual requests.
	g.Before(func(r *fastglue.Request) *fastglue.Request {
		setHeaders(r.RequestCtx)
		return r
	})

	// Preflight requests (the router answers OPTIONS for every registered path with this handler).
	g.Router.GlobalOPTIONS = func(ctx *fasthttp.RequestCtx) {
		if setHeaders(ctx) {
			ctx.Response.Header.Set(fasthttp.HeaderAccessControlAllowMethods, "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			ctx.Response.Header.Set(fasthttp.HeaderAccessControlAllowHeaders, "Authorization, Content-Type")
			ctx.Response.Header.Set(fasthttp.HeaderAccessControlMaxAge, "86400")
		}
		ctx.SetStatusCode(fasthttp.StatusNoContent)
	}
}
