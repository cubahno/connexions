package connexions

import (
	"github.com/go-chi/chi/v5/middleware"
	"net/http"
	"strings"
)

func ConditionalLoggingMiddleware(cfg *Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		logger := middleware.DefaultLogger(next)

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, cfg.App.HomeURL) {
				next.ServeHTTP(w, r)
				return
			}
			logger.ServeHTTP(w, r)
		})
	}
}
