package middleware

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/cubahno/connexions/v2/internal/history"
	"github.com/cubahno/connexions/v2/pkg/config"
	assert2 "github.com/stretchr/testify/assert"
)

func TestCreateCacheReadMiddleware(t *testing.T) {
	assert := assert2.New(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("fresh"))
	})

	t.Run("nil config passes through", func(t *testing.T) {
		mw := CreateCacheReadMiddleware(&Params{
			ServiceConfig: nil,
		})
		assert.NotNil(mw)

		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/foo/bar", nil)

		mw(handler).ServeHTTP(w, req)

		assert.Equal("fresh", string(w.buf))
	})

	t.Run("nil cache config passes through", func(t *testing.T) {
		mw := CreateCacheReadMiddleware(&Params{
			ServiceConfig: &config.ServiceConfig{
				Cache: nil,
			},
		})
		assert.NotNil(mw)

		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/foo/bar", nil)

		mw(handler).ServeHTTP(w, req)

		assert.Equal("fresh", string(w.buf))
	})

	t.Run("on", func(t *testing.T) {
		hst := history.NewCurrentRequestStorage(100 * time.Second)
		resp := &history.HistoryResponse{
			Data:        []byte("cached"),
			StatusCode:  http.StatusOK,
			ContentType: "application/json",
		}
		histReq := &http.Request{
			URL:    &url.URL{Path: "/foo/bar"},
			Method: http.MethodGet,
		}
		hst.Set("foo", "/foo/bar", histReq, resp)

		mw := CreateCacheReadMiddleware(&Params{
			ServiceConfig: &config.ServiceConfig{
				Cache: &config.CacheConfig{
					Requests: true,
				},
			},
			History: hst,
		})
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
		hst := history.NewCurrentRequestStorage(100 * time.Second)
		resp := &history.HistoryResponse{
			Data:        []byte("cached"),
			StatusCode:  http.StatusOK,
			ContentType: "application/json",
		}
		histReq := &http.Request{
			URL:    &url.URL{Path: "/foo/bar"},
			Method: http.MethodGet,
		}
		hst.Set("foo", "/foo/bar", histReq, resp)

		mw := CreateCacheReadMiddleware(&Params{
			ServiceConfig: &config.ServiceConfig{
				Cache: &config.CacheConfig{
					Requests: false,
				},
			},
			History: hst,
		})
		assert.NotNil(mw)

		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/foo/bar", nil)

		mw(handler).ServeHTTP(w, req)

		assert.Equal("fresh", string(w.buf))
	})

	t.Run("restores content-type from cache", func(t *testing.T) {
		hst := history.NewCurrentRequestStorage(100 * time.Second)
		resp := &history.HistoryResponse{
			Data:        []byte(`{"cached": true}`),
			StatusCode:  http.StatusOK,
			ContentType: "application/json",
		}
		histReq := &http.Request{
			URL:    &url.URL{Path: "/api/data"},
			Method: http.MethodGet,
		}
		hst.Set("service", "/api/data", histReq, resp)

		mw := CreateCacheReadMiddleware(&Params{
			ServiceConfig: &config.ServiceConfig{
				Cache: &config.CacheConfig{
					Requests: true,
				},
			},
			History: hst,
		})

		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/api/data", nil)

		mw(handler).ServeHTTP(w, req)

		assert.Equal(`{"cached": true}`, string(w.buf))
		assert.Equal("application/json", w.header.Get("Content-Type"))
	})

	t.Run("applies latency when configured", func(t *testing.T) {
		hst := history.NewCurrentRequestStorage(100 * time.Second)
		resp := &history.HistoryResponse{
			Data:        []byte("cached"),
			StatusCode:  http.StatusOK,
			ContentType: "application/json",
		}
		histReq := &http.Request{
			URL:    &url.URL{Path: "/api/latency"},
			Method: http.MethodGet,
		}
		hst.Set("service", "/api/latency", histReq, resp)

		mw := CreateCacheReadMiddleware(&Params{
			ServiceConfig: &config.ServiceConfig{
				Latency: 10 * time.Millisecond,
				Cache: &config.CacheConfig{
					Requests: true,
				},
			},
			History: hst,
		})

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
