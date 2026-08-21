package main

import (
	"bytes"
	"runtime/pprof"
	"sync"
	"time"

	"github.com/abhinavxd/libredesk/internal/colorlog"
	"github.com/valyala/fasthttp"
)

const goroutineDumpAfter = 5

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

// watchShutdown reports open connections every second while the server drains, and dumps
// goroutine stacks once if it is still draining after goroutineDumpAfter seconds.
func watchShutdown(s *fasthttp.Server, done <-chan struct{}) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	var elapsed int
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			elapsed++
			colorlog.Red("shutdown: %ds elapsed, %d open connection(s), %d concurrent request(s)", elapsed, s.GetOpenConnectionsCount(), s.GetCurrentConcurrency())
			if elapsed == goroutineDumpAfter {
				var buf bytes.Buffer
				if err := pprof.Lookup("goroutine").WriteTo(&buf, 1); err != nil {
					colorlog.Red("error dumping goroutines: %v", err)
					continue
				}
				colorlog.Red("goroutine dump while draining:\n%s", buf.String())
			}
		}
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
