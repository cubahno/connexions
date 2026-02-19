// Package middleware provides HTTP middleware for connexions services.
package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cubahno/connexions/v2/pkg/config"
)

// Header prefix and names for per-request config overrides.
// Headers are case-insensitive (Go's http.Header canonicalizes them).
const (
	// headerPrefix is the prefix for all config override headers.
	headerPrefix = "X-Cxs-"

	// Supported header names (without prefix, canonicalized form)
	headerCacheRequests    = "Cache-Requests"
	headerValidateRequest  = "Validate-Request"
	headerValidateResponse = "Validate-Response"
	headerLatency          = "Latency"
	headerUpstreamURL      = "Upstream-Url"
)

// CreateConfigOverrideMiddleware creates a middleware that reads X-Cxs-* headers
// and temporarily overrides ServiceConfig values for the current request.
//
// This allows per-request configuration changes without modifying the service config file.
// Useful for testing, debugging, or special request handling.
//
// # Headers are case-insensitive
//
// Go's http.Header canonicalizes header names, so these are equivalent:
//   - x-cxs-cache-requests
//   - X-Cxs-Cache-Requests
//   - X-CXS-CACHE-REQUESTS
//
// # Supported headers
//
//   - X-Cxs-Cache-Requests: true|false
//     Enable or disable request caching for this request.
//
//   - X-Cxs-Validate-Request: true|false
//     Enable or disable request validation for this request.
//
//   - X-Cxs-Validate-Response: true|false
//     Enable or disable response validation for this request.
//
//   - X-Cxs-Latency: duration
//     Override latency for this request. Accepts Go duration format (e.g., "100ms", "1s", "2s500ms").
//
//   - X-Cxs-Upstream-Url: URL or empty
//     Override upstream URL for this request. Empty string disables upstream proxy.
//
// # Examples
//
//	# Disable caching
//	curl -H "X-Cxs-Cache-Requests: false" http://localhost:8080/petstore/pets
//
//	# Add 500ms latency
//	curl -H "X-Cxs-Latency: 500ms" http://localhost:8080/petstore/pets
//
//	# Disable upstream proxy (use mock response)
//	curl -H "X-Cxs-Upstream-Url: " http://localhost:8080/petstore/pets
//
//	# Enable response validation
//	curl -H "X-Cxs-Validate-Response: true" http://localhost:8080/petstore/pets
//
// The original config is restored after the request completes, even if the handler panics.
func CreateConfigOverrideMiddleware(params *Params) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			overrides := parseConfigOverrides(req.Header)
			if len(overrides) == 0 {
				next.ServeHTTP(w, req)
				return
			}

			// Save original config and restore after request
			originalConfig := params.ServiceConfig
			params.ServiceConfig = applyOverrides(originalConfig, overrides)
			defer func() {
				params.ServiceConfig = originalConfig
			}()

			next.ServeHTTP(w, req)
		})
	}
}

// configOverride represents a single config override from a header.
type configOverride struct {
	key   string
	value string
}

// parseConfigOverrides extracts X-Cxs-* headers from the request.
func parseConfigOverrides(headers http.Header) []configOverride {
	var overrides []configOverride

	for name, values := range headers {
		if !strings.HasPrefix(name, headerPrefix) {
			continue
		}
		if len(values) == 0 {
			continue
		}

		key := strings.TrimPrefix(name, headerPrefix)
		overrides = append(overrides, configOverride{
			key: key,

			// Use first value if multiple
			value: values[0],
		})
	}

	return overrides
}

// applyOverrides creates a shallow copy of the config with overrides applied.
func applyOverrides(original *config.ServiceConfig, overrides []configOverride) *config.ServiceConfig {
	if original == nil {
		return nil
	}

	// Create a shallow copy
	cfg := *original

	// Deep copy nested structs that we might modify
	if original.Cache != nil {
		cacheCopy := *original.Cache
		cfg.Cache = &cacheCopy
	}

	if original.Validate != nil {
		validateCopy := *original.Validate
		cfg.Validate = &validateCopy
	}

	if original.Upstream != nil {
		upstreamCopy := *original.Upstream
		cfg.Upstream = &upstreamCopy
	}

	for _, o := range overrides {
		applyOverride(&cfg, o)
	}

	return &cfg
}

// applyOverride applies a single override to the config.
func applyOverride(cfg *config.ServiceConfig, o configOverride) {
	switch o.key {
	case headerCacheRequests:
		if cfg.Cache == nil {
			cfg.Cache = config.NewCacheConfig()
		}
		if b, err := strconv.ParseBool(o.value); err == nil {
			cfg.Cache.Requests = b
		}

	case headerValidateRequest:
		if cfg.Validate == nil {
			cfg.Validate = config.NewValidateConfig()
		}
		if b, err := strconv.ParseBool(o.value); err == nil {
			cfg.Validate.Request = b
		}

	case headerValidateResponse:
		if cfg.Validate == nil {
			cfg.Validate = config.NewValidateConfig()
		}
		if b, err := strconv.ParseBool(o.value); err == nil {
			cfg.Validate.Response = b
		}

	case headerLatency:
		if d, err := time.ParseDuration(o.value); err == nil {
			cfg.Latency = d
		}

	case headerUpstreamURL:
		// Empty string means disable upstream
		if o.value == "" {
			cfg.Upstream = nil
		} else {
			if cfg.Upstream == nil {
				cfg.Upstream = &config.UpstreamConfig{}
			}
			cfg.Upstream.URL = o.value
		}
	}
}
