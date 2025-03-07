package api

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"plugin"
	"strings"
	"time"

	"github.com/cubahno/connexions/internal/config"
	"github.com/cubahno/connexions_plugin"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sony/gobreaker/v2"
)

type MiddlewareParams struct {
	ServiceConfig  *config.ServiceConfig
	Service        string
	Resource       string
	ResourcePrefix string
	Plugin         *plugin.Plugin
	history        *CurrentRequestStorage
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

// CreateCacheRequestMiddleware returns a middleware that checks if GET request is cached in history.
// Depends on service settings.
// Service timeouts still apply.
func CreateCacheRequestMiddleware(params *MiddlewareParams) func(http.Handler) http.Handler {
	cfg := params.ServiceConfig

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// Check if the service has caching enabled and it is GET request
			if req.Method != http.MethodGet || cfg == nil || cfg.Cache == nil || !cfg.Cache.GetRequests {
				next.ServeHTTP(w, req)
				return
			}

			res, exists := params.history.Get(req)
			if !exists {
				next.ServeHTTP(w, req)
				return
			}

			log.Printf("Cache hit for %s", req.URL.Path)

			latency := cfg.GetLatency()
			if latency > 0 {
				time.Sleep(latency)
			}

			response := res.Response
			w.WriteHeader(response.StatusCode)
			_, _ = w.Write(response.Data)
		})
	}
}

// CreateBeforeHandlerMiddleware returns a middleware that modifies the request.
// Function names must be defined in the service configuration middleware config and refer to compiled plugin.
func CreateBeforeHandlerMiddleware(params *MiddlewareParams) func(http.Handler) http.Handler {
	cfg := params.ServiceConfig
	p := params.Plugin

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if p == nil || cfg == nil || cfg.Middleware == nil || len(cfg.Middleware.BeforeHandler) == 0 {
				next.ServeHTTP(w, req)
				return
			}

			record, ok := params.history.Get(req)
			if !ok {
				record = params.history.Set(params.Service, params.Resource, req, nil)
			}

			for _, fn := range cfg.Middleware.BeforeHandler {
				log.Printf("Service has before handler middleware %s defined", fn)

				symbol, err := p.Lookup(fn)
				if err != nil {
					log.Printf("middleware function not found for %s\n", fn)
					next.ServeHTTP(w, req)
					return
				}

				mw, ok := symbol.(func(*connexions_plugin.RequestedResource) ([]byte, error))
				if !ok {
					log.Printf("invalid middleware function signature for %s\n", fn)
					next.ServeHTTP(w, req)
					return
				}

				response, err := mw(record)
				if err != nil {
					log.Printf("Error calling middleware: %v", err)
					next.ServeHTTP(w, req)
					return
				}

				if response != nil {
					log.Printf("middleware %s applied", fn)
					_, _ = w.Write(response)
					params.history.SetResponse(req, &connexions_plugin.HistoryResponse{
						Data:       response,
						StatusCode: http.StatusOK,
					})
					return
				}

			}

			next.ServeHTTP(w, req)
		})
	}
}

