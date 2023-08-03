package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	assert2 "github.com/stretchr/testify/assert"
)

func TestLoggerMiddleware(t *testing.T) {
	assert := assert2.New(t)

	t.Run("on", func(t *testing.T) {
		current := os.Getenv("DISABLE_LOGGER")
		defer func() {
			_ = os.Setenv("DISABLE_LOGGER", current)
		}()
		_ = os.Setenv("DISABLE_LOGGER", "false")

		// Capture slog output
		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		oldLogger := slog.Default()
		slog.SetDefault(logger)
		defer slog.SetDefault(oldLogger)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Res", "OK")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Hallo, welt!"))
		})

		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer 123")
		req.Header.Set("X-Test", "test")

		f := LoggerMiddleware(handler)
		f.ServeHTTP(w, req)

		assert.Equal("Hallo, welt!", string(w.buf))
		assert.Equal("application/json", w.Header().Get("Content-Type"))
		assert.Equal("OK", w.Header().Get("X-Res"))

		// Verify slog was called
		logOutput := buf.String()
		assert.True(strings.Contains(logOutput, "Incoming HTTP request"), "Expected log output to contain 'incoming HTTP request'")
		assert.True(strings.Contains(logOutput, "GET"), "Expected log output to contain method 'GET'")
	})

	t.Run("off", func(t *testing.T) {
		current := os.Getenv("DISABLE_LOGGER")
		defer func() {
			_ = os.Setenv("DISABLE_LOGGER", current)
		}()
		_ = os.Setenv("DISABLE_LOGGER", "true")

		// Capture slog output
		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		oldLogger := slog.Default()
		slog.SetDefault(logger)
		defer slog.SetDefault(oldLogger)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Res", "OK")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Hallo, welt!"))
		})

		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer 123")
		req.Header.Set("X-Test", "test")

		f := LoggerMiddleware(handler)
		f.ServeHTTP(w, req)

		assert.Equal("Hallo, welt!", string(w.buf))
		assert.Equal("application/json", w.Header().Get("Content-Type"))
		assert.Equal("OK", w.Header().Get("X-Res"))

		// Verify slog was NOT called
		logOutput := buf.String()
		assert.Equal("", logOutput, "Expected no log output when logger is disabled")
	})
}
