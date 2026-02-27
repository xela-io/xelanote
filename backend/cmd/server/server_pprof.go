package main

import (
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"time"
)

// Start pprof server (ONLY when explicitly enabled via PPROF_ENABLED=true).
// Security: Disabled by default, must be explicitly enabled even in dev/staging.
func startPprofServerIfEnabled() {
	if os.Getenv("PPROF_ENABLED") != "true" {
		return
	}

	go func() {
		// Dedicated ServeMux for security (prevents exposure of other handlers)
		pprofMux := http.NewServeMux()
		pprofMux.HandleFunc("/debug/pprof/", pprof.Index)
		pprofMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		pprofMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		pprofMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		pprofMux.HandleFunc("/debug/pprof/trace", pprof.Trace)
		pprofMux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
		pprofMux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
		pprofMux.Handle("/debug/pprof/block", pprof.Handler("block"))
		pprofMux.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))

		pprofAddr := "127.0.0.1:6060"
		log.Printf("pprof server available at http://%s/debug/pprof/", pprofAddr)
		log.Printf("  CPU Profile: http://%s/debug/pprof/profile?seconds=30", pprofAddr)
		log.Printf("  Heap Profile: http://%s/debug/pprof/heap", pprofAddr)
		log.Printf("  Goroutines: http://%s/debug/pprof/goroutine", pprofAddr)

		pprofServer := &http.Server{
			Addr:         pprofAddr,
			Handler:      pprofMux,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 60 * time.Second,
			IdleTimeout:  120 * time.Second,
		}
		if err := pprofServer.ListenAndServe(); err != nil {
			log.Printf("pprof server failed: %v", err)
		}
	}()
}
