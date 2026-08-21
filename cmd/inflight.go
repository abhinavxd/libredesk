package main

import (
	"sync"
	"time"

	"github.com/abhinavxd/libredesk/internal/colorlog"
	"github.com/valyala/fasthttp"
)

var inFlight sync.Map

type inFlightRequest struct {
	method string
	path   string
	start  time.Time
}

func trackInFlight(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		id := ctx.ID()
		inFlight.Store(id, inFlightRequest{
			method: string(ctx.Method()),
			path:   string(ctx.Path()),
			start:  time.Now(),
		})
		defer inFlight.Delete(id)
		next(ctx)
	}
}

func logInFlight(when string) {
	var n int
	inFlight.Range(func(_, v any) bool {
		r := v.(inFlightRequest)
		colorlog.Red("in-flight %s: %s %s age=%s", when, r.method, r.path, time.Since(r.start).Round(time.Millisecond))
		n++
		return true
	})
	colorlog.Red("in-flight %s: %d request(s).", when, n)
}
