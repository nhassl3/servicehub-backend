package http

import (
	"net/http"
	"net/http/pprof"
)

// NewPprofServer creates an HTTP server exposing the standard library pprof
// endpoints bound to 127.0.0.1, so profiling data never leaves the host.
func NewPprofServer(port string) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	return &http.Server{
		Addr:    "127.0.0.1" + port,
		Handler: mux,
	}
}