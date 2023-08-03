package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestNewServiceConfig(t *testing.T) {
	t.Run("Creates config with default values", func(t *testing.T) {
		cfg := NewServiceConfig()

		assert.NotNil(t, cfg)
		assert.NotNil(t, cfg.Errors)
		assert.NotNil(t, cfg.Latencies)
		assert.NotNil(t, cfg.Validate)
		assert.NotNil(t, cfg.Cache)
		assert.Empty(t, cfg.Errors)
		assert.Empty(t, cfg.Latencies)
	})
}

func TestNewServiceConfigFromBytes(t *testing.T) {
	t.Run("Parses valid YAML config", func(t *testing.T) {
		yamlData := []byte(`
latency: 100ms
latencies:
  p50: 50ms
  p90: 100ms
  p99: 200ms
errors:
  p10: 400
  p20: 500
validate:
  request: true
  response: false
cache:
  requests: true
`)

		cfg, err := NewServiceConfigFromBytes(yamlData)
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.Equal(t, 100*time.Millisecond, cfg.Latency)
		assert.Len(t, cfg.Latencies, 3)
		assert.Len(t, cfg.Errors, 2)
		assert.True(t, cfg.Validate.Request)
		assert.False(t, cfg.Validate.Response)
		assert.True(t, cfg.Cache.Requests)
	})

	t.Run("Returns error for invalid YAML", func(t *testing.T) {
		yamlData := []byte(`invalid: yaml: data: [`)

		cfg, err := NewServiceConfigFromBytes(yamlData)
		assert.Error(t, err)
		assert.Nil(t, cfg)
	})

	t.Run("Parses empty config", func(t *testing.T) {
		yamlData := []byte(``)

		cfg, err := NewServiceConfigFromBytes(yamlData)
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
	})

	t.Run("Fills defaults for missing fields", func(t *testing.T) {
		yamlData := []byte(`
name: test-service
latency: 100ms
`)

		cfg, err := NewServiceConfigFromBytes(yamlData)
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.Equal(t, "test-service", cfg.Name)
		assert.Equal(t, 100*time.Millisecond, cfg.Latency)

		// These should be filled with defaults even though not in YAML
		assert.NotNil(t, cfg.Validate)
		assert.True(t, cfg.Validate.Request)
		assert.False(t, cfg.Validate.Response)

		assert.NotNil(t, cfg.Cache)
		assert.True(t, cfg.Cache.Requests)

		assert.NotNil(t, cfg.Errors)
		assert.NotNil(t, cfg.Latencies)
	})

	t.Run("Parses latencies and errors automatically", func(t *testing.T) {
		yamlData := []byte(`
latencies:
  p50: 50ms
  p90: 100ms
errors:
  p10: 400
  p20: 500
`)

		cfg, err := NewServiceConfigFromBytes(yamlData)
		assert.NoError(t, err)
		assert.NotNil(t, cfg)

		// Internal parsed slices should be populated
		assert.NotNil(t, cfg.latencies)
		assert.Len(t, cfg.latencies, 2)
		assert.NotNil(t, cfg.errors)
		assert.Len(t, cfg.errors, 2)
	})
}

func TestServiceConfig_GetLatency(t *testing.T) {
	t.Run("Returns default latency when no percentiles defined", func(t *testing.T) {
		cfg := &ServiceConfig{
			Latency:   100 * time.Millisecond,
			latencies: []*KeyValue[int, time.Duration]{},
		}

		latency := cfg.GetLatency()
		assert.Equal(t, 100*time.Millisecond, latency)
	})

	t.Run("Returns latency based on percentiles", func(t *testing.T) {
		cfg := &ServiceConfig{
			Latency: 100 * time.Millisecond,
			latencies: []*KeyValue[int, time.Duration]{
				{Key: 50, Value: 50 * time.Millisecond},
				{Key: 90, Value: 100 * time.Millisecond},
				{Key: 99, Value: 200 * time.Millisecond},
			},
		}

		// Run multiple times to test randomness
		latencies := make(map[time.Duration]bool)
		for i := 0; i < 100; i++ {
			latency := cfg.GetLatency()
			latencies[latency] = true
			// Should be one of the defined latencies or 0
			assert.Contains(t, []time.Duration{
				0,
				50 * time.Millisecond,
				100 * time.Millisecond,
				200 * time.Millisecond,
			}, latency)
		}
	})
}

