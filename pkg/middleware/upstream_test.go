package middleware

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/cubahno/connexions/v2/pkg/config"
	"github.com/cubahno/connexions/v2/pkg/db"
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
		req := httptest.NewRequest(http.MethodGet, "/test/foo", nil)
		req.Header.Set("Authorization", "Bearer 123")
		req.Header.Set("X-Test", "test")

		params := newTestParams(&config.ServiceConfig{
			Name: "test",
			Upstream: &config.UpstreamConfig{
				URL: upstreamServer.URL,
			},
		}, nil)

		f := CreateUpstreamRequestMiddleware(params)
		f(handler).ServeHTTP(w, req)

		assert.Equal(`{"message": "Hello, from remote!"}`, string(w.buf))

		// Check that headers were forwarded to upstream
		assert.Equal("Bearer 123", receivedHeaders.Get("Authorization"))
		assert.Equal("test", receivedHeaders.Get("X-Test"))
		assert.Equal("Connexions/2.0", receivedHeaders.Get("User-Agent"))

		// Check history
		data := params.DB().History().Data(context.Background())
		assert.Equal(1, len(data))
		rec := data["GET:/test/foo"]
		assert.Equal(200, rec.Response.StatusCode)
		assert.Equal([]byte(`{"message": "Hello, from remote!"}`), rec.Response.Data)
	})

	t.Run("upstream content-type header is forwarded", func(t *testing.T) {
		upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok":true}`))
		}))
		defer upstreamServer.Close()

		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/test/foo", nil)

		params := newTestParams(&config.ServiceConfig{
			Name: "test",
			Upstream: &config.UpstreamConfig{
				URL: upstreamServer.URL,
			},
		}, nil)

		f := CreateUpstreamRequestMiddleware(params)
		f(handler).ServeHTTP(w, req)

		assert.Equal(`{"ok":true}`, string(w.buf))
		assert.Equal("application/json; charset=utf-8", w.header.Get("Content-Type"))

		// Check history has content-type
		data := params.DB().History().Data(context.Background())
		rec := data["GET:/test/foo"]
		assert.Equal("application/json; charset=utf-8", rec.Response.ContentType)
	})

	t.Run("sets X-Cxs-Source header to upstream", func(t *testing.T) {
		upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"from":"upstream"}`))
		}))
		defer upstreamServer.Close()

		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/test/source", nil)

		params := newTestParams(&config.ServiceConfig{
			Name: "test",
			Upstream: &config.UpstreamConfig{
				URL: upstreamServer.URL,
			},
		}, nil)

		f := CreateUpstreamRequestMiddleware(params)
		f(handler).ServeHTTP(w, req)

		assert.Equal(ResponseHeaderSourceUpstream, w.header.Get(ResponseHeaderSource))
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
		req := httptest.NewRequest(http.MethodPost, "/foo/resource", body)

		params := newTestParams(&config.ServiceConfig{
			Name: "foo",
			Upstream: &config.UpstreamConfig{
				URL: upstreamServer.URL,
			},
		}, nil)

		resp := &db.HistoryResponse{
			Data:           []byte("cached"),
			StatusCode:     http.StatusOK,
			ContentType:    "application/json",
			IsFromUpstream: true,
		}
		histReq := &http.Request{
			URL:    &url.URL{Path: "/foo/resource"},
			Method: http.MethodPost,
			Body:   io.NopCloser(strings.NewReader(`{"bar": "car"}`)),
		}
		params.DB().History().Set(context.Background(), "/foo/resource", histReq, resp)

		f := CreateUpstreamRequestMiddleware(params)
		f(handler).ServeHTTP(w, req)

		assert.Equal(`{"message": "Hello, from remote!"}`, string(w.buf))
		assert.Equal(`{"foo": "bar"}`, rcvdBody)

		// Check history
		data := params.DB().History().Data(context.Background())
		assert.Equal(1, len(data))
		rec := data["POST:/foo/resource"]
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

		params := newTestParams(&config.ServiceConfig{
			Name:     "test",
			Upstream: &config.UpstreamConfig{},
		}, nil)

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
		req := httptest.NewRequest(http.MethodGet, "/test/foo", nil)

		params := newTestParams(&config.ServiceConfig{
			Name: "test",
			Upstream: &config.UpstreamConfig{
				URL: upstreamServer.URL,
			},
		}, nil)

		f := CreateUpstreamRequestMiddleware(params)
		f(handler).ServeHTTP(w, req)

		assert.Equal("Hello, from local!", string(w.buf))
	})

	t.Run("request create fails", func(t *testing.T) {
		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/test/foo", nil)

		params := newTestParams(&config.ServiceConfig{
			Name: "test",
			Upstream: &config.UpstreamConfig{
				URL: "ht tps://example.com",
			},
		}, nil)

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
		req := httptest.NewRequest(http.MethodGet, "/test/foo", nil)

		params := newTestParams(&config.ServiceConfig{
			Name: "test",
			Upstream: &config.UpstreamConfig{
				URL: upstreamServer.URL,
				FailOn: &config.UpstreamFailOnConfig{
					TimeOut: 50 * time.Millisecond,
				},
			},
		}, nil)

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
		req := httptest.NewRequest(http.MethodGet, "/test/foo", nil)

		params := newTestParams(&config.ServiceConfig{
			Name: "test",
			Upstream: &config.UpstreamConfig{
				URL: upstreamServer.URL,
				FailOn: &config.UpstreamFailOnConfig{
					HTTPStatus: config.HttpStatusFailOnConfig{
						{Exact: http.StatusMovedPermanently},
					},
				},
			},
		}, nil)

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
		req := httptest.NewRequest(http.MethodGet, "/test/foo", nil)

		params := newTestParams(&config.ServiceConfig{
			Name: "test",
			Upstream: &config.UpstreamConfig{
				URL: upstreamServer.URL,
				FailOn: &config.UpstreamFailOnConfig{
					HTTPStatus: config.HttpStatusFailOnConfig{
						{Range: "200-201"},
					},
				},
			},
		}, nil)

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

		params := newTestParams(&config.ServiceConfig{
			Name: "test",
			Upstream: &config.UpstreamConfig{
				URL: upstreamServer.URL,
				CircuitBreaker: &config.CircuitBreakerConfig{
					MinRequests:  3,
					FailureRatio: 0.6,
				},
			},
		}, nil)

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

	t.Run("no circuit breaker when not configured", func(t *testing.T) {
		callCount := 0
		upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message": "Internal Server Error!"}`))
		}))
		defer upstreamServer.Close()

		params := newTestParams(&config.ServiceConfig{
			Name: "test",
			Upstream: &config.UpstreamConfig{
				URL: upstreamServer.URL,
				// No CircuitBreaker config - should not use circuit breaker
			},
		}, nil)

		middleware := CreateUpstreamRequestMiddleware(params)
		wrappedHandler := middleware(handler)

		// Make 5 requests - all should hit upstream since no circuit breaker
		for i := 1; i <= 5; i++ {
			w := NewBufferedResponseWriter()
			req := httptest.NewRequest(http.MethodGet, "/test/foo", nil)
			wrappedHandler.ServeHTTP(w, req)
			assert.Equal("Hello, from local!", string(w.buf))
		}

		// All 5 requests should have hit upstream
		assert.Equal(5, callCount, "all requests should hit upstream when circuit breaker is not configured")
	})

	t.Run("distributed circuit breaker falls back to local when redis config is nil", func(t *testing.T) {
		callCount := 0
		upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message": "Internal Server Error!"}`))
		}))
		defer upstreamServer.Close()

		params := newTestParams(&config.ServiceConfig{
			Name: "test",
			Upstream: &config.UpstreamConfig{
				URL: upstreamServer.URL,
				CircuitBreaker: &config.CircuitBreakerConfig{
					MinRequests:  3,
					FailureRatio: 0.6,
				},
			},
		}, &config.StorageConfig{
			Type:  config.StorageTypeRedis,
			Redis: nil, // Redis type but no config - should fall back to local CB
		})

		middleware := CreateUpstreamRequestMiddleware(params)
		wrappedHandler := middleware(handler)

		// Make 3 requests to open the circuit breaker (local fallback)
		for i := 1; i <= 3; i++ {
			w := NewBufferedResponseWriter()
			req := httptest.NewRequest(http.MethodGet, "/test/foo", nil)
			wrappedHandler.ServeHTTP(w, req)
			assert.Equal("Hello, from local!", string(w.buf))
		}

		assert.Equal(3, callCount)

		// 4th request should be blocked by open circuit breaker
		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/test/foo", nil)
		wrappedHandler.ServeHTTP(w, req)

		assert.Equal("Hello, from local!", string(w.buf))
		assert.Equal(3, callCount, "circuit breaker should work even with fallback to local")
	})

	t.Run("distributed circuit breaker with valid redis config", func(t *testing.T) {
		mr := miniredis.RunT(t)

		callCount := 0
		upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message": "Internal Server Error!"}`))
		}))
		defer upstreamServer.Close()

		params := newTestParams(&config.ServiceConfig{
			Name: "test",
			Upstream: &config.UpstreamConfig{
				URL: upstreamServer.URL,
				CircuitBreaker: &config.CircuitBreakerConfig{
					MinRequests:  3,
					FailureRatio: 0.6,
				},
			},
		}, &config.StorageConfig{
			Type: config.StorageTypeRedis,
			Redis: &config.RedisConfig{
				Address: mr.Addr(),
			},
		})

		middleware := CreateUpstreamRequestMiddleware(params)
		wrappedHandler := middleware(handler)

		// Make 3 requests to open the distributed circuit breaker
		for i := 1; i <= 3; i++ {
			w := NewBufferedResponseWriter()
			req := httptest.NewRequest(http.MethodGet, "/test/foo", nil)
			wrappedHandler.ServeHTTP(w, req)
			assert.Equal("Hello, from local!", string(w.buf))
		}

		assert.Equal(3, callCount)

		// 4th request should be blocked by open circuit breaker
		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/test/foo", nil)
		wrappedHandler.ServeHTTP(w, req)

		assert.Equal("Hello, from local!", string(w.buf))
		assert.Equal(3, callCount, "distributed circuit breaker should block requests")
	})

	t.Run("distributed circuit breaker falls back when redis connection fails", func(t *testing.T) {
		callCount := 0
		upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message": "Internal Server Error!"}`))
		}))
		defer upstreamServer.Close()

		params := newTestParams(&config.ServiceConfig{
			Name: "test",
			Upstream: &config.UpstreamConfig{
				URL: upstreamServer.URL,
				CircuitBreaker: &config.CircuitBreakerConfig{
					MinRequests:  3,
					FailureRatio: 0.6,
				},
			},
		}, &config.StorageConfig{
			Type: config.StorageTypeRedis,
			Redis: &config.RedisConfig{
				Address: "invalid:99999", // Invalid address
			},
		})

		middleware := CreateUpstreamRequestMiddleware(params)
		wrappedHandler := middleware(handler)

		// Make 3 requests - should fall back to local CB
		for i := 1; i <= 3; i++ {
			w := NewBufferedResponseWriter()
			req := httptest.NewRequest(http.MethodGet, "/test/foo", nil)
			wrappedHandler.ServeHTTP(w, req)
			assert.Equal("Hello, from local!", string(w.buf))
		}

		assert.Equal(3, callCount)

		// 4th request should be blocked by local circuit breaker fallback
		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/test/foo", nil)
		wrappedHandler.ServeHTTP(w, req)

		assert.Equal("Hello, from local!", string(w.buf))
		assert.Equal(3, callCount, "fallback local circuit breaker should work")
	})
}
