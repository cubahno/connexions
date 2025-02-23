package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	assert2 "github.com/stretchr/testify/assert"
)

func TestCreateHealthRoutes(t *testing.T) {
	t.Parallel()
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}
	_ = createHealthRoutes(router)

	t.Run("health", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		resp := w.Body.String()
		assert.Equal(http.StatusOK, w.Code)
		assert.Equal("OK", resp)
		assert.Equal("text/plain; charset=utf-8", w.Header().Get("Content-Type"))
	})
}
