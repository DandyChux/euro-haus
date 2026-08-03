package middleware

import (
	"net/http"
	"strings"
)

// StaticCacheMiddleware adds long-lived cache headers to static assets
func StaticCacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Only cache static assets for a long time (1 year)
		// Avoid caching the root index.html so your app updates immediately
		if strings.HasPrefix(path, "/_app/") || // SvelteKit asset folder
			strings.HasSuffix(path, ".webp") ||
			strings.HasSuffix(path, ".css") ||
			strings.HasSuffix(path, ".js") ||
			strings.HasSuffix(path, ".woff2") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			// Short cache or no cache for HTML
			w.Header().Set("Cache-Control", "no-cache")
		}

		next.ServeHTTP(w, r)
	})
}
