package config

import (
	"strconv"
	"strings"
	"time"
)

type UpstreamConfig struct {
	URL            string                `yaml:"url"`
	Headers        map[string]string     `yaml:"headers"`
	FailOn         *UpstreamFailOnConfig `yaml:"fail-on"`
	CircuitBreaker *CircuitBreakerConfig `yaml:"circuit-breaker"`
}

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
	// Default: 60s
	Timeout time.Duration `yaml:"timeout"`

	// MaxRequests is the maximum number of requests allowed in half-open state.
	// Default: 1
	MaxRequests uint32 `yaml:"max-requests"`

	// Interval is the cyclic period of the closed state to clear internal counts.
	// Default: 0 (never clears)
	Interval time.Duration `yaml:"interval"`

	// MinRequests is the minimum number of requests before evaluating failure ratio.
	// Default: 3
	MinRequests uint32 `yaml:"min-requests"`

	// FailureRatio is the failure ratio threshold to trip the circuit breaker (0.0-1.0).
	// Default: 0.6
	FailureRatio float64 `yaml:"failure-ratio"`
}

// GetTimeout returns Timeout with default applied.
func (c *CircuitBreakerConfig) GetTimeout() time.Duration {
	if c.Timeout == 0 {
		return DefaultCBTimeout
	}
	return c.Timeout
}

// GetMaxRequests returns MaxRequests with default applied.
func (c *CircuitBreakerConfig) GetMaxRequests() uint32 {
	if c.MaxRequests == 0 {
		return DefaultCBMaxRequests
	}
	return c.MaxRequests
}

// GetMinRequests returns MinRequests with default applied.
func (c *CircuitBreakerConfig) GetMinRequests() uint32 {
	if c.MinRequests == 0 {
		return DefaultCBMinRequests
	}
	return c.MinRequests
}

// GetFailureRatio returns FailureRatio with default applied.
func (c *CircuitBreakerConfig) GetFailureRatio() float64 {
	if c.FailureRatio == 0 {
		return DefaultCBFailureRatio
	}
	return c.FailureRatio
}

type HTTPStatusConfig struct {
	Exact int    `yaml:"exact"`
	Range string `yaml:"range"`
}

func (s *HTTPStatusConfig) Is(status int) bool {
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

type HttpStatusFailOnConfig []HTTPStatusConfig

func (ss HttpStatusFailOnConfig) Is(status int) bool {
	for _, s := range ss {
		if s.Is(status) {
			return true
		}
	}

	return false
}

type UpstreamFailOnConfig struct {
	TimeOut    time.Duration          `yaml:"timeout"`
	HTTPStatus HttpStatusFailOnConfig `yaml:"http-status"`
}
