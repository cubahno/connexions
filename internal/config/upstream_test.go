package config

import (
	"fmt"
	"testing"

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
}
