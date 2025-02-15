package config

import (
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ServiceConfig defines the configuration for a particular service.
// Latency is the single latency to add to the response.
// Latencies are the map of percentiles latencies.
// Latencies not used in the services API, only when endpoint queried directly:
//
//	p50: 20ms
//	p99: 100ms
//
// If only 1 latency needed, set it with `p100` key.
// Errors is the error config with the percentiles as keys and http status codes as values.
// Contexts is the list of contexts to use for replacements.
// It is a map of context name defined either in the UI or filename without extension.
// You can refer to the name when building aliases.
// ParseConfig is the config for parsing the OpenAPI spec.
// Validate is the validation config.
// It is used to validate the request and/or response outside the Services API.
// Cache is the cache config.
type ServiceConfig struct {
	Upstream    *UpstreamConfig          `koanf:"upstream" yaml:"upstream"`
	Latency     time.Duration            `koanf:"latency" yaml:"latency"`
	Latencies   map[string]time.Duration `koanf:"latencies" yaml:"latencies"`
	Errors      map[string]int           `koanf:"errors" yaml:"errors"`
	Contexts    []map[string]string      `koanf:"contexts" yaml:"contexts"`
	ParseConfig *ParseConfig             `koanf:"parseConfig" yaml:"parseConfig"`
	Validate    *ServiceValidateConfig   `koanf:"validate" yaml:"validate"`
	Middleware  *MiddlewareConfig        `koanf:"middleware" yaml:"middleware"`
	Cache       *ServiceCacheConfig      `koanf:"cache" yaml:"cache"`

	latencies []*KeyValue[int, time.Duration]
	errors    []*KeyValue[int, int]
}

// MiddlewareConfig defines the middleware configuration for a service.
// BeforeHandler is the list of middleware to run before the handler.
// AfterHandler is the list of middleware to run after the handler.
// If any of the middleware returns an error or response,
// the request will be stopped and the response will be returned.
type MiddlewareConfig struct {
	BeforeHandler []string `koanf:"beforeHandler" yaml:"beforeHandler"`
	AfterHandler  []string `koanf:"afterHandler" yaml:"afterHandler"`
}

func NewServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		Errors:      make(map[string]int),
		Latencies:   make(map[string]time.Duration),
		ParseConfig: NewParseConfig(),
		Validate:    NewServiceValidateConfig(),
		Cache:       NewServiceCacheConfig(),
	}
}

func (s *ServiceConfig) ParseLatencies() []*KeyValue[int, time.Duration] {
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

// GetLatency returns the latency.
func (s *ServiceConfig) GetLatency() time.Duration {
	latencies := s.latencies
	if len(s.latencies) == 0 && s.Latencies != nil {
		latencies = s.ParseLatencies()
	}

	if len(latencies) == 0 {
		return s.Latency
	}

	rnd := rand.Intn(100) + 1
	for _, latencyKV := range latencies {
		if rnd <= latencyKV.Key {
			return latencyKV.Value
		}
	}

	return 0
}

func (s *ServiceConfig) ParseErrors() []*KeyValue[int, int] {
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

// GetError returns the error based on the percentiles:
//
//	random number is generated between 1 and 100 to simulate the percentile.
//
// If no errors are defined, it returns 0.
func (s *ServiceConfig) GetError() int {
	errors := s.errors
	// this is just needed for tests.
	// TODO: improve this.
	if len(errors) == 0 && s.Errors != nil {
		errors = s.ParseErrors()
	}

	if len(errors) == 0 {
		return 0
	}

	rnd := rand.Intn(100) + 1
	for _, errorKV := range errors {
		if rnd <= errorKV.Key {
			return errorKV.Value
		}
	}

	return 0
}

// ServiceValidateConfig defines the validation configuration for a service.
// Request is a flag whether to validate the request.
// Default: true
// Response is a flag whether to validate the response.
// Default: false
type ServiceValidateConfig struct {
	Request  bool `koanf:"request" yaml:"request"`
	Response bool `koanf:"response" yaml:"response"`
}

func NewServiceValidateConfig() *ServiceValidateConfig {
	return &ServiceValidateConfig{}
}

// ServiceCacheConfig defines the cache configuration for a service.
// Avoids multiple schema parsing by caching the parsed schema.
// Default: true
type ServiceCacheConfig struct {
	Schema      bool `koanf:"schema" yaml:"schema"`
	GetRequests bool `koanf:"getRequests" yaml:"getRequests"`
}

// NewServiceCacheConfig creates a new ServiceCacheConfig with default values.
func NewServiceCacheConfig() *ServiceCacheConfig {
	return &ServiceCacheConfig{
		Schema:      true,
		GetRequests: true,
	}
}
