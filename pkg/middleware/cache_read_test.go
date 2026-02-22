package middleware

import (
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

func TestCreateCacheReadMiddleware(t *testing.T) {
	assert := assert2.New(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("fresh"))
	})

	t.Run("nil config passes through", func(t *testing.T) {
		params := newTestParams(nil, nil)
		params.ServiceConfig = nil
		mw := CreateCacheReadMiddleware(params)
		assert.NotNil(mw)

		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/foo/bar", nil)

		mw(handler).ServeHTTP(w, req)

		assert.Equal("fresh", string(w.buf))
	})

	t.Run("nil cache config passes through", func(t *testing.T) {
		params := newTestParams(&config.ServiceConfig{
			Name:  "test",
			Cache: nil,
		}, nil)
		mw := CreateCacheReadMiddleware(params)
		assert.NotNil(mw)

		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/foo/bar", nil)

		mw(handler).ServeHTTP(w, req)

		assert.Equal("fresh", string(w.buf))
	})

	t.Run("on", func(t *testing.T) {
		params := newTestParams(&config.ServiceConfig{
			Name: "foo",
			Cache: &config.CacheConfig{
				Requests: true,
			},
		}, nil)

		resp := &db.HistoryResponse{
			Data:        []byte("cached"),
			StatusCode:  http.StatusOK,
			ContentType: "application/json",
		}
		histReq := &http.Request{
			URL:    &url.URL{Path: "/foo/bar"},
			Method: http.MethodGet,
		}
		params.DB().History().Set(context.Background(), "/foo/bar", histReq, resp)

		mw := CreateCacheReadMiddleware(params)
		assert.NotNil(mw)

		t.Run("not-get", func(t *testing.T) {
			w := NewBufferedResponseWriter()
			req := httptest.NewRequest(http.MethodPost, "/foo/bar", nil)

			mw(handler).ServeHTTP(w, req)

			assert.Equal("fresh", string(w.buf))
		})

		t.Run("get-no-cache", func(t *testing.T) {
			w := NewBufferedResponseWriter()
			req := httptest.NewRequest(http.MethodGet, "/foo/bar/new", nil)

			mw(handler).ServeHTTP(w, req)

			assert.Equal("fresh", string(w.buf))
		})

		t.Run("get-cache", func(t *testing.T) {
			w := NewBufferedResponseWriter()
			req := httptest.NewRequest(http.MethodGet, "/foo/bar", nil)

			mw(handler).ServeHTTP(w, req)

			assert.Equal("cached", string(w.buf))
		})
	})

	t.Run("off", func(t *testing.T) {
		params := newTestParams(&config.ServiceConfig{
			Name: "foo",
			Cache: &config.CacheConfig{
				Requests: false,
			},
		}, nil)

		resp := &db.HistoryResponse{
			Data:        []byte("cached"),
			StatusCode:  http.StatusOK,
			ContentType: "application/json",
		}
		histReq := &http.Request{
			URL:    &url.URL{Path: "/foo/bar"},
			Method: http.MethodGet,
		}
		params.DB().History().Set(context.Background(), "/foo/bar", histReq, resp)

		mw := CreateCacheReadMiddleware(params)
		assert.NotNil(mw)

		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/foo/bar", nil)

		mw(handler).ServeHTTP(w, req)

		assert.Equal("fresh", string(w.buf))
	})

	t.Run("restores content-type from cache", func(t *testing.T) {
		params := newTestParams(&config.ServiceConfig{
			Name: "service",
			Cache: &config.CacheConfig{
				Requests: true,
			},
		}, nil)

		resp := &db.HistoryResponse{
			Data:        []byte(`{"cached": true}`),
			StatusCode:  http.StatusOK,
			ContentType: "application/json",
		}
		histReq := &http.Request{
			URL:    &url.URL{Path: "/api/data"},
			Method: http.MethodGet,
		}
		params.DB().History().Set(context.Background(), "/api/data", histReq, resp)

		mw := CreateCacheReadMiddleware(params)

		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/api/data", nil)

		mw(handler).ServeHTTP(w, req)

		assert.Equal(`{"cached": true}`, string(w.buf))
		assert.Equal("application/json", w.header.Get("Content-Type"))
	})

	t.Run("sets custom response headers on cache hit", func(t *testing.T) {
		params := newTestParams(&config.ServiceConfig{
			Name: "service",
			Cache: &config.CacheConfig{
				Requests: true,
			},
		}, nil)

		resp := &db.HistoryResponse{
			Data:        []byte(`{"cached": true}`),
			StatusCode:  http.StatusOK,
			ContentType: "application/json",
		}
		histReq := &http.Request{
			URL:    &url.URL{Path: "/api/cached"},
			Method: http.MethodGet,
		}
		params.DB().History().Set(context.Background(), "/api/cached", histReq, resp)

		mw := CreateCacheReadMiddleware(params)

		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/api/cached", nil)

		mw(handler).ServeHTTP(w, req)

		assert.Equal(ResponseHeaderSourceCache, w.header.Get(ResponseHeaderSource))
	})

	t.Run("applies latency when configured", func(t *testing.T) {
		params := newTestParams(&config.ServiceConfig{
			Name:    "service",
			Latency: 10 * time.Millisecond,
			Cache: &config.CacheConfig{
				Requests: true,
			},
		}, nil)

		resp := &db.HistoryResponse{
			Data:        []byte("cached"),
			StatusCode:  http.StatusOK,
			ContentType: "application/json",
		}
		histReq := &http.Request{
			URL:    &url.URL{Path: "/api/latency"},
			Method: http.MethodGet,
		}
		params.DB().History().Set(context.Background(), "/api/latency", histReq, resp)

		mw := CreateCacheReadMiddleware(params)

		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/api/latency", nil)

		start := time.Now()
		mw(handler).ServeHTTP(w, req)
		elapsed := time.Since(start)

		assert.Equal("cached", string(w.buf))
		// Should have waited at least 10ms due to latency
		assert.GreaterOrEqual(elapsed.Milliseconds(), int64(10))
	})
}