func TestServiceConfig_GetError(t *testing.T) {
	t.Run("Returns 0 when no errors defined", func(t *testing.T) {
		cfg := &ServiceConfig{
			errors: []*KeyValue[int, int]{},
		}

		errorCode := cfg.GetError()
		assert.Equal(t, 0, errorCode)
	})

	t.Run("Returns error based on percentiles", func(t *testing.T) {
		cfg := &ServiceConfig{
			errors: []*KeyValue[int, int]{
				{Key: 10, Value: 400},
				{Key: 20, Value: 500},
				{Key: 30, Value: 503},
			},
		}

		// Run multiple times to test randomness
		errors := make(map[int]bool)
		for i := 0; i < 100; i++ {
			errorCode := cfg.GetError()
			errors[errorCode] = true
			// Should be one of the defined errors or 0
			assert.Contains(t, []int{0, 400, 500, 503}, errorCode)
		}
	})
}

func TestServiceConfig_parseLatencies(t *testing.T) {
	t.Run("Parses percentile latencies", func(t *testing.T) {
		cfg := &ServiceConfig{
			Latencies: map[string]time.Duration{
				"p50": 50 * time.Millisecond,
				"p90": 100 * time.Millisecond,
				"p99": 200 * time.Millisecond,
			},
		}

		latencies := cfg.parseLatencies()
		expected := []*KeyValue[int, time.Duration]{
			{Key: 50, Value: 50 * time.Millisecond},
			{Key: 90, Value: 100 * time.Millisecond},
			{Key: 99, Value: 200 * time.Millisecond},
		}
		assert.Equal(t, expected, latencies)
	})

	t.Run("Sorts latencies by percentile", func(t *testing.T) {
		cfg := &ServiceConfig{
			Latencies: map[string]time.Duration{
				"p99": 200 * time.Millisecond,
				"p50": 50 * time.Millisecond,
				"p90": 100 * time.Millisecond,
			},
		}

		latencies := cfg.parseLatencies()
		expected := []*KeyValue[int, time.Duration]{
			{Key: 50, Value: 50 * time.Millisecond},
			{Key: 90, Value: 100 * time.Millisecond},
			{Key: 99, Value: 200 * time.Millisecond},
		}
		assert.Equal(t, expected, latencies)
	})

	t.Run("Ignores non-percentile keys", func(t *testing.T) {
		cfg := &ServiceConfig{
			Latencies: map[string]time.Duration{
				"p50":     50 * time.Millisecond,
				"default": 100 * time.Millisecond,
				"max":     200 * time.Millisecond,
			},
		}

		latencies := cfg.parseLatencies()
		expected := []*KeyValue[int, time.Duration]{
			{Key: 50, Value: 50 * time.Millisecond},
		}
		assert.Equal(t, expected, latencies)
	})

	t.Run("Ignores invalid percentile values", func(t *testing.T) {
		cfg := &ServiceConfig{
			Latencies: map[string]time.Duration{
				"p50":  50 * time.Millisecond,
				"pabc": 100 * time.Millisecond,
				"p":    200 * time.Millisecond,
			},
		}

		latencies := cfg.parseLatencies()
		expected := []*KeyValue[int, time.Duration]{
			{Key: 50, Value: 50 * time.Millisecond},
		}
		assert.Equal(t, expected, latencies)
	})

	t.Run("Returns empty slice for empty latencies", func(t *testing.T) {
		cfg := &ServiceConfig{
			Latencies: map[string]time.Duration{},
		}

		latencies := cfg.parseLatencies()
		assert.Empty(t, latencies)
	})
}

func TestServiceConfig_parseErrors(t *testing.T) {
	t.Run("Parses percentile errors", func(t *testing.T) {
		cfg := &ServiceConfig{
			Errors: map[string]int{
				"p10": 400,
				"p20": 500,
				"p30": 503,
			},
		}

		errors := cfg.parseErrors()
		expected := []*KeyValue[int, int]{
			{Key: 10, Value: 400},
			{Key: 20, Value: 500},
			{Key: 30, Value: 503},
		}
		assert.Equal(t, expected, errors)
	})

	t.Run("Sorts errors by percentile", func(t *testing.T) {
		cfg := &ServiceConfig{
			Errors: map[string]int{
				"p30": 503,
				"p10": 400,
				"p20": 500,
			},
		}

		errors := cfg.parseErrors()
		expected := []*KeyValue[int, int]{
			{Key: 10, Value: 400},
			{Key: 20, Value: 500},
			{Key: 30, Value: 503},
		}
		assert.Equal(t, expected, errors)
	})

	t.Run("Ignores non-percentile keys", func(t *testing.T) {
		cfg := &ServiceConfig{
			Errors: map[string]int{
				"p10":     400,
				"default": 500,
				"max":     503,
			},
		}

		errors := cfg.parseErrors()
		expected := []*KeyValue[int, int]{
			{Key: 10, Value: 400},
		}
		assert.Equal(t, expected, errors)
	})

	t.Run("Ignores invalid percentile values", func(t *testing.T) {
		cfg := &ServiceConfig{
			Errors: map[string]int{
				"p10":  400,
				"pabc": 500,
				"p":    503,
			},
		}

		errors := cfg.parseErrors()
		expected := []*KeyValue[int, int]{
			{Key: 10, Value: 400},
		}
		assert.Equal(t, expected, errors)
	})

	t.Run("Returns empty slice for empty errors", func(t *testing.T) {
		cfg := &ServiceConfig{
			Errors: map[string]int{},
		}

		errors := cfg.parseErrors()
		assert.Empty(t, errors)
	})
}

