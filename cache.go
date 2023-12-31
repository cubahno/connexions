package connexions

import (
	"github.com/cubahno/connexions/config"
	"github.com/cubahno/connexions/openapi"
	"log"
	"sync"
)

// CacheStorage is an interface that describes a cache storage.
type CacheStorage interface {
	Set(key string, value any) error
	Get(key string) (any, bool)
}

// MemoryStorage is a cache storage that stores data in memory.
type MemoryStorage struct {
	data map[string]any
	mu   sync.Mutex
}

// NewMemoryStorage creates a new MemoryStorage instance.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		data: make(map[string]any),
	}
}

// Set sets the value for the given key.
func (s *MemoryStorage) Set(key string, value any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = value
	return nil
}

// Get returns the value for the given key.
func (s *MemoryStorage) Get(key string) (any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	value, ok := s.data[key]
	return value, ok
}

// SchemaWithContentType is a schema with a content type.
// It is used to cache the result of GetRequestBody and wrap 2 values together.
type SchemaWithContentType struct {
	Schema      *openapi.Schema
	ContentType string
}

// CacheOperationAdapter is an adapter that caches the result of the wrapped Operation.
// Implements Operation interface.
type CacheOperationAdapter struct {
	service      string
	operation    openapi.Operation
	cacheStorage CacheStorage
	mu           sync.Mutex
}

// NewCacheOperationAdapter creates a new CacheOperationAdapter instance.
func NewCacheOperationAdapter(service string, operation openapi.Operation, storage CacheStorage) openapi.Operation {
	return &CacheOperationAdapter{
		service:      service,
		operation:    operation,
		cacheStorage: storage,
	}
}

// WithParseConfig sets the ParseConfig for the Operation.
func (a *CacheOperationAdapter) WithParseConfig(parseConfig *config.ParseConfig) openapi.Operation {
	a.operation.WithParseConfig(parseConfig)
	return a
}

// ID returns the ID of the Operation.
func (a *CacheOperationAdapter) ID() string {
	return a.operation.ID()
}

// GetParameters returns the parameters for the Operation.
func (a *CacheOperationAdapter) GetParameters() openapi.Parameters {
	a.mu.Lock()
	defer a.mu.Unlock()

	key := a.key("parameters")
	if cached, ok := a.cacheStorage.Get(key); ok {
		return cached.(openapi.Parameters)
	}

	value := a.operation.GetParameters()
	if err := a.cacheStorage.Set(key, value); err != nil {
		log.Printf("Failed to set cache parameters for %s: %s\n", key, err.Error())
	}

	return value
}

// GetRequestBody returns the GeneratedRequest body for the Operation.
func (a *CacheOperationAdapter) GetRequestBody() (*openapi.Schema, string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	key := a.key("requestBody")
	if cached, ok := a.cacheStorage.Get(key); ok {
		res := cached.(*SchemaWithContentType)
		return res.Schema, res.ContentType
	}

	value, contentType := a.operation.GetRequestBody()
	if err := a.cacheStorage.Set(key, &SchemaWithContentType{
		Schema:      value,
		ContentType: contentType,
	}); err != nil {
		log.Printf("Failed to set cache requestBody for %s: %s\n", key, err.Error())
	}

	return value, contentType
}

// GetResponse returns the response for the Operation.
func (a *CacheOperationAdapter) GetResponse() *openapi.Response {
	a.mu.Lock()
	defer a.mu.Unlock()

	key := a.key("response")
	if cached, ok := a.cacheStorage.Get(key); ok {
		return cached.(*openapi.Response)
	}

	value := a.operation.GetResponse()
	if err := a.cacheStorage.Set(key, value); err != nil {
		log.Printf("Failed to set cache response for %s: %s\n", key, err.Error())
	}

	return value
}

// key returns a key for the given type to be stored in cache.
func (a *CacheOperationAdapter) key(typ string) string {
	return a.service + ":" + a.operation.ID() + ":" + typ
}
