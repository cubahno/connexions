package config

import (
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.yaml.in/yaml/v4"
)

type KeyValue[K, V any] struct {
	Key   K
	Value V
}

// ServiceConfig defines the configuration for a service.
// Name is the optional name of the service.
// Upstream is the upstream configuration.
// Latency is the default latency for the service.
// Latencies is a map of percentiles to latencies.
// Errors is a map of percentiles to error codes.
// Validate is the validation configuration.
// Cache is the cache configuration.
// ResourcesPrefix is the prefix for helper routes outside OpenAPI spec.
// SpecOptions allows OpenAPI spec simplifications for code generation.
type ServiceConfig struct {
	Name            string                   `yaml:"name,omitempty"`
	Upstream        *UpstreamConfig          `yaml:"upstream,omitempty"`
	Latency         time.Duration            `yaml:"latency,omitempty"`
	Latencies       map[string]time.Duration `yaml:"latencies,omitempty"`
	Errors          map[string]int           `yaml:"errors,omitempty"`
	Cache           *CacheConfig             `yaml:"cache,omitempty"`
	ResourcesPrefix string                   `yaml:"resources-prefix,omitempty"`
	SpecOptions     *SpecOptions             `yaml:"spec,omitempty"`
	Extra           map[string]any           `yaml:"extra,omitempty"`

	latencies []*KeyValue[int, time.Duration]
	errors    []*KeyValue[int, int]
}

// NewServiceConfig creates a new ServiceConfig with default values.
func NewServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		Errors:      make(map[string]int),
		Latencies:   make(map[string]time.Duration),
		Cache:       NewCacheConfig(),
		SpecOptions: NewSpecOptions(),
		Extra:       make(map[string]any),
	}
}

func NewServiceConfigFromBytes(bts []byte) (*ServiceConfig, error) {
	res := NewServiceConfig()
	err := yaml.Unmarshal(bts, res)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling service config: %w", err)
	}

	// Fill any nil fields with defaults (in case YAML didn't specify them)
	res.WithDefaults()

	return res, nil
}

// WithDefaults fills nil properties with default values from NewServiceConfig.
func (s *ServiceConfig) WithDefaults() *ServiceConfig {
	defaults := NewServiceConfig()

	// Fill nil pointer fields
	if s.Cache == nil {
		s.Cache = defaults.Cache
	}

	if s.SpecOptions == nil {
		s.SpecOptions = defaults.SpecOptions
	}

	// Fill nil map fields
	if s.Errors == nil {
		s.Errors = defaults.Errors
	}

	if s.Latencies == nil {
		s.Latencies = defaults.Latencies
	}

	if s.Extra == nil {
		s.Extra = defaults.Extra
	}

	// Fill empty string fields with defaults
	if s.ResourcesPrefix == "" {
		s.ResourcesPrefix = defaults.ResourcesPrefix
	}

	// Parse latencies and errors if they haven't been parsed yet
	if s.latencies == nil && len(s.Latencies) > 0 {
		s.latencies = s.parseLatencies()
	}

	if s.errors == nil && len(s.Errors) > 0 {
		s.errors = s.parseErrors()
	}

	return s
}

// OverwriteWith overwrites fields in s with non-nil/non-empty values from other.
// This is useful for merging configurations where other takes precedence.
func (s *ServiceConfig) OverwriteWith(other *ServiceConfig) *ServiceConfig {
	if other == nil {
		return s
	}

	// Overwrite string fields if not empty
	if other.Name != "" {
		s.Name = other.Name
	}

	if other.ResourcesPrefix != "" {
		s.ResourcesPrefix = other.ResourcesPrefix
	}

	// Overwrite pointer fields if not nil
	if other.Upstream != nil {
		s.Upstream = other.Upstream
	}

	if other.Cache != nil {
		s.Cache = other.Cache
	}

	// Overwrite duration if set (non-zero)
	if other.Latency != 0 {
		s.Latency = other.Latency
	}

	// Overwrite map fields if not nil (merge maps)
	if other.Latencies != nil {
		if s.Latencies == nil {
			s.Latencies = make(map[string]time.Duration)
		}
		for k, v := range other.Latencies {
			s.Latencies[k] = v
		}
		// Re-parse latencies after merge
		s.latencies = s.parseLatencies()
	}

	if other.Errors != nil {
		if s.Errors == nil {
			s.Errors = make(map[string]int)
		}
		for k, v := range other.Errors {
			s.Errors[k] = v
		}
		// Re-parse errors after merge
		s.errors = s.parseErrors()
	}

	if other.SpecOptions != nil {
		s.SpecOptions = other.SpecOptions
	}

	if other.Extra != nil {
		if s.Extra == nil {
			s.Extra = make(map[string]any)
		}
		for k, v := range other.Extra {
			s.Extra[k] = v
		}
	}

	return s
}

