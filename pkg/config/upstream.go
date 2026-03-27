package config

import (
	"strconv"
	"strings"
	"time"
)

type UpstreamConfig struct {
	URL            string                `yaml:"url"`
	Timeout        time.Duration         `yaml:"timeout"`
	Headers        map[string]string     `yaml:"headers"`
	CircuitBreaker *CircuitBreakerConfig `yaml:"circuit-breaker"`

	// FailOn defines which upstream HTTP status codes should be returned immediately
	// to the client without falling back to the generator.
	// nil (omitted): uses default (400-499 except 401, 403). Set to empty list (fail-on: []) to disable.
	FailOn *HTTPStatusMatchConfig `yaml:"fail-on"`
}

// DefaultFailOnStatus is the default fail-on config applied when FailOn is nil.
// Most 4xx errors indicate client-side problems that the generator cannot fix.
// 401/403 are excluded because they typically indicate missing/invalid credentials
// in the proxy setup, not a real client error.
var DefaultFailOnStatus = HTTPStatusMatchConfig{
	{Range: "400-499", Except: []int{401, 403}},
}

// DefaultUpstreamTimeout defaults.
const (
	DefaultUpstreamTimeout = 5 * time.Second
)

// Circuit breaker defaults.
const (
	DefaultCBTimeout      = 60 * time.Second
	DefaultCBMaxRequests  = 1
	DefaultCBMinRequests  = 3
	DefaultCBFailureRatio = 0.6
)

// CircuitBreakerConfig configures the circuit breaker for upstream requests.
// If not set, circuit breaker is disabled.
type CircuitBreakerConfig struct {
	// Timeout is the period of the open state, after which the state becomes half-open.
	Timeout time.Duration `yaml:"timeout"`

	// MaxRequests is the maximum number of requests allowed in half-open state.
	MaxRequests uint32 `yaml:"max-requests"`

	// Interval is the cyclic period of the closed state to clear internal counts.
	Interval time.Duration `yaml:"interval"`

	// MinRequests is the minimum number of requests before evaluating failure ratio.
	MinRequests uint32 `yaml:"min-requests"`

	// FailureRatio is the failure ratio threshold to trip the circuit breaker (0.0-1.0).
	FailureRatio float64 `yaml:"failure-ratio"`

	// TripOnStatus defines which HTTP status codes count as circuit breaker failures.
	// If not set, all errors (>= 400) count as failures.
	TripOnStatus HTTPStatusMatchConfig `yaml:"trip-on-status"`
}

// WithDefaults returns a copy with default values applied for zero fields.
func (c *CircuitBreakerConfig) WithDefaults() *CircuitBreakerConfig {
	if c == nil {
		return &CircuitBreakerConfig{
			Timeout:      DefaultCBTimeout,
			MaxRequests:  DefaultCBMaxRequests,
			MinRequests:  DefaultCBMinRequests,
			FailureRatio: DefaultCBFailureRatio,
		}
	}

	result := *c
	if result.Timeout == 0 {
		result.Timeout = DefaultCBTimeout
	}
	if result.MaxRequests == 0 {
		result.MaxRequests = DefaultCBMaxRequests
	}
	if result.MinRequests == 0 {
		result.MinRequests = DefaultCBMinRequests
	}
	if result.FailureRatio == 0 {
		result.FailureRatio = DefaultCBFailureRatio
	}
	return &result
}

type HTTPStatusConfig struct {
	Exact  int    `yaml:"exact"`
	Range  string `yaml:"range"`
	Except []int  `yaml:"except"`
}

func (s *HTTPStatusConfig) Is(status int) bool {
	for _, ex := range s.Except {
		if ex == status {
			return false
		}
	}

	if s.Exact == status {
		return true
	}

	rangeParts := strings.Split(s.Range, "-")
	if len(rangeParts) != 2 {
		return false
	}

	lower, err1 := strconv.Atoi(rangeParts[0])
	upper, err2 := strconv.Atoi(rangeParts[1])
	if err1 == nil && err2 == nil && status >= lower && status <= upper {
		return true
	}

	return false
}

type HTTPStatusMatchConfig []HTTPStatusConfig

func (ss HTTPStatusMatchConfig) Is(status int) bool {
	for _, s := range ss {
		if s.Is(status) {
			return true
		}
	}

	return false
}
