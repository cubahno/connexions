package middleware

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/cubahno/connexions/v2/pkg/config"
	"github.com/cubahno/connexions/v2/pkg/db"
	"github.com/sony/gobreaker/v2"
)

// upstreamHTTPError is returned when upstream responds with an error status code.
// It carries the status code so the circuit breaker can decide whether to count it as a failure.
type upstreamHTTPError struct {
	StatusCode int
	Body       string
}

func (e *upstreamHTTPError) Error() string {
	return fmt.Sprintf("upstream response failed with status code %d, body: %s", e.StatusCode, e.Body)
}

// upstreamResponse holds the response data from an upstream service.
type upstreamResponse struct {
	Body        []byte
	ContentType string
}

// circuitBreakerExecutor defines the interface for circuit breaker execution.
type circuitBreakerExecutor interface {
	Execute(req func() (*upstreamResponse, error)) (*upstreamResponse, error)
}

// CreateUpstreamRequestMiddleware returns a middleware that fetches data from an upstream service.
// If circuit breaker is configured and the upstream service fails, consequent requests will be blocked.
func CreateUpstreamRequestMiddleware(params *Params) func(http.Handler) http.Handler {
	// Circuit breaker is created at startup with the original config.
	// It's tied to the upstream URL and shared across requests.
	var cb circuitBreakerExecutor
	if cfg := params.ServiceConfig.Upstream; cfg != nil && cfg.URL != "" && cfg.CircuitBreaker != nil {
		settings := buildCircuitBreakerSettings(cfg.URL, cfg.CircuitBreaker, params.DB().Table("circuit-breaker"))
		cb = createCircuitBreaker(settings, params.DB().CircuitBreakerStore(), params.StorageConfig)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			cfg := params.ServiceConfig.Upstream
			if cfg == nil || cfg.URL == "" {
				next.ServeHTTP(w, req)
				return
			}

			log.Println("Service has upstream service defined")

			var resp *upstreamResponse
			var err error

			if cb != nil {
				resp, err = cb.Execute(func() (*upstreamResponse, error) {
					return getUpstreamResponse(params, req)
				})
			} else {
				resp, err = getUpstreamResponse(params, req)
			}

			// If an upstream service returns a successful response, write it and return immediately
			if err == nil && resp != nil {
				SetDurationHeader(w, req)
				w.Header().Set(ResponseHeaderSource, ResponseHeaderSourceUpstream)
				if resp.ContentType != "" {
					w.Header().Set("Content-Type", resp.ContentType)
				}
				_, _ = w.Write(resp.Body)
				return
			}

			if err != nil {
				slog.Error("Error fetching upstream service", "url", cfg.URL, "error", err)
			}

			// Proceed to the next handler if no upstream service matched
			next.ServeHTTP(w, req)
		})
	}
}

// createCircuitBreaker creates a circuit breaker from pre-built settings.
func createCircuitBreaker(settings gobreaker.Settings, cbStore gobreaker.SharedDataStore, storageCfg *config.StorageConfig) circuitBreakerExecutor {
	// Create distributed circuit breaker if Redis storage is configured
	if storageCfg != nil && storageCfg.Type == config.StorageTypeRedis {
		return createDistributedCircuitBreaker(settings, cbStore)
	}

	return gobreaker.NewCircuitBreaker[*upstreamResponse](settings)
}

