package connexions

import (
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
		defer os.Setenv("DISABLE_LOGGER", current)
		os.Setenv("DISABLE_LOGGER", "false")
		cfg := &Config{
			App: NewDefaultAppConfig(t.TempDir()),
		}

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hallo, welt!"))
		})

		w := newBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		f := ConditionalLoggingMiddleware(cfg)
		f(handler).ServeHTTP(w, req)

		assert.Equal("Hallo, welt!", string(w.buf))
	})
}
