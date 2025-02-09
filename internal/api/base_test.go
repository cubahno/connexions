//go:build !integration

package api

import (
	"net/http"
	"testing"
	"time"

	"github.com/cubahno/connexions/internal/config"
	assert2 "github.com/stretchr/testify/assert"
)

func TestNewBaseHandler(t *testing.T) {
	assert := assert2.New(t)

	t.Run("base-handler", func(t *testing.T) {
		h := NewBaseHandler()
		assert.NotNil(h)
	})
}

func TestHandleErrorAndLatency(t *testing.T) {
	assert := assert2.New(t)

	t.Run("latency>0", func(t *testing.T) {
		t1 := time.Now()
		svcConfig := &config.ServiceConfig{
			Latency: 100 * time.Millisecond,
		}

		res := HandleErrorAndLatency(svcConfig, nil)
		assert.False(res)

		t2 := time.Now()
		if t2.Sub(t1) < 100*time.Millisecond {
			t.Errorf("Expected latency of 100ms, got %s", t2.Sub(t1))
		}
	})

	t.Run("errors-chance-0", func(t *testing.T) {
		svcConfig := &config.ServiceConfig{
			Latency: 100 * time.Millisecond,
			Errors: map[string]int{
				"p0": 400,
			},
		}

		res := HandleErrorAndLatency(svcConfig, nil)
		assert.False(res)
	})

	t.Run("errors-chance-100", func(t *testing.T) {
		svcConfig := &config.ServiceConfig{
			Latency: 100 * time.Millisecond,
			Errors: map[string]int{
				"p100": 400,
			},
		}
		w := NewBufferedResponseWriter()

		res := HandleErrorAndLatency(svcConfig, w)
		assert.True(res)
		assert.Equal("configured service error: 400", string(w.buf))
		assert.Equal(http.StatusBadRequest, w.statusCode)
	})

	t.Run("errors-chance-100-without-codes", func(t *testing.T) {
		svcConfig := &config.ServiceConfig{
			Latency: 100 * time.Millisecond,
			Errors: map[string]int{
				"p100": 500,
			},
		}
		w := NewBufferedResponseWriter()

		res := HandleErrorAndLatency(svcConfig, w)
		assert.True(res)
		assert.Equal("configured service error: 500", string(w.buf))
		assert.Equal(http.StatusInternalServerError, w.statusCode)
	})
}
