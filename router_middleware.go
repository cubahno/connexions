package connexions

import (
	"github.com/go-chi/chi/v5/middleware"
	"net/http"
	"os"
	"strings"
)

// ConditionalLoggingMiddleware is a middleware that conditionally can disable logger.
// For example, in tests or when fetching static files.
func ConditionalLoggingMiddleware(cfg *Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		logger := middleware.DefaultLogger(next)
		disableLogger := os.Getenv("DISABLE_LOGGER") == "true"

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if disableLogger || strings.HasPrefix(r.URL.Path, cfg.App.HomeURL) {
				next.ServeHTTP(w, r)
				return
			}
			logger.ServeHTTP(w, r)
		})
	}
}
