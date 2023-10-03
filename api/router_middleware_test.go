package api

import (
	"github.com/cubahno/connexions/config"
	assert2 "github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestConditionalLoggingMiddleware(t *testing.T) {
	assert := assert2.New(t)

	t.Run("on", func(t *testing.T) {
		current := os.Getenv("DISABLE_LOGGER")
		defer func() {
			_ = os.Setenv("DISABLE_LOGGER", current)
		}()
		_ = os.Setenv("DISABLE_LOGGER", "false")
		cfg := &config.Config{
			App: config.NewDefaultAppConfig(t.TempDir()),
		}

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("Hallo, welt!"))
		})

		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		f := ConditionalLoggingMiddleware(cfg)
		f(handler).ServeHTTP(w, req)

		assert.Equal("Hallo, welt!", string(w.buf))
	})
}
