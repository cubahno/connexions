package middleware

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/cubahno/connexions/v2/internal/db"
	"github.com/cubahno/connexions/v2/pkg/config"
	assert2 "github.com/stretchr/testify/assert"
)

func TestCreateCacheWriteMiddleware(t *testing.T) {
	assert := assert2.New(t)

	t.Run("on", func(t *testing.T) {
		params := newTestParams(&config.ServiceConfig{
			Name: "foo",
			Cache: &config.CacheConfig{
				Requests: true,
			},
		}, nil)

		resp := &db.Response{
			Data:       []byte("cached"),
			StatusCode: http.StatusOK,
		}
		histReq := &http.Request{
			URL:    &url.URL{Path: "/foo/bar"},
			Method: http.MethodGet,
		}
		params.DB().History().Set("/foo/bar", histReq, resp)

		mw := CreateCacheWriteMiddleware(params)
		assert.NotNil(mw)

		t.Run("not-get", func(t *testing.T) {
			w := NewBufferedResponseWriter()
			req := httptest.NewRequest(http.MethodPost, "/foo/bar", nil)
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte("created"))
			})
			mw(handler).ServeHTTP(w, req)
			assert.Equal("created", string(w.buf))

			rec, exists := params.DB().History().Get(req)
			assert.True(exists)
			assert.Equal(http.StatusCreated, rec.Response.StatusCode)
			assert.Equal([]byte("created"), rec.Response.Data)
		})

		t.Run("get", func(t *testing.T) {
			w := NewBufferedResponseWriter()
			req := httptest.NewRequest(http.MethodGet, "/foo/bar", nil)
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("fresh"))
			})
			mw(handler).ServeHTTP(w, req)
			assert.Equal("fresh", string(w.buf))

			rec, exists := params.DB().History().Get(req)
			assert.True(exists)
			assert.Equal(http.StatusOK, rec.Response.StatusCode)
			assert.Equal([]byte("fresh"), rec.Response.Data)
		})
	})

	t.Run("history written regardless of cache config", func(t *testing.T) {
		params := newTestParams(&config.ServiceConfig{
			Name: "foo",
			Cache: &config.CacheConfig{
				Requests: false,
			},
		}, nil)

		resp := &db.Response{
			Data:        []byte("cached"),
			StatusCode:  http.StatusOK,
			ContentType: "text/plain",
		}
		histReq := &http.Request{
			URL:    &url.URL{Path: "/foo/bar"},
			Method: http.MethodGet,
		}
		params.DB().History().Set("/foo/bar", histReq, resp)

		mw := CreateCacheWriteMiddleware(params)
		assert.NotNil(mw)

		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/foo/bar", nil)
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("fresh"))
		})
		mw(handler).ServeHTTP(w, req)
		assert.Equal("fresh", string(w.buf))

		rec, exists := params.DB().History().Get(req)
		assert.True(exists)
		assert.Equal(http.StatusOK, rec.Response.StatusCode)
		assert.Equal([]byte("fresh"), rec.Response.Data)
		assert.Equal("application/json", rec.Response.ContentType)
	})

	t.Run("full cache flow: write then read", func(t *testing.T) {
		params := newTestParams(&config.ServiceConfig{
			Name: "test-service",
			Cache: &config.CacheConfig{
				Requests: true,
			},
		}, nil)

		// First request: cache_write captures the response
		writeMw := CreateCacheWriteMiddleware(params)
		readMw := CreateCacheReadMiddleware(params)

		// Handler that generates a response
		generateHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"generated": true}`))
		})

		// First request - should go through handler and cache the response
		w1 := NewBufferedResponseWriter()
		req1 := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		writeMw(generateHandler).ServeHTTP(w1, req1)
		assert.Equal(`{"generated": true}`, string(w1.buf))

		// Verify response was cached
		rec, exists := params.DB().History().Get(req1)
		assert.True(exists)
		assert.NotNil(rec.Response, "Response should be set after cache_write")
		assert.Equal(http.StatusOK, rec.Response.StatusCode)
		assert.Equal([]byte(`{"generated": true}`), rec.Response.Data)

		// Second request - should be served from cache by cache_read
		handlerCalled := false
		neverCalledHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			_, _ = w.Write([]byte("should not be called"))
		})

		w2 := NewBufferedResponseWriter()
		req2 := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		readMw(neverCalledHandler).ServeHTTP(w2, req2)

		assert.False(handlerCalled, "Handler should not be called when serving from cache")
		assert.Equal(`{"generated": true}`, string(w2.buf))
	})
}
