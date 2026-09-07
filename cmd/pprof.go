package main

import (
	"net"
	"net/http"
	"net/http/pprof"
	"runtime"

	"github.com/abhinavxd/libredesk/internal/colorlog"
)

// startPprof starts the net/http/pprof server on its own address if enabled in config.
func startPprof() {
	if !ko.Bool("app.pprof.enabled") {
		return
	}

	addr := ko.String("app.pprof.address")
	if addr == "" {
		addr = "127.0.0.1:6060"
	}

	if rate := ko.Int("app.pprof.block_profile_rate"); rate > 0 {
		runtime.SetBlockProfileRate(rate)
	}
	if frac := ko.Int("app.pprof.mutex_profile_fraction"); frac > 0 {
		runtime.SetMutexProfileFraction(frac)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		colorlog.Red("error starting pprof server: %v", err)
		return
	}
	colorlog.Green("pprof server started at %s", addr)

	go func() {
		if err := http.Serve(ln, mux); err != nil {
			colorlog.Red("pprof server stopped: %v", err)
		}
	}()
}
