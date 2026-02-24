package middleware

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

// skipPaths are path segments that should skip logging.
var skipPaths = []string{"/assets/", "/static/", "/favicon"}

// skipExtensions are file extensions that should skip logging.
var skipExtensions = []string{
	".js", ".css", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico",
	".woff", ".woff2", ".ttf", ".eot", ".map",
}

func shouldSkipLogging(urlPath string) bool {
	// Check path segments
	for _, p := range skipPaths {
		if strings.Contains(urlPath, p) {
			return true
		}
	}

	// Check file extensions
	ext := strings.ToLower(path.Ext(urlPath))
	for _, e := range skipExtensions {
		if ext == e {
			return true
		}
	}

	return false
}

// LoggerMiddleware is a custom logging middleware
func LoggerMiddleware(next http.Handler) http.Handler {
	disableLogger := os.Getenv("DISABLE_LOGGER") == "true"

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if disableLogger {
			next.ServeHTTP(w, r)
			return
		}

		// Skip logging for static assets and files
		if shouldSkipLogging(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Read request body
		var requestBody []byte
		if r.Body != nil {
			requestBody, _ = io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		start := time.Now()

		next.ServeHTTP(w, r)

		duration := time.Since(start)

		headers := make(map[string]string)
		for name, values := range w.Header() {
			headers[name] = strings.Join(values, ",")
		}

		slog.Info(fmt.Sprintf("Incoming HTTP request: %s", r.URL.String()),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("duration", duration.String()),
			slog.String("body", string(requestBody)),
		)
	})
}
