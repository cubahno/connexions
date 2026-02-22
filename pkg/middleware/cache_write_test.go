package middleware

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/cubahno/connexions/v2/pkg/config"
	"github.com/cubahno/connexions/v2/pkg/db"
	assert2 "github.com/stretchr/testify/assert"
)

func TestResponseWriter(t *testing.T) {
	assert := assert2.New(t)

	t.Run("buffers body without writing to underlying", func(t *testing.T) {
		underlying := httptest.NewRecorder()
		rw := &responseWriter{
			ResponseWriter: underlying,
			body:           new(bytes.Buffer),
			statusCode:     http.StatusOK,
		}

		_, _ = rw.Write([]byte("test body"))

		// Body should be in buffer, not in underlying
		assert.Equal("test body", rw.body.String())
		assert.Empty(underlying.Body.String())
	})

	t.Run("captures status code without sending to underlying", func(t *testing.T) {
		underlying := httptest.NewRecorder()
		rw := &responseWriter{
			ResponseWriter: underlying,
			body:           new(bytes.Buffer),
			statusCode:     http.StatusOK,
		}

		rw.WriteHeader(http.StatusCreated)

		// Status captured but not sent
		assert.Equal(http.StatusCreated, rw.statusCode)
		assert.Equal(http.StatusOK, underlying.Code) // httptest.Recorder defaults to 200
	})

	t.Run("delegates Header to underlying", func(t *testing.T) {
		underlying := httptest.NewRecorder()
		rw := &responseWriter{
			ResponseWriter: underlying,
			body:           new(bytes.Buffer),
			statusCode:     http.StatusOK,
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.Header().Set("X-Custom", "value")

		// Headers should be on underlying
		assert.Equal("application/json", underlying.Header().Get("Content-Type"))
		assert.Equal("value", underlying.Header().Get("X-Custom"))
	})
}

func TestCreateCacheWriteMiddleware(t *testing.T) {
	assert := assert2.New(t)

	t.Run("on", func(t *testing.T) {
		params := newTestParams(&config.ServiceConfig{
			Name: "foo",
			Cache: &config.CacheConfig{
				Requests: true,
			},
		}, nil)

		resp := &db.HistoryResponse{
			Data:       []byte("cached"),
			StatusCode: http.StatusOK,
		}
		histReq := &http.Request{
			URL:    &url.URL{Path: "/foo/bar"},
			Method: http.MethodGet,
		}
		params.DB().History().Set(context.Background(), "/foo/bar", histReq, resp)

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

			rec, exists := params.DB().History().Get(context.Background(), req)
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

			rec, exists := params.DB().History().Get(context.Background(), req)
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

		resp := &db.HistoryResponse{
			Data:        []byte("cached"),
			StatusCode:  http.StatusOK,
			ContentType: "text/plain",
		}
		histReq := &http.Request{
			URL:    &url.URL{Path: "/foo/bar"},
			Method: http.MethodGet,
		}
		params.DB().History().Set(context.Background(), "/foo/bar", histReq, resp)

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

		rec, exists := params.DB().History().Get(context.Background(), req)
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
		rec, exists := params.DB().History().Get(context.Background(), req1)
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

	t.Run("sets custom response headers", func(t *testing.T) {
		params := newTestParams(&config.ServiceConfig{
			Name: "test-service",
		}, nil)

		mw := CreateCacheWriteMiddleware(params)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id": 1}`))
		})

		// Add start time to context for duration header
		req := httptest.NewRequest(http.MethodPost, "/api/items", nil)
		ctx := context.WithValue(req.Context(), startTimeKey, time.Now().Add(-50*time.Millisecond))
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		mw(handler).ServeHTTP(w, req)

		// Verify custom headers are set
		assert.Equal(ResponseHeaderSourceGenerated, w.Header().Get(ResponseHeaderSource))
		assert.NotEmpty(w.Header().Get("X-Cxs-Duration"), "Duration header should be set")
		assert.Equal("application/json", w.Header().Get("Content-Type"))
		assert.Equal(http.StatusCreated, w.Code)
		assert.Equal(`{"id": 1}`, w.Body.String())
	})

	t.Run("handler WriteHeader does not send headers immediately", func(t *testing.T) {
		params := newTestParams(&config.ServiceConfig{
			Name: "test-service",
		}, nil)

		mw := CreateCacheWriteMiddleware(params)

		// Handler sets headers and writes - simulating generated service handler
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("X-Custom", "value")
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte("accepted"))
		})

		req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
		ctx := context.WithValue(req.Context(), startTimeKey, time.Now())
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		mw(handler).ServeHTTP(w, req)

		// Our headers should be present alongside handler's headers
		assert.Equal(ResponseHeaderSourceGenerated, w.Header().Get(ResponseHeaderSource))
		assert.Equal("text/plain", w.Header().Get("Content-Type"))
		assert.Equal("value", w.Header().Get("X-Custom"))
		assert.Equal(http.StatusAccepted, w.Code)
		assert.Equal("accepted", w.Body.String())
	})
}
