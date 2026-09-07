package main

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp"
	goredis "github.com/zerodha/fastcache/stores/goredis/v9"
	"github.com/zerodha/fastcache/v4"
	"github.com/zerodha/fastglue"
)

func TestHelpCenterPageCacheServesAndInvalidates(t *testing.T) {
	fc := newTestFastCache(t)

	renders := 0
	body := "<h1>original</h1>"
	handler := func(r *fastglue.Request) error {
		renders++
		r.RequestCtx.SetContentType("text/html; charset=utf-8")
		r.RequestCtx.Response.Header.Set("Cache-Control", helpCenterCacheControl)
		r.RequestCtx.Response.SetBodyString(body)
		return nil
	}
	cached := fc.Cached(handler, helpCenterCacheOpts, helpCenterCacheGroup)

	const uri = "/hc/support/en/articles/refunds"

	first := request(uri, "")
	if err := cached(first); err != nil {
		t.Fatalf("first request: %v", err)
	}
	if renders != 1 {
		t.Fatalf("first request should render, renders = %d", renders)
	}
	etag := string(first.RequestCtx.Response.Header.Peek("ETag"))
	if etag == "" {
		t.Fatal("expected an ETag on the rendered page")
	}

	second := request(uri, "")
	if err := cached(second); err != nil {
		t.Fatalf("second request: %v", err)
	}
	if renders != 1 {
		t.Errorf("second request re-rendered instead of using the cache, renders = %d", renders)
	}
	if got := string(second.RequestCtx.Response.Body()); got != body {
		t.Errorf("cached body = %q, want %q", got, body)
	}

	revalidate := request(uri, etag)
	if err := cached(revalidate); err != nil {
		t.Fatalf("revalidation: %v", err)
	}
	if got := revalidate.RequestCtx.Response.StatusCode(); got != fasthttp.StatusNotModified {
		t.Errorf("matching ETag returned %d, want 304", got)
	}
	if got := revalidate.RequestCtx.Response.Body(); len(got) != 0 {
		t.Errorf("304 carried a body of %d bytes", len(got))
	}

	if err := fc.DelGroup(helpCenterCacheNamespace, helpCenterCacheGroup); err != nil {
		t.Fatalf("clearing the group: %v", err)
	}
	body = "<h1>edited</h1>"
	afterEdit := request(uri, "")
	if err := cached(afterEdit); err != nil {
		t.Fatalf("request after edit: %v", err)
	}
	if renders != 2 {
		t.Errorf("cleared cache did not re-render, renders = %d", renders)
	}
	if got := string(afterEdit.RequestCtx.Response.Body()); got != body {
		t.Errorf("stale body served after an edit: got %q, want %q", got, body)
	}
	if newETag := string(afterEdit.RequestCtx.Response.Header.Peek("ETag")); newETag == etag {
		t.Error("ETag survived an edit, readers holding the old copy would never refetch")
	}
}

func TestHelpCenterPageCacheKeysPerQueryAndSurvivesRename(t *testing.T) {
	fc := newTestFastCache(t)

	served := map[string]int{}
	handler := func(r *fastglue.Request) error {
		uri := string(r.RequestCtx.URI().RequestURI())
		served[uri]++
		r.RequestCtx.Response.SetBodyString(uri)
		return nil
	}
	cached := fc.Cached(handler, helpCenterCacheOpts, helpCenterCacheGroup)

	for _, uri := range []string{
		"/hc/support/en/search?q=refund",
		"/hc/support/en/search?q=billing",
		"/hc/docs/en/search?q=refund",
	} {
		if err := cached(request(uri, "")); err != nil {
			t.Fatalf("request: %v", err)
		}
	}
	if len(served) != 3 {
		t.Fatalf("expected 3 distinct cache entries, got %d: %v", len(served), served)
	}

	clearAll(t, fc)
	for uri, before := range served {
		if err := cached(request(uri, "")); err != nil {
			t.Fatalf("request: %v", err)
		}
		if served[uri] == before {
			t.Errorf("%s still served from cache after a clear", uri)
		}
	}
}

func TestHelpCenterPageCacheReportsHitOrMiss(t *testing.T) {
	app := &App{fc: newTestFastCache(t)}
	handler := cachedHCPage(func(r *fastglue.Request) error {
		r.RequestCtx.Response.SetBodyString("<h1>page</h1>")
		return nil
	})

	const uri = "/hc/support/en/articles/refunds"
	run := func(ifNoneMatch string) *fasthttp.RequestCtx {
		req := request(uri, ifNoneMatch)
		req.Context = app
		if err := handler(req); err != nil {
			t.Fatalf("request: %v", err)
		}
		return req.RequestCtx
	}

	first := run("")
	if got := string(first.Response.Header.Peek("X-Cache")); got != "MISS" {
		t.Errorf("first request X-Cache = %q, want MISS", got)
	}

	second := run("")
	if got := string(second.Response.Header.Peek("X-Cache")); got != "HIT" {
		t.Errorf("second request X-Cache = %q, want HIT", got)
	}

	revalidated := run(string(first.Response.Header.Peek("ETag")))
	if got := revalidated.Response.StatusCode(); got != fasthttp.StatusNotModified {
		t.Fatalf("revalidation returned %d, want 304", got)
	}
	if got := string(revalidated.Response.Header.Peek("X-Cache")); got != "HIT" {
		t.Errorf("304 X-Cache = %q, want HIT", got)
	}
}

func TestHelpCenterPageCacheKeepsNoIndexOnHits(t *testing.T) {
	app := &App{fc: newTestFastCache(t)}
	body := func(r *fastglue.Request) error {
		r.RequestCtx.Response.SetBodyString("raw markdown")
		return nil
	}

	cases := []struct {
		name    string
		handler fastglue.FastRequestHandler
		uri     string
		slug    string
	}{
		{"search results", cachedHCNoIndexPage(body), "/hc/support/en/search?q=refund", ""},
		{"markdown article", cachedHCPage(body), "/hc/support/en/articles/refunds.md", "refunds.md"},
		{"html article", cachedHCPage(body), "/hc/support/en/articles/refunds", "refunds"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			want := tc.slug != "refunds"
			for _, pass := range []string{"miss", "hit"} {
				req := request(tc.uri, "")
				req.Context = app
				if tc.slug != "" {
					req.RequestCtx.SetUserValue("article_slug", tc.slug)
				}
				if err := tc.handler(req); err != nil {
					t.Fatalf("%s: %v", pass, err)
				}
				got := string(req.RequestCtx.Response.Header.Peek("X-Robots-Tag")) == noIndexHeader
				if got != want {
					t.Errorf("%s: noindex = %v, want %v", pass, got, want)
				}
			}
		})
	}
}

func newTestFastCache(t *testing.T) *fastcache.FastCache {
	t.Helper()
	mr := miniredis.RunT(t)
	return fastcache.New(goredis.New(goredis.Config{Prefix: fastCachePrefix}, redis.NewClient(&redis.Options{Addr: mr.Addr()})))
}

func request(uri, ifNoneMatch string) *fastglue.Request {
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI(uri)
	ctx.SetUserValue(helpCenterCacheNamespaceKey, helpCenterCacheNamespace)
	if ifNoneMatch != "" {
		ctx.Request.Header.Set("If-None-Match", ifNoneMatch)
	}
	return &fastglue.Request{RequestCtx: ctx}
}

func clearAll(t *testing.T, fc *fastcache.FastCache) {
	t.Helper()
	if err := fc.DelGroup(helpCenterCacheNamespace, helpCenterCacheGroup); err != nil {
		t.Fatalf("clearing the group: %v", err)
	}
}