func TestNewValidateConfig(t *testing.T) {
	t.Run("Creates validate config with default values", func(t *testing.T) {
		cfg := NewValidateConfig()

		assert.NotNil(t, cfg)
		assert.True(t, cfg.Request)
		assert.False(t, cfg.Response)
	})
}

func TestNewCacheConfig(t *testing.T) {
	t.Run("Creates cache config with default values", func(t *testing.T) {
		cfg := NewCacheConfig()

		assert.NotNil(t, cfg)
		assert.True(t, cfg.Requests)
	})
}

func TestServiceConfig_WithDefaults(t *testing.T) {
	t.Run("Fills nil Validate with defaults", func(t *testing.T) {
		cfg := &ServiceConfig{
			Validate: nil,
		}

		result := cfg.WithDefaults()

		assert.NotNil(t, result.Validate)
		assert.True(t, result.Validate.Request)
		assert.False(t, result.Validate.Response)
	})

	t.Run("Fills nil Cache with defaults", func(t *testing.T) {
		cfg := &ServiceConfig{
			Cache: nil,
		}

		result := cfg.WithDefaults()

		assert.NotNil(t, result.Cache)
		assert.True(t, result.Cache.Requests)
	})

	t.Run("Fills nil Errors map with defaults", func(t *testing.T) {
		cfg := &ServiceConfig{
			Errors: nil,
		}

		result := cfg.WithDefaults()

		assert.NotNil(t, result.Errors)
		assert.Empty(t, result.Errors)
	})

	t.Run("Fills nil Latencies map with defaults", func(t *testing.T) {
		cfg := &ServiceConfig{
			Latencies: nil,
		}

		result := cfg.WithDefaults()

		assert.NotNil(t, result.Latencies)
		assert.Empty(t, result.Latencies)
	})

	t.Run("Does not override existing Validate", func(t *testing.T) {
		customValidate := &ValidateConfig{
			Request:  false,
			Response: true,
		}
		cfg := &ServiceConfig{
			Validate: customValidate,
		}

		result := cfg.WithDefaults()

		assert.Equal(t, customValidate, result.Validate)
		assert.False(t, result.Validate.Request)
		assert.True(t, result.Validate.Response)
	})

	t.Run("Does not override existing Cache", func(t *testing.T) {
		customCache := &CacheConfig{
			Requests: false,
		}
		cfg := &ServiceConfig{
			Cache: customCache,
		}

		result := cfg.WithDefaults()

		assert.Equal(t, customCache, result.Cache)
		assert.False(t, result.Cache.Requests)
	})

	t.Run("Does not override existing Errors", func(t *testing.T) {
		customErrors := map[string]int{
			"p10": 400,
			"p20": 500,
		}
		cfg := &ServiceConfig{
			Errors: customErrors,
		}

		result := cfg.WithDefaults()

		assert.Equal(t, customErrors, result.Errors)
		assert.Len(t, result.Errors, 2)
	})

	t.Run("Does not override existing Latencies", func(t *testing.T) {
		customLatencies := map[string]time.Duration{
			"p50": 50 * time.Millisecond,
			"p90": 100 * time.Millisecond,
		}
		cfg := &ServiceConfig{
			Latencies: customLatencies,
		}

		result := cfg.WithDefaults()

		assert.Equal(t, customLatencies, result.Latencies)
		assert.Len(t, result.Latencies, 2)
	})

	t.Run("Does not override existing ResourcesPrefix", func(t *testing.T) {
		customPrefix := "/custom-prefix"
		cfg := &ServiceConfig{
			ResourcesPrefix: customPrefix,
		}

		result := cfg.WithDefaults()

		assert.Equal(t, customPrefix, result.ResourcesPrefix)
	})

	t.Run("Parses latencies when map is provided", func(t *testing.T) {
		cfg := &ServiceConfig{
			Latencies: map[string]time.Duration{
				"p50": 50 * time.Millisecond,
				"p90": 100 * time.Millisecond,
			},
		}

		result := cfg.WithDefaults()

		assert.NotNil(t, result.latencies)
		assert.Len(t, result.latencies, 2)
	})

	t.Run("Parses errors when map is provided", func(t *testing.T) {
		cfg := &ServiceConfig{
			Errors: map[string]int{
				"p10": 400,
				"p20": 500,
			},
		}

		result := cfg.WithDefaults()

		assert.NotNil(t, result.errors)
		assert.Len(t, result.errors, 2)
	})

	t.Run("Fills all nil fields at once", func(t *testing.T) {
		cfg := &ServiceConfig{
			Name:            "test-service",
			Validate:        nil,
			Cache:           nil,
			Errors:          nil,
			Latencies:       nil,
			ResourcesPrefix: "",
		}

		result := cfg.WithDefaults()

		assert.Equal(t, "test-service", result.Name)
		assert.NotNil(t, result.Validate)
		assert.NotNil(t, result.Cache)
		assert.NotNil(t, result.Errors)
		assert.NotNil(t, result.Latencies)
	})

	t.Run("Returns same instance (modifies in place)", func(t *testing.T) {
		cfg := &ServiceConfig{
			Validate: nil,
		}

		result := cfg.WithDefaults()

		assert.Equal(t, cfg, result)
		assert.NotNil(t, cfg.Validate)
	})

	t.Run("Does not override Upstream (not set in defaults)", func(t *testing.T) {
		customUpstream := &UpstreamConfig{
			URL: "http://example.com",
		}
		cfg := &ServiceConfig{
			Upstream: customUpstream,
		}

		result := cfg.WithDefaults()

		assert.Equal(t, customUpstream, result.Upstream)
	})

	t.Run("Leaves Upstream nil if not set", func(t *testing.T) {
		cfg := &ServiceConfig{
			Upstream: nil,
		}

		result := cfg.WithDefaults()

		assert.Nil(t, result.Upstream)
	})
}

