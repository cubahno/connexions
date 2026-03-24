package middleware

import (
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
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
		waitForAsync()

		assert.Equal(`{"message": "Hello, from remote!"}`, string(w.buf))

		// Check that headers were forwarded to upstream
		assert.Equal("Bearer 123", receivedHeaders.Get("Authorization"))
		assert.Equal("test", receivedHeaders.Get("X-Test"))
		assert.Equal("Connexions/2.0", receivedHeaders.Get("User-Agent"))

		// Check history
		data := params.DB().History().Data(context.Background())
		assert.Equal(1, len(data))
		rec := data[0]
		assert.Equal(200, rec.Response.StatusCode)
		assert.Equal([]byte(`{"message": "Hello, from remote!"}`), rec.Response.Body)
	})

	t.Run("X-Cxs-Upstream-Headers allowlist forwards only listed headers", func(t *testing.T) {
		var receivedHeaders http.Header
		upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedHeaders = r.Header.Clone()
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"message": "OK"}`))
		}))
		defer upstreamServer.Close()

		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/test/foo", nil)
		req.Header.Set("Authorization", "Basic internal-creds")
		req.Header.Set("Smartum-Version", "2020-04-02")
		req.Header.Set("X-Custom", "keep-me")
		req.Header.Set("Cookie", "session=abc")
		req.Header.Set("Origin", "http://localhost:2200")
		req.Header.Set("Sec-Fetch-Mode", "cors")
		req.Header.Set("X-Cxs-Upstream-Headers", "Smartum-Version,X-Custom")

		params := newTestParams(&config.ServiceConfig{
			Name: "test",
			Upstream: &config.UpstreamConfig{
				URL: upstreamServer.URL,
			},
		}, nil)

		f := CreateUpstreamRequestMiddleware(params)
		f(handler).ServeHTTP(w, req)
		waitForAsync()

		assert.Equal(`{"message": "OK"}`, string(w.buf))

		// Only allowlisted headers should be forwarded
		assert.Equal("2020-04-02", receivedHeaders.Get("Smartum-Version"))
		assert.Equal("keep-me", receivedHeaders.Get("X-Custom"))
		assert.Equal("Connexions/2.0", receivedHeaders.Get("User-Agent"))

		// Everything else should be stripped
		assert.Empty(receivedHeaders.Get("Authorization"))
		assert.Empty(receivedHeaders.Get("Cookie"))
		assert.Empty(receivedHeaders.Get("Origin"))
		assert.Empty(receivedHeaders.Get("Sec-Fetch-Mode"))
		assert.Empty(receivedHeaders.Get("X-Cxs-Upstream-Headers"))
	})

	t.Run("query parameters are forwarded to upstream", func(t *testing.T) {
		var receivedURL string
		upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedURL = r.URL.String()
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"message": "OK"}`))
		}))
		defer upstreamServer.Close()

		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/test/payment/charge?reference=abc-123&amount=1000", nil)

		params := newTestParams(&config.ServiceConfig{
			Name: "test",
			Upstream: &config.UpstreamConfig{
				URL: upstreamServer.URL,
			},
		}, nil)

		f := CreateUpstreamRequestMiddleware(params)
		f(handler).ServeHTTP(w, req)
		waitForAsync()

		assert.Equal(`{"message": "OK"}`, string(w.buf))
		assert.Equal("/payment/charge?reference=abc-123&amount=1000", receivedURL)
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
		waitForAsync()

		assert.Equal(`{"ok":true}`, string(w.buf))
		assert.Equal("application/json; charset=utf-8", w.header.Get("Content-Type"))

		// Check history has content-type
		data := params.DB().History().Data(context.Background())
		assert.Equal(1, len(data))
		rec := data[0]
		assert.Equal("application/json; charset=utf-8", rec.Response.ContentType)
	})

	t.Run("gzip upstream response is decompressed", func(t *testing.T) {
		upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Set("Content-Type", "application/json")
			gz := gzip.NewWriter(w)
			_, _ = gz.Write([]byte(`{"compressed": true}`))
			_ = gz.Close()
		}))
		defer upstreamServer.Close()

		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/test/gzip", nil)
		req.Header.Set("Accept-Encoding", "gzip")

		params := newTestParams(&config.ServiceConfig{
			Name: "test",
			Upstream: &config.UpstreamConfig{
				URL: upstreamServer.URL,
			},
		}, nil)

		f := CreateUpstreamRequestMiddleware(params)
		f(handler).ServeHTTP(w, req)
		waitForAsync()

		assert.Equal(`{"compressed": true}`, string(w.buf))
	})

	t.Run("configured upstream headers are applied", func(t *testing.T) {
		var receivedHeaders http.Header
		upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedHeaders = r.Header.Clone()
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok":true}`))
		}))
		defer upstreamServer.Close()

		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/test/headers", nil)
		req.Header.Set("Authorization", "Bearer original")

		params := newTestParams(&config.ServiceConfig{
			Name: "test",
			Upstream: &config.UpstreamConfig{
				URL: upstreamServer.URL,
				Headers: map[string]string{
					"Authorization": "Bearer configured",
					"X-Custom":      "custom-value",
				},
			},
		}, nil)

		f := CreateUpstreamRequestMiddleware(params)
		f(handler).ServeHTTP(w, req)
		waitForAsync()

		assert.Equal(`{"ok":true}`, string(w.buf))
		assert.Equal("Bearer configured", receivedHeaders.Get("Authorization"))
		assert.Equal("custom-value", receivedHeaders.Get("X-Custom"))
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
		waitForAsync()

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
			Body:           []byte("cached"),
			StatusCode:     http.StatusOK,
			ContentType:    "application/json",
			IsFromUpstream: true,
		}
		params.DB().History().Set(context.Background(), "/foo/resource", &db.HistoryRequest{
			Method: http.MethodPost,
			URL:    "/foo/resource",
			Body:   []byte(`{"bar": "car"}`),
		}, resp)

		f := CreateUpstreamRequestMiddleware(params)
		f(handler).ServeHTTP(w, req)
		waitForAsync()

		assert.Equal(`{"message": "Hello, from remote!"}`, string(w.buf))
		assert.Equal(`{"foo": "bar"}`, rcvdBody)

		// Check history - 2 entries: the seeded one + the new upstream result
		data := params.DB().History().Data(context.Background())
		assert.Equal(2, len(data))
		// Latest entry should have the upstream response
		rec := data[len(data)-1]
		assert.Equal(200, rec.Response.StatusCode)
		assert.Equal([]byte(`{"message": "Hello, from remote!"}`), rec.Response.Body)
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
		waitForAsync()

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
		waitForAsync()

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
		waitForAsync()

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
				URL:     upstreamServer.URL,
				Timeout: 50 * time.Millisecond,
			},
		}, nil)

		f := CreateUpstreamRequestMiddleware(params)
		f(handler).ServeHTTP(w, req)
		waitForAsync()

		assert.Equal("Hello, from local!", string(w.buf))
	})

	t.Run("400 returns upstream error by default (fail-on 400)", func(t *testing.T) {
		upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"message": "Bad Request"}`))
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
		waitForAsync()

		assert.Equal(`{"message": "Bad Request"}`, string(w.buf))
		assert.Equal(http.StatusBadRequest, w.statusCode)
		assert.Equal("application/json", w.header.Get("Content-Type"))
		assert.Equal(ResponseHeaderSourceUpstream, w.header.Get(ResponseHeaderSource))
	})

	t.Run("non-400 4xx falls back to generator by default", func(t *testing.T) {
		upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"message": "Unauthorized"}`))
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
		waitForAsync()

		assert.Equal("Hello, from local!", string(w.buf))
	})

	t.Run("4xx falls back to generator when fail-on is empty", func(t *testing.T) {
		upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"message": "Unauthorized"}`))
		}))
		defer upstreamServer.Close()

		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/test/foo", nil)

		params := newTestParams(&config.ServiceConfig{
			Name: "test",
			Upstream: &config.UpstreamConfig{
				URL:    upstreamServer.URL,
				FailOn: &config.HTTPStatusMatchConfig{},
			},
		}, nil)

		f := CreateUpstreamRequestMiddleware(params)
		f(handler).ServeHTTP(w, req)
		waitForAsync()

		assert.Equal("Hello, from local!", string(w.buf))
	})

	t.Run("5xx falls back to generator by default", func(t *testing.T) {
		upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"message": "Bad Gateway"}`))
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
		waitForAsync()

		assert.Equal("Hello, from local!", string(w.buf))
	})

	t.Run("custom fail-on includes 5xx", func(t *testing.T) {
		upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"message": "Bad Gateway"}`))
		}))
		defer upstreamServer.Close()

		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/test/foo", nil)

		params := newTestParams(&config.ServiceConfig{
			Name: "test",
			Upstream: &config.UpstreamConfig{
				URL: upstreamServer.URL,
				FailOn: &config.HTTPStatusMatchConfig{
					{Range: "400-599"},
				},
			},
		}, nil)

		f := CreateUpstreamRequestMiddleware(params)
		f(handler).ServeHTTP(w, req)
		waitForAsync()

		assert.Equal(`{"message": "Bad Gateway"}`, string(w.buf))
		assert.Equal(http.StatusBadGateway, w.statusCode)
	})

	t.Run("successful requests do not trip circuit breaker with trip-on-status", func(t *testing.T) {
		callCount := 0
		upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"message": "OK"}`))
		}))
		defer upstreamServer.Close()

		params := newTestParams(&config.ServiceConfig{
			Name: "test",
			Upstream: &config.UpstreamConfig{
				URL: upstreamServer.URL,
				CircuitBreaker: &config.CircuitBreakerConfig{
					MinRequests:  3,
					FailureRatio: 0.6,
					TripOnStatus: config.HTTPStatusMatchConfig{
						{Range: "500-599"},
					},
				},
			},
		}, nil)

		middleware := CreateUpstreamRequestMiddleware(params)
		wrappedHandler := middleware(handler)

		// Make 5 successful requests - should NOT trip circuit breaker.
		for i := 1; i <= 5; i++ {
			w := NewBufferedResponseWriter()
			req := httptest.NewRequest(http.MethodGet, "/test/foo", nil)
			wrappedHandler.ServeHTTP(w, req)
			waitForAsync()
			assert.Equal(`{"message": "OK"}`, string(w.buf))
		}

		// All 5 should have hit upstream (CB never opened)
		assert.Equal(5, callCount, "successful requests should not trip circuit breaker")
	})

	t.Run("non-trip-on 4xx does not trip circuit breaker", func(t *testing.T) {
		callCount := 0
		upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"message": "Unauthorized"}`))
		}))
		defer upstreamServer.Close()

		params := newTestParams(&config.ServiceConfig{
			Name: "test",
			Upstream: &config.UpstreamConfig{
				URL: upstreamServer.URL,
				CircuitBreaker: &config.CircuitBreakerConfig{
					MinRequests:  3,
					FailureRatio: 0.6,
					TripOnStatus: config.HTTPStatusMatchConfig{
						{Range: "500-599"},
					},
				},
			},
		}, nil)

		middleware := CreateUpstreamRequestMiddleware(params)
		wrappedHandler := middleware(handler)

		// Make 5 requests - all return 401, which is outside trip-on-status range.
		// Circuit breaker should NOT open because these are not counted as failures.
		for i := 1; i <= 5; i++ {
			w := NewBufferedResponseWriter()
			req := httptest.NewRequest(http.MethodGet, "/test/foo", nil)
			wrappedHandler.ServeHTTP(w, req)
			waitForAsync()
			assert.Equal("Hello, from local!", string(w.buf))
		}

		// All 5 requests should have hit upstream (CB never opened)
		assert.Equal(5, callCount, "circuit breaker should not trip on non-trip-on status codes")
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
			waitForAsync()

			// Should fall back to local handler
			assert.Equal("Hello, from local!", string(w.buf))
		}

		// Verify upstream was called 3 times
		assert.Equal(3, callCount)

		// 4th request should be blocked by open circuit breaker
		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/test/foo", nil)
		wrappedHandler.ServeHTTP(w, req)
		waitForAsync()

		// Circuit breaker is open, should fall back to local handler
		assert.Equal("Hello, from local!", string(w.buf))

		// Verify upstream was NOT called on 4th request (still 3 calls)
		assert.Equal(3, callCount, "upstream should not be called when circuit breaker is open")
	})

	t.Run("circuit breaker state preserves counts and error on transition", func(t *testing.T) {
		callCount := 0
		upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"message": "Bad Gateway"}`))
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

		mw := CreateUpstreamRequestMiddleware(params)
		wrappedHandler := mw(handler)

		// Make 3 requests to trip the circuit breaker
		for i := 1; i <= 3; i++ {
			w := NewBufferedResponseWriter()
			req := httptest.NewRequest(http.MethodGet, "/test/foo", nil)
			wrappedHandler.ServeHTTP(w, req)
			waitForAsync()
		}

		assert.Equal(3, callCount)

		// Verify state has counts preserved
		cbTable := params.DB().Table("circuit-breaker")
		ctx := context.Background()

		state, ok := GetCBState(ctx, cbTable)
		assert.True(ok)
		assert.Equal("open", state.State)
		assert.Equal(uint32(3), state.Requests, "state should preserve request count")
		assert.Equal(uint32(3), state.TotalFailures, "state should preserve failure count")
		assert.Equal(float64(1), state.FailureRatio, "state should preserve failure ratio")
		assert.NotEmpty(state.LastError, "state should contain the last error")
		assert.Contains(state.LastError, "502")

		// Verify event has error
		events := GetCBEvents(ctx, cbTable)
		assert.NotEmpty(events)
		var openEvent *CBEvent
		for i := range events {
			if events[i].From == "closed" && events[i].To == "open" {
				openEvent = &events[i]
				break
			}
		}
		assert.NotNil(openEvent, "should have closed->open event")
		assert.NotEmpty(openEvent.Error, "event should contain the error")
		assert.Contains(openEvent.Error, "502")
	})

	t.Run("circuit breaker writes state and events to table", func(t *testing.T) {
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

		mw := CreateUpstreamRequestMiddleware(params)
		wrappedHandler := mw(handler)

		// Make 3 requests to open the circuit breaker
		for i := 1; i <= 3; i++ {
			w := NewBufferedResponseWriter()
			req := httptest.NewRequest(http.MethodGet, "/test/foo", nil)
			wrappedHandler.ServeHTTP(w, req)
			waitForAsync()
		}

		assert.Equal(3, callCount)

		// Verify CB state was written
		cbTable := params.DB().Table("circuit-breaker")
		ctx := context.Background()

		state, ok := GetCBState(ctx, cbTable)
		assert.True(ok, "CB state should be written")
		assert.NotNil(state)
		// After tripping, OnStateChange writes state "open"
		assert.Equal("open", state.State)
		assert.NotEmpty(state.LastUpdated)

		// Verify events were written
		events := GetCBEvents(ctx, cbTable)
		assert.NotEmpty(events, "CB events should be written")
		// Should have at least one transition: closed -> open
		found := false
		for _, e := range events {
			if e.From == "closed" && e.To == "open" {
				found = true
				break
			}
		}
		assert.True(found, "should have closed->open transition event")
	})

	t.Run("successful requests update stored state", func(t *testing.T) {
		upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"message": "OK"}`))
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

		mw := CreateUpstreamRequestMiddleware(params)
		wrappedHandler := mw(handler)

		// Make 5 successful requests
		for i := 1; i <= 5; i++ {
			w := NewBufferedResponseWriter()
			req := httptest.NewRequest(http.MethodGet, "/test/foo", nil)
			wrappedHandler.ServeHTTP(w, req)
			waitForAsync()
			assert.Equal(`{"message": "OK"}`, string(w.buf))
		}

		// Verify stored state reflects all 5 requests
		cbTable := params.DB().Table("circuit-breaker")
		ctx := context.Background()

		state, ok := GetCBState(ctx, cbTable)
		assert.True(ok, "CB state should be written after successful requests")
		assert.NotNil(state)
		assert.Equal("closed", state.State)
		assert.Equal(uint32(5), state.Requests, "state should count all requests including successes")
		assert.Equal(uint32(5), state.TotalSuccesses, "state should track successful requests")
		assert.Equal(uint32(0), state.TotalFailures)
	})

	t.Run("mixed success and failure requests update stored state", func(t *testing.T) {
		callCount := 0
		upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			// First 2 requests succeed, 3rd fails
			if callCount == 2 {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error": "fail"}`))
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"message": "OK"}`))
		}))
		defer upstreamServer.Close()

		params := newTestParams(&config.ServiceConfig{
			Name: "test",
			Upstream: &config.UpstreamConfig{
				URL: upstreamServer.URL,
				CircuitBreaker: &config.CircuitBreakerConfig{
					MinRequests:  5,
					FailureRatio: 0.6,
				},
			},
		}, nil)

		mw := CreateUpstreamRequestMiddleware(params)
		wrappedHandler := mw(handler)

		// Make 3 requests: success, failure, success
		for i := 0; i < 3; i++ {
			w := NewBufferedResponseWriter()
			req := httptest.NewRequest(http.MethodGet, "/test/foo", nil)
			wrappedHandler.ServeHTTP(w, req)
			waitForAsync()
		}

		// Verify stored state reflects all 3 requests
		cbTable := params.DB().Table("circuit-breaker")
		ctx := context.Background()

		state, ok := GetCBState(ctx, cbTable)
		assert.True(ok, "CB state should be present")
		assert.Equal(uint32(3), state.Requests, "state should count all requests")
		assert.Equal(uint32(2), state.TotalSuccesses)
		assert.Equal(uint32(1), state.TotalFailures)
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
			waitForAsync()
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
			waitForAsync()
			assert.Equal("Hello, from local!", string(w.buf))
		}

		assert.Equal(3, callCount)

		// 4th request should be blocked by open circuit breaker
		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/test/foo", nil)
		wrappedHandler.ServeHTTP(w, req)
		waitForAsync()

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
			waitForAsync()
			assert.Equal("Hello, from local!", string(w.buf))
		}

		assert.Equal(3, callCount)

		// 4th request should be blocked by open circuit breaker
		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/test/foo", nil)
		wrappedHandler.ServeHTTP(w, req)
		waitForAsync()

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
			waitForAsync()
			assert.Equal("Hello, from local!", string(w.buf))
		}

		assert.Equal(3, callCount)

		// 4th request should be blocked by local circuit breaker fallback
		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/test/foo", nil)
		wrappedHandler.ServeHTTP(w, req)
		waitForAsync()

		assert.Equal("Hello, from local!", string(w.buf))
		assert.Equal(3, callCount, "fallback local circuit breaker should work")
	})
}
