package middleware

import (
	"fmt"
	"net/http"
	"time"
)

// DurationMiddleware adds X-Cxs-Duration header with request duration in milliseconds.
func DurationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		w.Header().Set("X-Cxs-Duration", fmt.Sprintf("%.3fms", float64(time.Since(start).Microseconds())/1000))
	})
}