func TestServiceConfig_OverwriteWith(t *testing.T) {
	t.Run("Overwrites Name when other has non-empty Name", func(t *testing.T) {
		cfg := &ServiceConfig{
			Name: "original",
		}
		other := &ServiceConfig{
			Name: "overwritten",
		}

		result := cfg.OverwriteWith(other)

		assert.Equal(t, "overwritten", result.Name)
	})

	t.Run("Does not overwrite Name when other has empty Name", func(t *testing.T) {
		cfg := &ServiceConfig{
			Name: "original",
		}
		other := &ServiceConfig{
			Name: "",
		}

		result := cfg.OverwriteWith(other)

		assert.Equal(t, "original", result.Name)
	})

	t.Run("Overwrites ResourcesPrefix when other has non-empty value", func(t *testing.T) {
		cfg := &ServiceConfig{
			ResourcesPrefix: "/original",
		}
		other := &ServiceConfig{
			ResourcesPrefix: "/overwritten",
		}

		result := cfg.OverwriteWith(other)

		assert.Equal(t, "/overwritten", result.ResourcesPrefix)
	})

	t.Run("Overwrites Upstream when other has non-nil Upstream", func(t *testing.T) {
		cfg := &ServiceConfig{
			Upstream: &UpstreamConfig{
				URL: "http://original.com",
			},
		}
		other := &ServiceConfig{
			Upstream: &UpstreamConfig{
				URL: "http://overwritten.com",
			},
		}

		result := cfg.OverwriteWith(other)

		assert.Equal(t, "http://overwritten.com", result.Upstream.URL)
	})

	t.Run("Does not overwrite Upstream when other has nil Upstream", func(t *testing.T) {
		originalUpstream := &UpstreamConfig{
			URL: "http://original.com",
		}
		cfg := &ServiceConfig{
			Upstream: originalUpstream,
		}
		other := &ServiceConfig{
			Upstream: nil,
		}

		result := cfg.OverwriteWith(other)

		assert.Equal(t, originalUpstream, result.Upstream)
	})

	t.Run("Overwrites Validate when other has non-nil Validate", func(t *testing.T) {
		cfg := &ServiceConfig{
			Validate: &ValidateConfig{
				Request:  true,
				Response: false,
			},
		}
		other := &ServiceConfig{
			Validate: &ValidateConfig{
				Request:  false,
				Response: true,
			},
		}

		result := cfg.OverwriteWith(other)

		assert.False(t, result.Validate.Request)
		assert.True(t, result.Validate.Response)
	})

	t.Run("Overwrites Cache when other has non-nil Cache", func(t *testing.T) {
		cfg := &ServiceConfig{
			Cache: &CacheConfig{
				Requests: true,
			},
		}
		other := &ServiceConfig{
			Cache: &CacheConfig{
				Requests: false,
			},
		}

		result := cfg.OverwriteWith(other)

		assert.False(t, result.Cache.Requests)
	})

	t.Run("Overwrites Latency when other has non-zero Latency", func(t *testing.T) {
		cfg := &ServiceConfig{
			Latency: 100 * time.Millisecond,
		}
		other := &ServiceConfig{
			Latency: 200 * time.Millisecond,
		}

		result := cfg.OverwriteWith(other)

		assert.Equal(t, 200*time.Millisecond, result.Latency)
	})

	t.Run("Does not overwrite Latency when other has zero Latency", func(t *testing.T) {
		cfg := &ServiceConfig{
			Latency: 100 * time.Millisecond,
		}
		other := &ServiceConfig{
			Latency: 0,
		}

		result := cfg.OverwriteWith(other)

		assert.Equal(t, 100*time.Millisecond, result.Latency)
	})

	t.Run("Merges Latencies maps", func(t *testing.T) {
		cfg := &ServiceConfig{
			Latencies: map[string]time.Duration{
				"p50": 50 * time.Millisecond,
				"p90": 100 * time.Millisecond,
			},
		}
		other := &ServiceConfig{
			Latencies: map[string]time.Duration{
				"p90": 150 * time.Millisecond, // Override existing
				"p99": 200 * time.Millisecond, // Add new
			},
		}

		result := cfg.OverwriteWith(other)

		assert.Len(t, result.Latencies, 3)
		assert.Equal(t, 50*time.Millisecond, result.Latencies["p50"])
		assert.Equal(t, 150*time.Millisecond, result.Latencies["p90"]) // Overwritten
		assert.Equal(t, 200*time.Millisecond, result.Latencies["p99"]) // Added
	})

	t.Run("Merges Errors maps", func(t *testing.T) {
		cfg := &ServiceConfig{
			Errors: map[string]int{
				"p10": 400,
				"p20": 500,
			},
		}
		other := &ServiceConfig{
			Errors: map[string]int{
				"p20": 503, // Override existing
				"p30": 502, // Add new
			},
		}

		result := cfg.OverwriteWith(other)

		assert.Len(t, result.Errors, 3)
		assert.Equal(t, 400, result.Errors["p10"])
		assert.Equal(t, 503, result.Errors["p20"]) // Overwritten
		assert.Equal(t, 502, result.Errors["p30"]) // Added
	})

	t.Run("Creates Latencies map if nil before merge", func(t *testing.T) {
		cfg := &ServiceConfig{
			Latencies: nil,
		}
		other := &ServiceConfig{
			Latencies: map[string]time.Duration{
				"p50": 50 * time.Millisecond,
			},
		}

		result := cfg.OverwriteWith(other)

		assert.NotNil(t, result.Latencies)
		assert.Len(t, result.Latencies, 1)
		assert.Equal(t, 50*time.Millisecond, result.Latencies["p50"])
	})

	t.Run("Creates Errors map if nil before merge", func(t *testing.T) {
		cfg := &ServiceConfig{
			Errors: nil,
		}
		other := &ServiceConfig{
			Errors: map[string]int{
				"p10": 400,
			},
		}

		result := cfg.OverwriteWith(other)

		assert.NotNil(t, result.Errors)
		assert.Len(t, result.Errors, 1)
		assert.Equal(t, 400, result.Errors["p10"])
	})

	t.Run("Re-parses latencies after merge", func(t *testing.T) {
		cfg := &ServiceConfig{
			Latencies: map[string]time.Duration{
				"p50": 50 * time.Millisecond,
			},
			latencies: []*KeyValue[int, time.Duration]{
				{Key: 50, Value: 50 * time.Millisecond},
			},
		}
		other := &ServiceConfig{
			Latencies: map[string]time.Duration{
				"p90": 100 * time.Millisecond,
			},
		}

		result := cfg.OverwriteWith(other)

		assert.NotNil(t, result.latencies)
		assert.Len(t, result.latencies, 2) // Should have both p50 and p90
	})

	t.Run("Re-parses errors after merge", func(t *testing.T) {
		cfg := &ServiceConfig{
			Errors: map[string]int{
				"p10": 400,
			},
			errors: []*KeyValue[int, int]{
				{Key: 10, Value: 400},
			},
		}
		other := &ServiceConfig{
			Errors: map[string]int{
				"p20": 500,
			},
		}

		result := cfg.OverwriteWith(other)

		assert.NotNil(t, result.errors)
		assert.Len(t, result.errors, 2) // Should have both p10 and p20
	})

	t.Run("Returns same instance (modifies in place)", func(t *testing.T) {
		cfg := &ServiceConfig{
			Name: "original",
		}
		other := &ServiceConfig{
			Name: "overwritten",
		}

		result := cfg.OverwriteWith(other)

		assert.Equal(t, cfg, result)
		assert.Equal(t, "overwritten", cfg.Name)
	})

	t.Run("Handles nil other gracefully", func(t *testing.T) {
		cfg := &ServiceConfig{
			Name: "original",
		}

		result := cfg.OverwriteWith(nil)

		assert.Equal(t, cfg, result)
		assert.Equal(t, "original", result.Name)
	})

	t.Run("Overwrites multiple fields at once", func(t *testing.T) {
		cfg := &ServiceConfig{
			Name:            "original",
			Latency:         100 * time.Millisecond,
			ResourcesPrefix: "/original",
			Validate: &ValidateConfig{
				Request:  true,
				Response: false,
			},
		}
		other := &ServiceConfig{
			Name:            "overwritten",
			Latency:         200 * time.Millisecond,
			ResourcesPrefix: "/overwritten",
			Validate: &ValidateConfig{
				Request:  false,
				Response: true,
			},
			Cache: &CacheConfig{
				Requests: false,
			},
		}

		result := cfg.OverwriteWith(other)

		assert.Equal(t, "overwritten", result.Name)
		assert.Equal(t, 200*time.Millisecond, result.Latency)
		assert.Equal(t, "/overwritten", result.ResourcesPrefix)
		assert.False(t, result.Validate.Request)
		assert.True(t, result.Validate.Response)
		assert.NotNil(t, result.Cache)
		assert.False(t, result.Cache.Requests)
	})

	t.Run("Overwrites Generate when other has non-nil Generate", func(t *testing.T) {
		cfg := &ServiceConfig{
			Generate: nil,
		}
		other := &ServiceConfig{
			Generate: &GenerateConfig{
				Server: &struct{}{},
			},
		}

		result := cfg.OverwriteWith(other)

		assert.NotNil(t, result.Generate)
		assert.NotNil(t, result.Generate.Server)
	})

	t.Run("Overwrites SpecOptions when other has non-nil SpecOptions", func(t *testing.T) {
		cfg := &ServiceConfig{
			SpecOptions: nil,
		}
		other := &ServiceConfig{
			SpecOptions: &SpecOptions{
				LazyLoad: true,
				Simplify: true,
			},
		}

		result := cfg.OverwriteWith(other)

		assert.NotNil(t, result.SpecOptions)
		assert.True(t, result.SpecOptions.LazyLoad)
		assert.True(t, result.SpecOptions.Simplify)
	})
}

func TestOptionalProperties_UnmarshalYAML(t *testing.T) {
	t.Run("parses min and max", func(t *testing.T) {
		yamlData := []byte(`
min: 1
max: 6
`)
		var opts OptionalProperties
		err := yaml.Unmarshal(yamlData, &opts)

		assert.NoError(t, err)
		assert.Equal(t, 1, opts.Min)
		assert.Equal(t, 6, opts.Max)
	})

	t.Run("parses only min", func(t *testing.T) {
		yamlData := []byte(`
min: 5
`)
		var opts OptionalProperties
		err := yaml.Unmarshal(yamlData, &opts)

		assert.NoError(t, err)
		assert.Equal(t, 5, opts.Min)
		assert.Equal(t, 0, opts.Max)
	})

	t.Run("parses only max", func(t *testing.T) {
		yamlData := []byte(`
max: 10
`)
		var opts OptionalProperties
		err := yaml.Unmarshal(yamlData, &opts)

		assert.NoError(t, err)
		assert.Equal(t, 0, opts.Min)
		assert.Equal(t, 10, opts.Max)
	})
}
