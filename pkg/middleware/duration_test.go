package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	assert2 "github.com/stretchr/testify/assert"
)

func TestDurationMiddleware(t *testing.T) {
	assert := assert2.New(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mw := DurationMiddleware(handler)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	mw.ServeHTTP(w, req)

	// Check that X-Cxs-Duration header is set
	duration := w.Header().Get("X-Cxs-Duration")
	assert.NotEmpty(duration)
	assert.True(strings.HasSuffix(duration, "ms"), "duration should end with 'ms'")
}
