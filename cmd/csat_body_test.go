package main

import (
	"testing"

	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// bodyRequest builds a request whose body (and therefore Content-Length) is n bytes.
func bodyRequest(n int) *fastglue.Request {
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.SetBody(make([]byte, n))
	return &fastglue.Request{RequestCtx: ctx}
}

func TestLimitBody(t *testing.T) {
	const max = 24 * 1024

	tests := []struct {
		name     string
		size     int
		wantCall bool
		wantCode int
	}{
		{"under limit passes", max - 1, true, fasthttp.StatusOK},
		{"at limit passes", max, true, fasthttp.StatusOK},
		{"over limit rejected", max + 1, false, fasthttp.StatusRequestEntityTooLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			next := func(r *fastglue.Request) error {
				called = true
				return r.SendEnvelope(true)
			}

			r := bodyRequest(tt.size)
			if err := limitBody(next, max)(r); err != nil {
				t.Fatalf("limitBody returned error: %v", err)
			}
			if called != tt.wantCall {
				t.Fatalf("handler called = %v, want %v", called, tt.wantCall)
			}
			if got := r.RequestCtx.Response.StatusCode(); got != tt.wantCode {
				t.Fatalf("status = %d, want %d", got, tt.wantCode)
			}
		})
	}
}
