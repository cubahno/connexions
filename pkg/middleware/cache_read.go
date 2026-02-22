package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
)

// CreateCacheReadMiddleware returns a middleware that checks if GET request is cached in History.
func CreateCacheReadMiddleware(params *Params) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			cfg := params.ServiceConfig
			if cfg == nil || cfg.Cache == nil {
				next.ServeHTTP(w, req)
				return
			}

			// Check if it is GET request
			if req.Method != http.MethodGet || !cfg.Cache.Requests {
				next.ServeHTTP(w, req)
				return
			}

			res, exists := params.DB().History().Get(req.Context(), req)
			if !exists {
				next.ServeHTTP(w, req)
				return
			}

			slog.Info(fmt.Sprintf("Cache hit for %s", req.URL.Path))

			response := res.Response
			SetDurationHeader(w, req)
			w.Header().Set(ResponseHeaderSource, ResponseHeaderSourceCache)
			if response.ContentType != "" {
				w.Header().Set("Content-Type", response.ContentType)
			}
			w.WriteHeader(response.StatusCode)
			_, _ = w.Write(response.Data)
		})
	}
}
