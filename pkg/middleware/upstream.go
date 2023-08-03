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

	"github.com/cubahno/connexions/v2/internal/history"
	"github.com/sony/gobreaker/v2"
)

// CreateUpstreamRequestMiddleware returns a middleware that fetches data from an upstream service.
// If the upstream service fails, consequent requests will be blocked for a certain time.
func CreateUpstreamRequestMiddleware(params *Params) func(http.Handler) http.Handler {
	upstreamURL := ""
	cfg := params.ServiceConfig.Upstream

	if cfg != nil {
		upstreamURL = cfg.URL
	}

	cbSettings := gobreaker.Settings{
		Name: upstreamURL,
		// TODO: make it configurable
		Timeout: 10 * time.Minute,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			slog.Debug("Circuit breaker check",
				"url", upstreamURL,
				"requests", counts.Requests,
				"totalSuccesses", counts.TotalSuccesses,
				"totalFailures", counts.TotalFailures,
				"consecutiveSuccesses", counts.ConsecutiveSuccesses,
				"consecutiveFailures", counts.ConsecutiveFailures,
			)
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			isOpen := counts.Requests >= 3 && failureRatio >= 0.6
			if isOpen {
				slog.Info(fmt.Sprintf("Circuit breaker is open for %s, failure ratio: %v", cfg.URL, failureRatio))
			}
			return isOpen
		},
	}
	cb := gobreaker.NewCircuitBreaker[[]byte](cbSettings)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if cfg == nil || upstreamURL == "" {
				next.ServeHTTP(w, req)
				return
			}

			log.Println("Service has upstream service defined")

			response, err := cb.Execute(func() ([]byte, error) {
				return getUpstreamResponse(params, req)
			})

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

	resourcePrefix := "/" + params.ServiceConfig.Name
	rec := params.History.Set(params.ServiceConfig.Name, req.URL.Path, req, nil)

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

	slog.Info("Upstream request", "method", upReq.Method, "url", upReq.URL.String(), "headers", upReq.Header)

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

	historyResponse := &history.HistoryResponse{
		Data:           body,
		StatusCode:     statusCode,
		IsFromUpstream: true,
	}
	params.History.Set(params.ServiceConfig.Name, req.URL.Path, req, historyResponse)

	return body, nil
}
