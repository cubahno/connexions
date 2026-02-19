package config

import (
	"fmt"
	"testing"
	"time"

	assert2 "github.com/stretchr/testify/assert"
)

func TestHTTPStatusConfig_Is(t *testing.T) {
	assert := assert2.New(t)
	type testcase struct {
		received int
		cfg      *HTTPStatusConfig
		expected bool
	}

	testcases := []testcase{
		{received: 400, cfg: &HTTPStatusConfig{400, ""}, expected: true},
		{received: 401, cfg: &HTTPStatusConfig{400, ""}, expected: false},
		{received: 400, cfg: &HTTPStatusConfig{0, "400-404"}, expected: true},
		{received: 400, cfg: &HTTPStatusConfig{0, "500-600"}, expected: false},
	}

	for i, tc := range testcases {
		t.Run(fmt.Sprintf("case %d", i+1), func(t *testing.T) {
			assert.Equal(tc.expected, tc.cfg.Is(tc.received))
		})
	}

	t.Run("invalid range", func(t *testing.T) {
		cfg := &HTTPStatusConfig{0, "400-"}
		assert.False(cfg.Is(401))
	})

	t.Run("range boundaries", func(t *testing.T) {
		cfg := &HTTPStatusConfig{0, "400-404"}
		testCases := []struct {
			status   int
			expected bool
		}{
			{400, true},  // lower boundary
			{404, true},  // upper boundary
			{402, true},  // middle
			{399, false}, // below range
			{405, false}, // above range
		}
		for _, tc := range testCases {
			assert.Equal(tc.expected, cfg.Is(tc.status))
		}
	})

	t.Run("both exact and range can match", func(t *testing.T) {
		cfg := &HTTPStatusConfig{400, "500-600"}
		testCases := []struct {
			status   int
			expected bool
		}{
			{400, true},  // matches exact
			{500, true},  // matches range
			{550, true},  // matches range
			{450, false}, // matches neither
		}
		for _, tc := range testCases {
			assert.Equal(tc.expected, cfg.Is(tc.status))
		}
	})

	t.Run("no match", func(t *testing.T) {
		cfg := &HTTPStatusConfig{0, ""}
		assert.False(cfg.Is(400))
	})

	t.Run("invalid range format - single number", func(t *testing.T) {
		cfg := &HTTPStatusConfig{0, "400"}
		assert.False(cfg.Is(400))
	})

	t.Run("invalid range format - non-numeric", func(t *testing.T) {
		cfg := &HTTPStatusConfig{0, "abc-def"}
		assert.False(cfg.Is(400))
	})
}

func TestHttpStatusFailOnConfig_Is(t *testing.T) {
	assert := assert2.New(t)

	t.Run("single", func(t *testing.T) {
		cfg := HttpStatusFailOnConfig{
			{400, ""},
		}
		assert.True(cfg.Is(400))
		assert.False(cfg.Is(401))
	})

	t.Run("range", func(t *testing.T) {
		cfg := HttpStatusFailOnConfig{
			{0, "400-404"},
		}
		assert.True(cfg.Is(400))
		assert.True(cfg.Is(404))
		assert.False(cfg.Is(405))
	})

	t.Run("multiple", func(t *testing.T) {
		cfg := HttpStatusFailOnConfig{
			{400, ""},
			{0, "500-600"},
		}
		assert.True(cfg.Is(400))
		assert.True(cfg.Is(500))
		assert.True(cfg.Is(501))
		assert.True(cfg.Is(600))
		assert.False(cfg.Is(401))
	})

	t.Run("empty config", func(t *testing.T) {
		cfg := HttpStatusFailOnConfig{}
		assert.False(cfg.Is(400))
		assert.False(cfg.Is(500))
	})

	t.Run("overlapping ranges", func(t *testing.T) {
		cfg := HttpStatusFailOnConfig{
			{0, "400-450"},
			{0, "440-500"},
		}
		assert.True(cfg.Is(400))
		assert.True(cfg.Is(445)) // in both ranges
		assert.True(cfg.Is(500))
		assert.False(cfg.Is(350))
		assert.False(cfg.Is(550))
	})
}

func TestCircuitBreakerConfig(t *testing.T) {
	assert := assert2.New(t)

	t.Run("zero values are valid", func(t *testing.T) {
		cfg := &CircuitBreakerConfig{}
		assert.Zero(cfg.Timeout)
		assert.Zero(cfg.MaxRequests)
		assert.Zero(cfg.Interval)
		assert.Zero(cfg.MinRequests)
		assert.Zero(cfg.FailureRatio)
	})

	t.Run("all fields can be set", func(t *testing.T) {
		cfg := &CircuitBreakerConfig{
			Timeout:      60,
			MaxRequests:  5,
			Interval:     30,
			MinRequests:  3,
			FailureRatio: 0.5,
		}
		assert.Equal(60, int(cfg.Timeout))
		assert.Equal(uint32(5), cfg.MaxRequests)
		assert.Equal(30, int(cfg.Interval))
		assert.Equal(uint32(3), cfg.MinRequests)
		assert.Equal(0.5, cfg.FailureRatio)
	})

	t.Run("WithDefaults returns defaults when zero", func(t *testing.T) {
		cfg := (&CircuitBreakerConfig{}).WithDefaults()
		assert.Equal(DefaultCBTimeout, cfg.Timeout)
		assert.Equal(uint32(DefaultCBMaxRequests), cfg.MaxRequests)
		assert.Equal(uint32(DefaultCBMinRequests), cfg.MinRequests)
		assert.Equal(DefaultCBFailureRatio, cfg.FailureRatio)
	})

	t.Run("WithDefaults preserves set values", func(t *testing.T) {
		cfg := (&CircuitBreakerConfig{
			Timeout:      30 * time.Second,
			MaxRequests:  5,
			MinRequests:  10,
			FailureRatio: 0.8,
		}).WithDefaults()
		assert.Equal(30*time.Second, cfg.Timeout)
		assert.Equal(uint32(5), cfg.MaxRequests)
		assert.Equal(uint32(10), cfg.MinRequests)
		assert.Equal(0.8, cfg.FailureRatio)
	})

	t.Run("WithDefaults on nil returns defaults", func(t *testing.T) {
		var cfg *CircuitBreakerConfig
		result := cfg.WithDefaults()
		assert.Equal(DefaultCBTimeout, result.Timeout)
		assert.Equal(uint32(DefaultCBMaxRequests), result.MaxRequests)
		assert.Equal(uint32(DefaultCBMinRequests), result.MinRequests)
		assert.Equal(DefaultCBFailureRatio, result.FailureRatio)
	})
}

func TestUpstreamConfig_WithCircuitBreaker(t *testing.T) {
	assert := assert2.New(t)

	t.Run("upstream config without circuit breaker", func(t *testing.T) {
		cfg := &UpstreamConfig{
			URL: "https://api.example.com",
		}
		assert.Nil(cfg.CircuitBreaker)
	})

	t.Run("upstream config with circuit breaker", func(t *testing.T) {
		cfg := &UpstreamConfig{
			URL: "https://api.example.com",
			CircuitBreaker: &CircuitBreakerConfig{
				Timeout:      60,
				MinRequests:  3,
				FailureRatio: 0.6,
			},
		}
		assert.NotNil(cfg.CircuitBreaker)
		assert.Equal(60, int(cfg.CircuitBreaker.Timeout))
		assert.Equal(uint32(3), cfg.CircuitBreaker.MinRequests)
		assert.Equal(0.6, cfg.CircuitBreaker.FailureRatio)
	})
}
