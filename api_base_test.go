package connexions

import (
	assert2 "github.com/stretchr/testify/assert"
	"net/http"
	"testing"
	"time"
)

func TestHandleErrorAndLatency(t *testing.T) {
	assert := assert2.New(t)

	t.Run("latency>0", func(t *testing.T) {
		t1 := time.Now()
		svcConfig := &ServiceConfig{
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
		svcConfig := &ServiceConfig{
			Latency: 100 * time.Millisecond,
			Errors: &ServiceError{
				Codes: map[int]int{
					400: 100,
				},
				Chance: 0,
			},
		}

		res := HandleErrorAndLatency(svcConfig, nil)
		assert.False(res)
	})

	t.Run("errors-chance-100", func(t *testing.T) {
		svcConfig := &ServiceConfig{
			Latency: 100 * time.Millisecond,
			Errors: &ServiceError{
				Codes: map[int]int{
					400: 100,
				},
				Chance: 100,
			},
		}
		w := newBufferedResponseWriter()

		res := HandleErrorAndLatency(svcConfig, w)
		assert.True(res)
		assert.Equal("Random config error", string(w.buf))
		assert.Equal(http.StatusBadRequest, w.statusCode)
	})

	t.Run("errors-chance-100-without-codes", func(t *testing.T) {
		svcConfig := &ServiceConfig{
			Latency: 100 * time.Millisecond,
			Errors: &ServiceError{
				Chance: 100,
			},
		}
		w := newBufferedResponseWriter()

		res := HandleErrorAndLatency(svcConfig, w)
		assert.True(res)
		assert.Equal("Random config error", string(w.buf))
		assert.Equal(http.StatusInternalServerError, w.statusCode)
	})
}
