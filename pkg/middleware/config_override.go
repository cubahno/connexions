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
	headerCacheRequests = "Cache-Requests"
	headerLatency       = "Latency"
	headerUpstreamURL   = "Upstream-Url"
)

// CreateConfigOverrideMiddleware creates a middleware that reads X-Cxs-* headers
// and temporarily overrides ServiceConfig values for the current request.
// Headers are case-insensitive. The original config is restored after the request completes.
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