// CreateUpstreamRequestMiddleware returns a middleware that fetches data from an upstream service.
// If the upstream service fails, consequent requests will be blocked for a certain time.
func CreateUpstreamRequestMiddleware(params *MiddlewareParams) func(http.Handler) http.Handler {
	timeOut := 30 * time.Second
	upstreamURL := ""
	cfg := params.ServiceConfig.Upstream

	if cfg != nil && cfg.FailOn != nil && cfg.FailOn.TimeOut > 0 {
		timeOut = cfg.FailOn.TimeOut
	}

	if cfg != nil {
		upstreamURL = cfg.URL
	}

	cbSettings := gobreaker.Settings{
		Name:    upstreamURL,
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
			if cfg == nil || upstreamURL == "" {
				next.ServeHTTP(w, req)
				return
			}

			log.Println("Service has upstream service defined")

			response, err := cb.Execute(func() ([]byte, error) {
				return getUpstreamResponse(params, req)
			})
			// If an upstream service returns a response, write it and return immediately
			if response != nil {
				// history was set already
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

// CreateAfterHandlerMiddleware is a method on the Router to create a middleware
func CreateAfterHandlerMiddleware(params *MiddlewareParams) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// Create a responseWriter to capture the response.
			// default to 200 status code
			rw := &responseWriter{
				ResponseWriter: w,
				body:           new(bytes.Buffer),
				statusCode:     http.StatusOK,
			}
			rw.Header().Set("Content-Type", "application/json")

			next.ServeHTTP(rw, req)

			params.history.SetResponse(req, &connexions_plugin.HistoryResponse{
				Data:       rw.body.Bytes(),
				StatusCode: rw.statusCode,
			})

			modifiedResponse, err := handleResponseMiddleware(params, req)
			if err != nil {
				// TODO: decide if this is an error, maybe return the original response
				http.Error(w, fmt.Sprintf("error handling callback: %v", err), http.StatusInternalServerError)
				return
			}

			params.history.SetResponse(req, &connexions_plugin.HistoryResponse{
				Data:       modifiedResponse,
				StatusCode: rw.statusCode,
			})

			_, _ = w.Write(modifiedResponse)
		})
	}
}

func getUpstreamResponse(params *MiddlewareParams, req *http.Request) ([]byte, error) {
	cfg := params.ServiceConfig.Upstream

	failOn := cfg.FailOn
	resource := params.Resource
	resourcePrefix := params.ResourcePrefix

	// TODO: add time out to the upstream config
	client := http.Client{
		Timeout: 5 * time.Second,
	}

	rec := params.history.Set(params.Service, resource, req, nil)

	bodyBytes := rec.Body
	if bodyBytes != nil {
		// Reset the body so it can be read again
		req.Body = io.NopCloser(bytes.NewBuffer(rec.Body))
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
	upReq.Header.Set("User-Agent", "Connexions/1.0")

	log.Printf("Request Method: %s, URL: %s, Headers: %+v", upReq.Method, upReq.URL.String(), upReq.Header)
	resp, err := client.Do(upReq)
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

	if statusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("upstream response failed with status code %d, body: %s", statusCode, string(body))
	}

	log.Printf("received successful upstream response: %s", string(body))

	historyResponse := &connexions_plugin.HistoryResponse{
		Data:           body,
		StatusCode:     statusCode,
		IsFromUpstream: true,
	}
	params.history.Set(params.Service, resource, req, historyResponse)

	return body, nil
}

// handleResponseMiddleware uses router's internal data to trigger middleware
func handleResponseMiddleware(params *MiddlewareParams, request *http.Request) ([]byte, error) {
	record, _ := params.history.Get(request)
	response := record.Response
	svcCfg := params.ServiceConfig

	// nothing to transform, return original
	if params.Plugin == nil || svcCfg.Middleware == nil || len(svcCfg.Middleware.AfterHandler) == 0 {
		return response.Data, nil
	}

	service := params.Service
	log.Println("After handler middleware", service, request.Method, request.URL.String())

	if response.StatusCode >= http.StatusBadRequest {
		log.Println("Response status code is not 2xx, skipping middleware")
		return response.Data, nil
	}

	for _, funcName := range svcCfg.Middleware.AfterHandler {
		// Lookup the user-defined function
		symbol, err := params.Plugin.Lookup(funcName)
		if err != nil {
			log.Printf("service %s does not have any callback function %s", service, funcName)
			return response.Data, nil
		}

		// Assert the function's type
		mw, ok := symbol.(func(*connexions_plugin.RequestedResource) ([]byte, error))
		if !ok {
			return nil, fmt.Errorf("invalid middleware function signature for %s", funcName)
		}

		res, err := mw(record)
		if err != nil {
			return nil, fmt.Errorf("error calling middleware: %w", err)
		}

		// mw not applied
		if res == nil {
			continue
		}

		record = &connexions_plugin.RequestedResource{
			Resource: record.Resource,
			Body:     record.Body,
			Request:  record.Request,
			Response: &connexions_plugin.HistoryResponse{
				Data:       res,
				StatusCode: http.StatusOK,
			},
		}
	}

	return record.Response.Data, nil
}