// GetLatency returns the latency.
func (s *ServiceConfig) GetLatency() time.Duration {
	if len(s.latencies) == 0 {
		return s.Latency
	}

	rnd := rand.Intn(100) + 1
	for _, latencyKV := range s.latencies {
		if rnd <= latencyKV.Key {
			return latencyKV.Value
		}
	}

	return 0
}

// GetError returns the error based on the percentiles:
//
//	random number is generated between 1 and 100 to simulate the percentile.
//
// If no errors are defined, it returns 0.
func (s *ServiceConfig) GetError() int {
	if len(s.errors) == 0 {
		return 0
	}

	rnd := rand.Intn(100) + 1
	for _, errorKV := range s.errors {
		if rnd <= errorKV.Key {
			return errorKV.Value
		}
	}

	return 0
}

func (s *ServiceConfig) parseLatencies() []*KeyValue[int, time.Duration] {
	latencies := make([]*KeyValue[int, time.Duration], 0)
	for k, v := range s.Latencies {
		if strings.HasPrefix(k, "p") {
			kNum, err := strconv.Atoi(strings.TrimPrefix(k, "p"))
			if err == nil {
				res := &KeyValue[int, time.Duration]{Key: kNum, Value: v}
				latencies = append(latencies, res)
			}
		}
	}

	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i].Key < latencies[j].Key
	})
	return latencies
}

func (s *ServiceConfig) parseErrors() []*KeyValue[int, int] {
	errors := make([]*KeyValue[int, int], 0)
	for k, v := range s.Errors {
		if strings.HasPrefix(k, "p") {
			kNum, err := strconv.Atoi(strings.TrimPrefix(k, "p"))
			if err == nil {
				res := &KeyValue[int, int]{Key: kNum, Value: v}
				errors = append(errors, res)
			}
		}
	}
	sort.Slice(errors, func(i, j int) bool {
		return errors[i].Key < errors[j].Key
	})
	return errors
}

// CacheConfig defines the cache configuration for a service.
// Requests is a flag whether to cache GET requests.
type CacheConfig struct {
	Requests bool `yaml:"requests"`
}

// NewCacheConfig creates a new CacheConfig with default values.
func NewCacheConfig() *CacheConfig {
	return &CacheConfig{
		Requests: true,
	}
}

// SpecOptions allows OpenAPI spec simplifications for code generation.
// These simplifications are particularly helpful for enormous schemas that would
// otherwise generate unwieldy code.
//
// Simplifications include:
//   - Removal of extra union elements (anyOf/oneOf/allOf)
//   - Keeping only a reasonable number of optional properties instead of all
//
// LazyLoad enables on-demand parsing of operations. When true, operations are
// parsed only when first accessed and cached for subsequent requests. This
// significantly speeds up server startup for large specs (e.g., Stripe with 500+ endpoints).
//
// Example usage in YAML:
//
//	spec:
//	  lazyLoad: true    # Parse operations on-demand instead of at startup
//	  simplify: true
//	  optional-properties:
//	    min: 5        # Keep exactly 5 optional properties (when min == max)
//	    max: 5
//	    # OR
//	    min: 2        # Keep random number between 2-8 optional properties
//	    max: 8
type SpecOptions struct {
	LazyLoad           bool                `yaml:"lazyLoad"`
	Simplify           bool                `yaml:"simplify"`
	OptionalProperties *OptionalProperties `yaml:"optional-properties"`
}

func NewSpecOptions() *SpecOptions {
	return &SpecOptions{
		LazyLoad:           true,
		Simplify:           false,
		OptionalProperties: NewDefaultOptionalProperties(),
	}
}

// OptionalProperties controls how many optional properties to keep in generated types.
// This helps reduce the size of generated code for schemas with many optional fields.
//
// Min and Max specify the range of optional properties to keep:
//   - If Min == Max, keeps exactly that many optional properties
//   - If Min < Max, keeps a random number between Min and Max (inclusive)
//
// Default: Min=5, Max=5 (keeps exactly 5 optional properties)
type OptionalProperties struct {
	Min int `yaml:"min"`
	Max int `yaml:"max"`
}

// NewDefaultOptionalProperties creates a new OptionalProperties with default values.
func NewDefaultOptionalProperties() *OptionalProperties {
	return &OptionalProperties{
		Min: 5,
		Max: 5,
	}
}
