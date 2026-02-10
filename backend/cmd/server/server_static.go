package main

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// Serve static files for SPA (if embedded).
func setupStaticFiles(router chi.Router) {
	// setCacheHeaders sets Cache-Control based on the URL path.
	// Vite-hashed files under /_app/immutable/ can be cached aggressively.
	// All other files must revalidate on each request.
	setCacheHeaders := func(w http.ResponseWriter, path string) {
		if strings.HasPrefix(path, "/_app/immutable/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			return
		}
		w.Header().Set("Cache-Control", "no-cache")
	}

	staticSub, err := fs.Sub(staticFS, "static")
	if err != nil {
		return
	}

	// Serve static files
	fileServer := http.FileServer(http.FS(staticSub))
	router.Handle("/*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the file
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}

		// Check if file exists
		if _, err := staticFS.Open("static" + path); err == nil {
			setCacheHeaders(w, path)
			fileServer.ServeHTTP(w, r)
			return
		}

		// Fall back to index.html for SPA routing
		setCacheHeaders(w, "/index.html")
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	}))
}
