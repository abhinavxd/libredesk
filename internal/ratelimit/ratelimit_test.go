package ratelimit

import (
	"bytes"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/logf"
)

func TestUnavailableLimiterLogsFailOpen(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr(), MaxRetries: -1})
	t.Cleanup(func() { client.Close() })
	var logs bytes.Buffer
	lo := logf.New(logf.Opts{Writer: &logs})
	limiter := New(client, &lo, "X-Client-IP")
	limiter.AddRule(Rule{Name: "auth", Enabled: true, RequestsPerMinute: 1})
	var request fasthttp.RequestCtx
	request.Init(&fasthttp.Request{}, nil /** remoteAddr **/, nil /** logger **/)
	if err := limiter.Check(&request, "auth"); err != nil {
		t.Fatal(err)
	}
	if logs.Len() != 0 {
		t.Fatal("allowed request produced a log")
	}
	if err := limiter.Check(&request, "auth"); err == nil {
		t.Fatal("rate limit was not enforced")
	}
	server.SetError("storage unavailable")
	if err := limiter.Check(&request, "auth"); err != nil {
		t.Fatalf("existing fail open policy changed: %v", err)
	}
	if !strings.Contains(logs.String(), "level=warn") || !strings.Contains(logs.String(), "rule=auth") {
		t.Fatalf("missing storage failure warning: %s", logs.String())
	}
}

func TestLimiterUsesConfiguredClientIPHeader(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { client.Close() })
	var logs bytes.Buffer
	lo := logf.New(logf.Opts{Writer: &logs})
	limiter := New(client, &lo, "X-Real-IP")
	limiter.AddRule(Rule{Name: "auth", Enabled: true, RequestsPerMinute: 1})
	var request fasthttp.RequestCtx
	request.Init(&fasthttp.Request{}, nil /** remoteAddr **/, nil /** logger **/)
	request.Request.Header.Set("X-Real-IP", "203.0.113.1")
	request.Request.Header.Set("X-Client-IP", "198.51.100.1")
	if err := limiter.Check(&request, "auth"); err != nil {
		t.Fatal(err)
	}
	request.Request.Header.Set("X-Client-IP", "198.51.100.2")
	request.Request.Header.Set("X-Forwarded-For", "198.51.100.3")
	if err := limiter.Check(&request, "auth"); err == nil {
		t.Fatal("changing unselected headers bypassed the limit")
	}
	request.Request.Header.Set("X-Real-IP", "203.0.113.2")
	if err := limiter.Check(&request, "auth"); err != nil {
		t.Fatalf("different client shared the limit: %v", err)
	}
}
