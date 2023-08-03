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

func TestCreateCacheWriteMiddleware(t *testing.T) {
	assert := assert2.New(t)

	t.Run("on", func(t *testing.T) {
		hst := history.NewCurrentRequestStorage(100 * time.Second)
		resp := &history.HistoryResponse{
			Data:       []byte("cached"),
			StatusCode: http.StatusOK,
		}
		histReq := &http.Request{
			URL:    &url.URL{Path: "/foo/bar"},
			Method: http.MethodGet,
		}
		hst.Set("foo", "/foo/bar", histReq, resp)

		mw := CreateCacheWriteMiddleware(&Params{
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
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte("created"))
			})
			mw(handler).ServeHTTP(w, req)
			assert.Equal("created", string(w.buf))

			rec, exists := hst.Get(req)
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

			rec, exists := hst.Get(req)
			assert.True(exists)
			assert.Equal(http.StatusOK, rec.Response.StatusCode)
			assert.Equal([]byte("fresh"), rec.Response.Data)
		})
	})

	t.Run("history written regardless of cache config", func(t *testing.T) {
		hst := history.NewCurrentRequestStorage(100 * time.Second)
		resp := &history.HistoryResponse{
			Data:        []byte("cached"),
			StatusCode:  http.StatusOK,
			ContentType: "text/plain",
		}
		histReq := &http.Request{
			URL:    &url.URL{Path: "/foo/bar"},
			Method: http.MethodGet,
		}
		hst.Set("foo", "/foo/bar", histReq, resp)

		mw := CreateCacheWriteMiddleware(&Params{
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
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("fresh"))
		})
		mw(handler).ServeHTTP(w, req)
		assert.Equal("fresh", string(w.buf))

		rec, exists := hst.Get(req)
		assert.True(exists)
		assert.Equal(http.StatusOK, rec.Response.StatusCode)
		assert.Equal([]byte("fresh"), rec.Response.Data)
		assert.Equal("application/json", rec.Response.ContentType)
	})
}
