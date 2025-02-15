package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewServiceConfig(t *testing.T) {
	res := NewServiceConfig()

	expected := &ServiceConfig{
		Errors:    make(map[string]int),
		Latencies: make(map[string]time.Duration),
		ParseConfig: &ParseConfig{
			MaxLevels: 0,
		},
		Validate: &ServiceValidateConfig{},
		Cache: &ServiceCacheConfig{
			Schema:      true,
			GetRequests: true,
		},
	}

	assert.Equal(t, expected, res)
}

func TestServiceConfig_ParseLatencies(t *testing.T) {
	s := &ServiceConfig{
		Latencies: map[string]time.Duration{
			"p1": 100 * time.Millisecond,
			"p2": 200 * time.Millisecond,
			"p3": 300 * time.Millisecond,
		},
	}

	expected := []*KeyValue[int, time.Duration]{
		{Key: 1, Value: 100 * time.Millisecond},
		{Key: 2, Value: 200 * time.Millisecond},
		{Key: 3, Value: 300 * time.Millisecond},
	}

	res := s.ParseLatencies()
	assert.Equal(t, expected, res)
}

func TestServiceConfig_GetLatency(t *testing.T) {
	s := &ServiceConfig{
		Latency: 50 * time.Millisecond,
		latencies: []*KeyValue[int, time.Duration]{
			{Key: 1, Value: 100 * time.Millisecond},
			{Key: 2, Value: 200 * time.Millisecond},
			{Key: 3, Value: 300 * time.Millisecond},
		},
	}

	res := s.GetLatency()

	assert.Contains(t, []time.Duration{
		0,
		50 * time.Millisecond,
		100 * time.Millisecond,
		200 * time.Millisecond, 300 * time.Millisecond}, res)
}

func TestServiceConfig_ParseErrors(t *testing.T) {
	s := &ServiceConfig{
		Errors: map[string]int{
			"p1": 100,
			"p2": 200,
			"p3": 300,
		},
	}

	res := s.ParseErrors()

	expected := []*KeyValue[int, int]{
		{Key: 1, Value: 100},
		{Key: 2, Value: 200},
		{Key: 3, Value: 300},
	}

	assert.Equal(t, expected, res)
}

func TestServiceConfig_GetError(t *testing.T) {
	s := &ServiceConfig{
		errors: []*KeyValue[int, int]{
			{Key: 1, Value: 200},
			{Key: 2, Value: 400},
			{Key: 3, Value: 500},
		},
	}

	res := s.GetError()

	assert.Contains(t, []int{0, 200, 400, 500}, res)
}

func TestNewServiceCacheConfig(t *testing.T) {
	res := NewServiceCacheConfig()

	assert.True(t, res.GetRequests)
	assert.True(t, res.Schema)
}
