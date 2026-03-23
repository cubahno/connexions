package middleware

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/cubahno/connexions/v2/pkg/config"
	"github.com/cubahno/connexions/v2/pkg/db"
	"github.com/sony/gobreaker/v2"
)

// upstreamHTTPError is returned when upstream responds with an error status code.
// It carries the status code so the circuit breaker can decide whether to count it as a failure,
// and the body/content-type so fail-on can forward the response to the client.
type upstreamHTTPError struct {
	StatusCode  int
	Body        string
	ContentType string
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

// observableCircuitBreaker wraps a circuit breaker to persist state after every request,
// not just on failures. Without this, successful requests never update the stored CBState
// because gobreaker's ReadyToTrip callback only fires on failures.
type observableCircuitBreaker struct {
	inner     circuitBreakerExecutor
	cb        *gobreaker.CircuitBreaker[*upstreamResponse]
	cbTable   db.Table
	lastError *string
}

func (o *observableCircuitBreaker) Execute(req func() (*upstreamResponse, error)) (*upstreamResponse, error) {
	prevState := o.cb.State()
	resp, err := o.inner.Execute(req)

	// Persist state for requests that went through to the upstream.
	// Skip when the CB rejected the request (no actual request happened)
	// or when a state transition occurred (OnStateChange handles that with
	// the pre-clear counts snapshot).
	if !errors.Is(err, gobreaker.ErrOpenState) && !errors.Is(err, gobreaker.ErrTooManyRequests) && o.cb.State() == prevState {
		state := newCBState(o.cb.State().String(), o.cb.Counts())
		state.LastError = *o.lastError
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), asyncWriteTimeout)
			defer cancel()
			o.cbTable.Set(ctx, cbKeyState, state, 0)
		}()
	}

	return resp, err
}

// CreateUpstreamRequestMiddleware returns a middleware that fetches data from an upstream service.
// If circuit breaker is configured and the upstream service fails, consequent requests will be blocked.
func CreateUpstreamRequestMiddleware(params *Params) func(http.Handler) http.Handler {
	log := params.Logger("upstream")

	// Circuit breaker is created at startup with the original config.
	// It's tied to the upstream URL and shared across requests.
	var cb circuitBreakerExecutor
	if cfg := params.ServiceConfig.Upstream; cfg != nil && cfg.URL != "" && cfg.CircuitBreaker != nil {
		cbTable := params.DB().Table("circuit-breaker")
		var lastError string
		settings := buildCircuitBreakerSettings(log, cfg.URL, cfg.CircuitBreaker, cbTable, &lastError)

		var inner circuitBreakerExecutor
		var localCB *gobreaker.CircuitBreaker[*upstreamResponse]

		if storageCfg := params.StorageConfig; storageCfg != nil && storageCfg.Type == config.StorageTypeRedis {
			dcb, err := gobreaker.NewDistributedCircuitBreaker[*upstreamResponse](params.DB().CircuitBreakerStore(), settings)
			if err != nil {
				log.Error("Failed to create distributed circuit breaker, falling back to local",
					"name", settings.Name,
					"error", err,
				)
			} else {
				log.Info("Created distributed circuit breaker", "name", settings.Name)
				inner = dcb
				localCB = dcb.CircuitBreaker
			}
		}

		if inner == nil {
			lcb := gobreaker.NewCircuitBreaker[*upstreamResponse](settings)
			inner = lcb
			localCB = lcb
		}

		cb = &observableCircuitBreaker{
			inner:     inner,
			cb:        localCB,
			cbTable:   cbTable,
			lastError: &lastError,
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			cfg := params.ServiceConfig.Upstream
			if cfg == nil || cfg.URL == "" {
				next.ServeHTTP(w, req)
				return
			}

			log.Debug("Service has upstream service defined")

			var resp *upstreamResponse
			var err error

			if cb != nil {
				resp, err = cb.Execute(func() (*upstreamResponse, error) {
					return getUpstreamResponse(log, params, req)
				})
			} else {
				resp, err = getUpstreamResponse(log, params, req)
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
				log.Error("Error fetching upstream service", "url", cfg.URL, "error", err)

				// Check fail-on: return upstream error directly without generator fallback.
				// nil (omitted) = default (400); pointer to empty list = disabled.
				failOn := cfg.FailOn
				if failOn == nil {
					failOn = &config.DefaultFailOnStatus
				}
				var httpErr *upstreamHTTPError
				if len(*failOn) > 0 && errors.As(err, &httpErr) && failOn.Is(httpErr.StatusCode) {
					log.Info("Upstream error matches fail-on, returning directly",
						"status", httpErr.StatusCode,
					)

					if params.ServiceConfig.HistoryEnabled() {
						urlCopy := *req.URL
						go func() {
							ctx, cancel := context.WithTimeout(context.Background(), asyncWriteTimeout)
							defer cancel()
							params.DB().History().Set(ctx, req.URL.Path, &http.Request{
								Method:     req.Method,
								URL:        &urlCopy,
								RemoteAddr: req.RemoteAddr,
								Body:       http.NoBody,
							}, &db.HistoryResponse{
								Data:           []byte(httpErr.Body),
								StatusCode:     httpErr.StatusCode,
								ContentType:    httpErr.ContentType,
								IsFromUpstream: true,
							})
						}()
					}

					SetDurationHeader(w, req)
					w.Header().Set(ResponseHeaderSource, ResponseHeaderSourceUpstream)
					if httpErr.ContentType != "" {
						w.Header().Set("Content-Type", httpErr.ContentType)
					}
					w.WriteHeader(httpErr.StatusCode)
					_, _ = w.Write([]byte(httpErr.Body))
					return
				}
			}

			// Proceed to the next handler if no upstream service matched
			next.ServeHTTP(w, req)
		})
	}
}

