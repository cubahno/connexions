package api

import (
	"bytes"
	"fmt"
	"github.com/cubahno/connexions/config"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sony/gobreaker/v2"
	"io"
	"log"
	"net/http"
	"os"
	"plugin"
	"strings"
	"time"
)

type MiddlewareParams struct {
	ServiceConfig  *config.ServiceConfig
	Service        string
	Resource       string
	ResourcePrefix string
	Plugin         *plugin.Plugin
}

// responseWriter is a custom response writer that captures the response body
type responseWriter struct {
	http.ResponseWriter
	body       *bytes.Buffer
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write intercepts the response and writes to a buffer
func (rw *responseWriter) Write(b []byte) (int, error) {
	return rw.body.Write(b)
}

// ConditionalLoggingMiddleware is a middleware that conditionally can disable logger.
// For example, in tests or when fetching static files.
func ConditionalLoggingMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		logger := middleware.DefaultLogger(next)
		disableLogger := os.Getenv("DISABLE_LOGGER") == "true"

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if disableLogger || strings.HasPrefix(r.URL.Path, cfg.App.HomeURL) {
				next.ServeHTTP(w, r)
				return
			}
			logger.ServeHTTP(w, r)
		})
	}
}

func CreateUpstreamRequestMiddleware(params *MiddlewareParams) func(http.Handler) http.Handler {
	timeOut := 30 * time.Second
	cfg := params.ServiceConfig.Upstream
	failOn := cfg.FailOn
	if failOn.TimeOut > 0 {
		timeOut = failOn.TimeOut
	}

	cbSettings := gobreaker.Settings{
		Name:    cfg.URL,
		Timeout: timeOut,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			isOpen := counts.Requests >= 3 && failureRatio >= 0.6
			if isOpen {
				log.Printf("Circuit breaker is open for %s, failure ratio: %v", cfg.URL, failureRatio)
			}
			return isOpen
		},
	}
	cb := gobreaker.NewCircuitBreaker[[]byte](cbSettings)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if cfg == nil || cfg.URL == "" {
				next.ServeHTTP(w, req)
				return
			}

			log.Println("Service has upstream service defined")

			response, err := cb.Execute(func() ([]byte, error) {
				return getUpstreamResponse(params, req)
			})
			// If an upstream service returns a response, write it and return immediately
			if response != nil {
				_, _ = w.Write(response)
				return
			}

			if err != nil {
				log.Printf("Error fetching upstream service %s: %s", cfg.URL, err)
			}

			// Proceed to the next handler if no upstream service matched
			next.ServeHTTP(w, req)
		})
	}
}

// CreateResponseMiddleware is a method on the Router to create a middleware
func CreateResponseMiddleware(params *MiddlewareParams) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// Create a responseWriter to capture the response.
			// default to 200 status code
			rw := &responseWriter{
				ResponseWriter: w,
				body:           new(bytes.Buffer),
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(rw, req)

			modifiedResponse, err := handleResponseCallback(params, req, rw.body.Bytes(), rw.statusCode)
			if err != nil {
				http.Error(w, fmt.Sprintf("error handling callback: %v", err), http.StatusInternalServerError)
				return
			}

			_, _ = w.Write(modifiedResponse)
		})
	}
}

func getUpstreamResponse(params *MiddlewareParams, req *http.Request) ([]byte, error) {
	cfg := params.ServiceConfig.Upstream
	upstreamHTTPOptions := cfg.HTTPOptions
	if upstreamHTTPOptions == nil {
		upstreamHTTPOptions = &config.UpstreamHTTPOptionsConfig{
			Headers: map[string]string{},
		}
	}

	failOn := cfg.FailOn
	resource := params.Resource
	resourcePrefix := params.ResourcePrefix
	p := params.Plugin

	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		// Reset the body so it can be read again
		req.Body.Close()
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	upReq, err := http.NewRequest(req.Method, cfg.URL+req.URL.Path[len(resourcePrefix):], bytes.NewBuffer(bodyBytes))
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}
	for name, values := range req.Header {
		for _, value := range values {
			upReq.Header.Add(name, value)
		}
	}
	upReq.Header.Set("User-Agent", "Connexions/0.1")

	// custom request transformer
	if p != nil && upstreamHTTPOptions.RequestTransformer != "" {
		fn := upstreamHTTPOptions.RequestTransformer
		symbol, err := p.Lookup(fn)
		if err != nil {
			return nil, fmt.Errorf("request transformer function not found for %s", fn)
		}

		transformer, ok := symbol.(func(string, *http.Request) (*http.Request, error))
		if !ok {
			return nil, fmt.Errorf("invalid request transformer function signature for %s", fn)
		}

		upReq, err = transformer(resource, upReq)
		if err != nil {
			return nil, err
		}
		log.Println("Request transformed")
	}

	log.Printf("Request Method: %s, URL: %s, Headers: %+v", upReq.Method, upReq.URL.String(), upReq.Header)
	resp, err := http.DefaultClient.Do(upReq)
	if err != nil {
		return nil, fmt.Errorf("error calling upstream service %s: %s", upReq.URL.String(), err)
	}

	statusCode := resp.StatusCode

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response from upstream service %s: %s", upReq.URL, err)
	}

	if failOn != nil && failOn.HTTPStatus != nil && failOn.HTTPStatus.Is(statusCode) {
		return nil, fmt.Errorf("failOn condition met for status code: %d", statusCode)
	}

	return body, nil
}

// handleResponseCallback uses router's internal data to trigger callbacks
func handleResponseCallback(params *MiddlewareParams, req *http.Request, response []byte, statusCode int) ([]byte, error) {
	funcName := params.ServiceConfig.ResponseTransformer
	if params.Plugin == nil || funcName == "" {
		return response, nil
	}

	service := params.Service
	resource := params.Resource
	log.Println("Response callback", service, req.Method, req.URL.Path)

	if statusCode >= http.StatusBadRequest {
		log.Println("Response status code is not 2xx, skipping callback")
		return response, nil
	}

	// Lookup the user-defined function
	symbol, err := params.Plugin.Lookup(funcName)
	if err != nil {
		log.Printf("service %s does not have any callback function %s", service, funcName)
		return response, nil
	}

	// Assert the function's type
	callback, ok := symbol.(func(string, *http.Request, []byte) ([]byte, error))
	if !ok {
		return nil, fmt.Errorf("invalid callback function signature for %s", funcName)
	}

	return callback(resource, req, response)
}
