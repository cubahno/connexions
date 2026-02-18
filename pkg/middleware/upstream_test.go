package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/cubahno/connexions/v2/internal/history"
	"github.com/cubahno/connexions/v2/pkg/config"
	assert2 "github.com/stretchr/testify/assert"
)

func TestCreateUpstreamRequestMiddleware(t *testing.T) {
	assert := assert2.New(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("Hello, from local!"))
	})

	t.Run("upstream service response is used if present", func(t *testing.T) {
		var receivedHeaders http.Header
		upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedHeaders = r.Header.Clone()
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"message": "Hello, from remote!"}`))
		}))
		defer upstreamServer.Close()

		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/foo", nil)
		req.Header.Set("Authorization", "Bearer 123")
		req.Header.Set("X-Test", "test")

		hst := history.NewCurrentRequestStorage(100 * time.Second)

		params := &Params{
			ServiceConfig: &config.ServiceConfig{
				Upstream: &config.UpstreamConfig{
					URL: upstreamServer.URL,
				},
			},
			History: hst,
		}

		f := CreateUpstreamRequestMiddleware(params)
		f(handler).ServeHTTP(w, req)

		assert.Equal(`{"message": "Hello, from remote!"}`, string(w.buf))

		// Check that headers were forwarded to upstream
		assert.Equal("Bearer 123", receivedHeaders.Get("Authorization"))
		assert.Equal("test", receivedHeaders.Get("X-Test"))
		assert.Equal("Connexions/2.0", receivedHeaders.Get("User-Agent"))

		// Check history
		data := hst.Data()
		assert.Equal(1, len(data))
		rec := data["GET:/foo"]
		assert.Equal(200, rec.Response.StatusCode)
		assert.Equal([]byte(`{"message": "Hello, from remote!"}`), rec.Response.Data)
	})

	t.Run("history is present", func(t *testing.T) {
		rcvdBody := ""
		upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rcvdBodyBts, _ := io.ReadAll(r.Body)
			rcvdBody = string(rcvdBodyBts)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"message": "Hello, from remote!"}`))
		}))
		defer upstreamServer.Close()

		w := NewBufferedResponseWriter()
		body := io.NopCloser(strings.NewReader(`{"foo": "bar"}`))
		req := httptest.NewRequest(http.MethodPost, "/foo", body)

		hst := history.NewCurrentRequestStorage(100 * time.Second)
		resp := &history.HistoryResponse{
			Data:           []byte("cached"),
			StatusCode:     http.StatusOK,
			ContentType:    "application/json",
			IsFromUpstream: true,
		}
		histReq := &http.Request{
			URL:    &url.URL{Path: "/foo"},
			Method: http.MethodPost,
			Body:   io.NopCloser(strings.NewReader(`{"bar": "car"}`)),
		}
		hst.Set("foo", "/foo", histReq, resp)

		params := &Params{
			ServiceConfig: &config.ServiceConfig{
				Upstream: &config.UpstreamConfig{
					URL: upstreamServer.URL,
				},
			},
			History: hst,
		}

		f := CreateUpstreamRequestMiddleware(params)
		f(handler).ServeHTTP(w, req)

		assert.Equal(`{"message": "Hello, from remote!"}`, string(w.buf))
		assert.Equal(`{"foo": "bar"}`, rcvdBody)

		// Check history
		data := hst.Data()
		assert.Equal(1, len(data))
		rec := data["POST:/foo"]
		assert.Equal(200, rec.Response.StatusCode)
		assert.Equal([]byte(`{"message": "Hello, from remote!"}`), rec.Response.Data)
	})

	t.Run("not called if url is empty", func(t *testing.T) {
		callCount := 0
		upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"message": "OK"}`))
		}))
		defer upstreamServer.Close()

		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/foo", nil)

		hst := history.NewCurrentRequestStorage(100 * time.Second)

		params := &Params{
			ServiceConfig: &config.ServiceConfig{
				Name:     "test",
				Upstream: &config.UpstreamConfig{},
			},
			History: hst,
		}

		f := CreateUpstreamRequestMiddleware(params)
		f(handler).ServeHTTP(w, req)

		assert.Equal("Hello, from local!", string(w.buf))
		assert.Equal(0, callCount)
	})

	t.Run("upstream service response fails", func(t *testing.T) {
		upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message": "Internal Server Error!"}`))
		}))

		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/foo", nil)
		hst := history.NewCurrentRequestStorage(100 * time.Second)

		params := &Params{
			ServiceConfig: &config.ServiceConfig{
				Upstream: &config.UpstreamConfig{
					URL: upstreamServer.URL,
				},
			},
			History: hst,
		}

		f := CreateUpstreamRequestMiddleware(params)
		f(handler).ServeHTTP(w, req)

		assert.Equal("Hello, from local!", string(w.buf))
	})

	t.Run("request create fails", func(t *testing.T) {
		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/foo", nil)
		hst := history.NewCurrentRequestStorage(100 * time.Second)

		params := &Params{
			ServiceConfig: &config.ServiceConfig{
				Upstream: &config.UpstreamConfig{
					URL: "ht tps://example.com",
				},
			},
			History: hst,
		}

		f := CreateUpstreamRequestMiddleware(params)
		f(handler).ServeHTTP(w, req)

		assert.Equal("Hello, from local!", string(w.buf))
	})

	t.Run("upstream service times out", func(t *testing.T) {
		upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(200 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"message": "OK"}`))
		}))

		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/foo", nil)
		hst := history.NewCurrentRequestStorage(100 * time.Second)

		params := &Params{
			ServiceConfig: &config.ServiceConfig{
				Upstream: &config.UpstreamConfig{
					URL: upstreamServer.URL,
					FailOn: &config.UpstreamFailOnConfig{
						TimeOut: 50 * time.Millisecond,
					},
				},
			},
			History: hst,
		}

		f := CreateUpstreamRequestMiddleware(params)
		f(handler).ServeHTTP(w, req)

		assert.Equal("Hello, from local!", string(w.buf))
	})

	t.Run("upstream service returns failOn status", func(t *testing.T) {
		upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusMovedPermanently)
			_, _ = w.Write([]byte(`{"message": "OK"}`))
		}))

		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/foo", nil)
		hst := history.NewCurrentRequestStorage(100 * time.Second)

		params := &Params{
			ServiceConfig: &config.ServiceConfig{
				Upstream: &config.UpstreamConfig{
					URL: upstreamServer.URL,
					FailOn: &config.UpstreamFailOnConfig{
						HTTPStatus: config.HttpStatusFailOnConfig{
							{Exact: http.StatusMovedPermanently},
						},
					},
				},
			},
			History: hst,
		}

		f := CreateUpstreamRequestMiddleware(params)
		f(handler).ServeHTTP(w, req)

		assert.Equal("Hello, from local!", string(w.buf))
	})

	t.Run("upstream service returns one of the failOn statuses", func(t *testing.T) {
		upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"message": "OK"}`))
		}))

		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/foo", nil)
		hst := history.NewCurrentRequestStorage(100 * time.Second)

		params := &Params{
			ServiceConfig: &config.ServiceConfig{
				Upstream: &config.UpstreamConfig{
					URL: upstreamServer.URL,
					FailOn: &config.UpstreamFailOnConfig{
						HTTPStatus: config.HttpStatusFailOnConfig{
							{Range: "200-201"},
						},
					},
				},
			},
			History: hst,
		}

		f := CreateUpstreamRequestMiddleware(params)
		f(handler).ServeHTTP(w, req)

		assert.Equal("Hello, from local!", string(w.buf))
	})

	t.Run("circuit breaker opened", func(t *testing.T) {
		callCount := 0
		upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message": "Internal Server Error!"}`))
		}))
		defer upstreamServer.Close()

		hst := history.NewCurrentRequestStorage(100 * time.Second)

		params := &Params{
			ServiceConfig: &config.ServiceConfig{
				Name: "test",
				Upstream: &config.UpstreamConfig{
					URL: upstreamServer.URL,
				},
			},
			History: hst,
		}

		// Create middleware ONCE
		middleware := CreateUpstreamRequestMiddleware(params)
		wrappedHandler := middleware(handler)

		// Make 3 requests to open the circuit breaker
		for i := 1; i <= 3; i++ {
			w := NewBufferedResponseWriter()
			req := httptest.NewRequest(http.MethodGet, "/test/foo", nil)
			wrappedHandler.ServeHTTP(w, req)

			// Should fall back to local handler
			assert.Equal("Hello, from local!", string(w.buf))
		}

		// Verify upstream was called 3 times
		assert.Equal(3, callCount)

		// 4th request should be blocked by open circuit breaker
		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/test/foo", nil)
		wrappedHandler.ServeHTTP(w, req)

		// Circuit breaker is open, should fall back to local handler
		assert.Equal("Hello, from local!", string(w.buf))

		// Verify upstream was NOT called on 4th request (still 3 calls)
		assert.Equal(3, callCount, "upstream should not be called when circuit breaker is open")
	})
}
