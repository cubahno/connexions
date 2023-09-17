//go:build !integration

package connexions

import (
	assert2 "github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIResponse(t *testing.T) {
	assert := assert2.New(t)

	t.Run("happy-path", func(t *testing.T) {
		w := httptest.NewRecorder()
		resp := NewAPIResponse(w)

		resp.
			WithStatusCode(http.StatusOK).
			WithStatusCode(http.StatusCreated).
			WithHeader("key-1", "value-1").
			WithHeader("key-2", "value-2").
			Send([]byte("test"))

		assert.Equal(http.StatusCreated, w.Code)
		assert.Equal("value-1", w.Header().Get("key-1"))
		assert.Equal("value-2", w.Header().Get("key-2"))
		assert.Equal("test", w.Body.String())
	})

	t.Run("no-status-code", func(t *testing.T) {
		w := httptest.NewRecorder()
		resp := NewAPIResponse(w)

		resp.Send([]byte("test"))
		assert.Equal(http.StatusOK, w.Code)
		assert.Equal("test", w.Body.String())
	})

	t.Run("nil-case", func(t *testing.T) {
		w := httptest.NewRecorder()
		resp := NewAPIResponse(w)

		resp.Send(nil)
		assert.Equal("", w.Body.String())
	})
}

func TestJSONResponse(t *testing.T) {
	assert := assert2.New(t)

	t.Run("happy-path", func(t *testing.T) {
		w := httptest.NewRecorder()
		resp := NewJSONResponse(w)

		resp.
			WithStatusCode(http.StatusOK).
			WithStatusCode(http.StatusCreated).
			WithHeader("key-1", "value-1").
			WithHeader("key-2", "value-2").
			Send(map[string]string{"nice": "rice"})

		assert.Equal(http.StatusCreated, w.Code)
		assert.Equal("value-1", w.Header().Get("key-1"))
		assert.Equal("value-2", w.Header().Get("key-2"))
		assert.Equal("application/json", w.Header().Get("Content-Type"))
		assert.Equal(`{"nice":"rice"}`, w.Body.String())
	})

	t.Run("no-status-code", func(t *testing.T) {
		w := httptest.NewRecorder()
		resp := NewJSONResponse(w)

		resp.Send(map[string]string{"nice": "rice"})
		assert.Equal(http.StatusOK, w.Code)
		assert.Equal("application/json", w.Header().Get("Content-Type"))
		assert.Equal(`{"nice":"rice"}`, w.Body.String())
	})

	t.Run("nil-case", func(t *testing.T) {
		w := httptest.NewRecorder()
		resp := NewJSONResponse(w)

		resp.Send(nil)
		assert.Equal("application/json", w.Header().Get("Content-Type"))
		assert.Equal("", w.Body.String())
	})

	t.Run("marshall-error", func(t *testing.T) {
		w := httptest.NewRecorder()
		resp := NewJSONResponse(w)

		resp.Send(func() {})
		assert.Equal("application/json", w.Header().Get("Content-Type"))
		assert.Equal(http.StatusInternalServerError, w.Code)
		assert.Equal("json: unsupported type: func()", w.Body.String())
	})
}
