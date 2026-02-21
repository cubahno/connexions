package middleware

import (
	"bytes"
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

// circuitBreakerExecutor defines the interface for circuit breaker execution.
type circuitBreakerExecutor interface {
	Execute(req func() ([]byte, error)) ([]byte, error)
}

// CreateUpstreamRequestMiddleware returns a middleware that fetches data from an upstream service.
// If circuit breaker is configured and the upstream service fails, consequent requests will be blocked.
func CreateUpstreamRequestMiddleware(params *Params) func(http.Handler) http.Handler {
	// Circuit breaker is created at startup with the original config.
	// It's tied to the upstream URL and shared across requests.
	var cb circuitBreakerExecutor
	if cfg := params.ServiceConfig.Upstream; cfg != nil && cfg.URL != "" {
		cb = createCircuitBreaker(cfg.URL, cfg.CircuitBreaker, params.DB().CircuitBreakerStore(), params.StorageConfig)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			cfg := params.ServiceConfig.Upstream
			if cfg == nil || cfg.URL == "" {
				next.ServeHTTP(w, req)
				return
			}

			log.Println("Service has upstream service defined")

			var response []byte
			var err error

			if cb != nil {
				response, err = cb.Execute(func() ([]byte, error) {
					return getUpstreamResponse(params, req)
				})
			} else {
				response, err = getUpstreamResponse(params, req)
			}

			// If an upstream service returns a successful response, write it and return immediately
			if err == nil && response != nil {
				// history was set already
				_, _ = w.Write(response)
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

// createCircuitBreaker creates a circuit breaker based on the configuration.
// Returns nil if circuit breaker is not configured.
func createCircuitBreaker(upstreamURL string, cbCfg *config.CircuitBreakerConfig, cbStore gobreaker.SharedDataStore, storageCfg *config.StorageConfig) circuitBreakerExecutor {
	if cbCfg == nil {
		return nil
	}

	settings := buildCircuitBreakerSettings(upstreamURL, cbCfg)

	// Create distributed circuit breaker if Redis storage is configured
	if storageCfg != nil && storageCfg.Type == config.StorageTypeRedis {
		return createDistributedCircuitBreaker(settings, cbStore)
	}

	return gobreaker.NewCircuitBreaker[[]byte](settings)
}

// buildCircuitBreakerSettings creates gobreaker.Settings from config.
func buildCircuitBreakerSettings(upstreamURL string, cbCfg *config.CircuitBreakerConfig) gobreaker.Settings {
	cfg := cbCfg.WithDefaults()

	return gobreaker.Settings{
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
			return isOpen
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			slog.Info("Circuit breaker state changed",
				"name", name,
				"from", from.String(),
				"to", to.String(),
			)
		},
	}
}

// createDistributedCircuitBreaker creates a distributed circuit breaker using the provided store.
func createDistributedCircuitBreaker(settings gobreaker.Settings, store gobreaker.SharedDataStore) circuitBreakerExecutor {
	dcb, err := gobreaker.NewDistributedCircuitBreaker[[]byte](store, settings)
	if err != nil {
		slog.Error("Failed to create distributed circuit breaker, falling back to local",
			"name", settings.Name,
			"error", err,
		)
		return gobreaker.NewCircuitBreaker[[]byte](settings)
	}

	slog.Info("Created distributed circuit breaker",
		"name", settings.Name,
	)
	return dcb
}

func getUpstreamResponse(params *Params, req *http.Request) ([]byte, error) {
	cfg := params.ServiceConfig.Upstream

	failOn := cfg.FailOn

	timeOut := 5 * time.Second
	if failOn != nil && cfg.FailOn.TimeOut > 0 {
		timeOut = cfg.FailOn.TimeOut
	}

	// TODO: add time out to the upstream config
	client := http.Client{
		Timeout: timeOut,
	}

	history := params.DB().History()
	resourcePrefix := "/" + params.ServiceConfig.Name
	rec := history.Set(req.URL.Path, req, nil)

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
	upReq.Header.Set("User-Agent", "Connexions/2.0")

	slog.Info("Upstream request",
		"method", upReq.Method,
		"url", upReq.URL.String(),
		"headers", upReq.Header,
	)

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

	if failOn != nil && failOn.HTTPStatus != nil && failOn.HTTPStatus.Is(statusCode) {
		return nil, fmt.Errorf("failOn condition met for status code: %d", statusCode)
	}

	if statusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("upstream response failed with status code %d, body: %s", statusCode, string(body))
	}

	slog.Info("received successful upstream response", "body", string(body))

	historyResponse := &db.Response{
		Data:           body,
		StatusCode:     statusCode,
		IsFromUpstream: true,
	}
	history.Set(req.URL.Path, req, historyResponse)

	return body, nil
}