// buildCircuitBreakerSettings creates gobreaker.Settings from config.
func buildCircuitBreakerSettings(upstreamURL string, cbCfg *config.CircuitBreakerConfig, cbTable db.Table) gobreaker.Settings {
	cfg := cbCfg.WithDefaults()

	settings := gobreaker.Settings{
		Name:        upstreamURL,
		Timeout:     cfg.Timeout,
		MaxRequests: cfg.MaxRequests,
		Interval:    cfg.Interval,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			slog.Debug("Circuit breaker check",
				"url", upstreamURL,
				"requests", counts.Requests,
				"totalSuccesses", counts.TotalSuccesses,
				"totalFailures", counts.TotalFailures,
				"consecutiveSuccesses", counts.ConsecutiveSuccesses,
				"consecutiveFailures", counts.ConsecutiveFailures,
			)
			if counts.Requests < cfg.MinRequests {
				return false
			}
			ratio := float64(counts.TotalFailures) / float64(counts.Requests)
			isOpen := ratio >= cfg.FailureRatio
			if isOpen {
				slog.Info("Circuit breaker is open",
					"url", upstreamURL,
					"failureRatio", ratio,
				)
			}

			// Write current snapshot to observability table
			ctx := context.Background()
			state := newCBState("closed", counts)
			cbTable.Set(ctx, cbKeyState, state, 0)

			return isOpen
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			slog.Info("Circuit breaker state changed",
				"name", name,
				"from", from.String(),
				"to", to.String(),
			)

			// Write state snapshot and event to observability table
			ctx := context.Background()
			state := CBState{
				State:       to.String(),
				LastUpdated: time.Now().UTC().Format(time.RFC3339),
			}
			cbTable.Set(ctx, cbKeyState, state, 0)

			appendCBEvent(ctx, cbTable, CBEvent{
				From:      from.String(),
				To:        to.String(),
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			})
		},
	}

	if len(cfg.TripOnStatus) > 0 {
		settings.IsSuccessful = func(err error) bool {
			var httpErr *upstreamHTTPError
			if errors.As(err, &httpErr) {
				return !cfg.TripOnStatus.Is(httpErr.StatusCode)
			}
			return false
		}
	}

	return settings
}

// createDistributedCircuitBreaker creates a distributed circuit breaker using the provided store.
func createDistributedCircuitBreaker(settings gobreaker.Settings, store gobreaker.SharedDataStore) circuitBreakerExecutor {
	dcb, err := gobreaker.NewDistributedCircuitBreaker[*upstreamResponse](store, settings)
	if err != nil {
		slog.Error("Failed to create distributed circuit breaker, falling back to local",
			"name", settings.Name,
			"error", err,
		)
		return gobreaker.NewCircuitBreaker[*upstreamResponse](settings)
	}

	slog.Info("Created distributed circuit breaker",
		"name", settings.Name,
	)
	return dcb
}

func getUpstreamResponse(params *Params, req *http.Request) (*upstreamResponse, error) {
	cfg := params.ServiceConfig.Upstream

	timeout := config.DefaultUpstreamTimeout
	if cfg.Timeout > 0 {
		timeout = cfg.Timeout
	}

	client := http.Client{
		Timeout: timeout,
	}

	ctx := req.Context()
	history := params.DB().History()
	resourcePrefix := "/" + params.ServiceConfig.Name
	rec := history.Set(ctx, req.URL.Path, req, nil)

	bodyBytes := rec.Body

	outURL := fmt.Sprintf("%s/%s",
		strings.TrimSuffix(cfg.URL, "/"),
		strings.TrimPrefix(req.URL.Path[len(resourcePrefix):], "/"))

	log.Println("Upstream request", "method", req.Method, "url", outURL)

	upReq, err := http.NewRequest(req.Method, outURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		slog.Error("Failed to create request", "error", err)
		return nil, err
	}

	for name, values := range req.Header {
		for _, value := range values {
			upReq.Header.Add(name, value)
		}
	}

	// Remove Accept-Encoding so Go's http.Transport handles decompression
	// transparently. When set explicitly, Transport skips auto-decompression
	// and io.ReadAll returns raw compressed bytes (e.g. gzip).
	upReq.Header.Del("Accept-Encoding")
	upReq.Header.Set("User-Agent", "Connexions/2.0")
	for name, value := range cfg.Headers {
		upReq.Header.Set(name, value)
	}

	slog.Info("Upstream request", "method", upReq.Method, "url", upReq.URL.String())

	resp, err := client.Do(upReq)
	if err != nil {
		return nil, fmt.Errorf("error calling upstream service %s: %s", upReq.URL.String(), err)
	}

	statusCode := resp.StatusCode

	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response from upstream service %s: %s", upReq.URL, err)
	}

	if statusCode >= http.StatusBadRequest {
		return nil, &upstreamHTTPError{StatusCode: statusCode, Body: string(body)}
	}

	slog.Info("received successful upstream response", "body", string(body))

	contentType := resp.Header.Get("Content-Type")

	historyResponse := &db.HistoryResponse{
		Data:           body,
		StatusCode:     statusCode,
		ContentType:    contentType,
		IsFromUpstream: true,
	}
	history.Set(ctx, req.URL.Path, req, historyResponse)

	return &upstreamResponse{
		Body:        body,
		ContentType: contentType,
	}, nil
}