// buildCircuitBreakerSettings creates gobreaker.Settings from config.
// lastError is shared with the observableCircuitBreaker wrapper for state persistence.
func buildCircuitBreakerSettings(log *slog.Logger, upstreamURL string, cbCfg *config.CircuitBreakerConfig, cbTable db.Table, lastError *string) gobreaker.Settings {
	cfg := cbCfg.WithDefaults()

	// lastCounts captures the counts from ReadyToTrip so OnStateChange can persist them.
	var lastCounts gobreaker.Counts

	settings := gobreaker.Settings{
		Name:        upstreamURL,
		Timeout:     cfg.Timeout,
		MaxRequests: cfg.MaxRequests,
		Interval:    cfg.Interval,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			log.Debug("Circuit breaker check",
				"url", upstreamURL,
				"cb.requests", counts.Requests,
				"cb.totalSuccesses", counts.TotalSuccesses,
				"cb.totalFailures", counts.TotalFailures,
				"cb.consecutiveSuccesses", counts.ConsecutiveSuccesses,
				"cb.consecutiveFailures", counts.ConsecutiveFailures,
			)
			lastCounts = counts

			if counts.Requests < cfg.MinRequests {
				return false
			}
			ratio := float64(counts.TotalFailures) / float64(counts.Requests)
			if ratio >= cfg.FailureRatio {
				log.Info("Circuit breaker is open",
					"url", upstreamURL,
					"cb.failureRatio", ratio,
				)
				return true
			}

			return false
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			log.Info("Circuit breaker state changed",
				"name", name,
				"cb.from", from.String(),
				"cb.to", to.String(),
				"cb.lastError", *lastError,
			)

			// Write state snapshot (with counts from last ReadyToTrip) and event
			ctx := context.Background()
			state := newCBState(to.String(), lastCounts)
			state.LastError = *lastError
			cbTable.Set(ctx, cbKeyState, state, 0)

			appendCBEvent(ctx, cbTable, CBEvent{
				From:      from.String(),
				To:        to.String(),
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				Error:     *lastError,
			})
		},
		IsSuccessful: func(err error) bool {
			if err == nil {
				return true
			}
			*lastError = err.Error()
			log.Warn("Upstream error",
				"url", upstreamURL,
				"error", err,
			)
			if len(cfg.TripOnStatus) > 0 {
				var httpErr *upstreamHTTPError
				if errors.As(err, &httpErr) {
					return !cfg.TripOnStatus.Is(httpErr.StatusCode)
				}
			}
			return false
		},
	}

	return settings
}

func getUpstreamResponse(log *slog.Logger, params *Params, req *http.Request) (*upstreamResponse, error) {
	cfg := params.ServiceConfig.Upstream

	timeout := config.DefaultUpstreamTimeout
	if cfg.Timeout > 0 {
		timeout = cfg.Timeout
	}

	client := http.Client{
		Timeout: timeout,
	}

	history := params.DB().History()
	resourcePrefix := "/" + params.ServiceConfig.Name
	recordHistory := params.ServiceConfig.HistoryEnabled()

	bodyBytes := readAndRestoreBody(req)

	outURL := fmt.Sprintf("%s/%s",
		strings.TrimSuffix(cfg.URL, "/"),
		strings.TrimPrefix(req.URL.Path[len(resourcePrefix):], "/"))

	if req.URL.RawQuery != "" {
		outURL += "?" + req.URL.RawQuery
	}

	log.Debug("Upstream request", "method", req.Method, "url", outURL)

	upReq, err := http.NewRequest(req.Method, outURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		log.Error("Failed to create request", "error", err)
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

	log.Info("Upstream request", "method", upReq.Method, "url", upReq.URL.String())

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
		return nil, &upstreamHTTPError{
			StatusCode:  statusCode,
			Body:        string(body),
			ContentType: resp.Header.Get("Content-Type"),
		}
	}

	log.Info("Received successful upstream response", "body", string(body))

	contentType := resp.Header.Get("Content-Type")

	if recordHistory {
		urlCopy := *req.URL
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), asyncWriteTimeout)
			defer cancel()
			history.Set(ctx, req.URL.Path, &http.Request{
				Method:     req.Method,
				URL:        &urlCopy,
				RemoteAddr: req.RemoteAddr,
				Body:       io.NopCloser(bytes.NewBuffer(bodyBytes)),
			}, &db.HistoryResponse{
				Data:           body,
				StatusCode:     statusCode,
				ContentType:    contentType,
				IsFromUpstream: true,
				UpstreamURL:    outURL,
			})
		}()
	}

	return &upstreamResponse{
		Body:        body,
		ContentType: contentType,
	}, nil
}
