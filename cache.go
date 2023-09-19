package connexions

import (
	"log"
	"sync"
)

type CacheStorage interface {
	Set(key string, value any) error
	Get(key string) (any, bool)
}

type MemoryStorage struct {
	data map[string]any
	mu   sync.Mutex
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		data: make(map[string]any),
	}
}

func (s *MemoryStorage) Set(key string, value any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = value
	return nil
}

func (s *MemoryStorage) Get(key string) (any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	value, ok := s.data[key]
	return value, ok
}

type SchemaWithContentType struct {
	Schema      *Schema
	ContentType string
}

type CacheOperationAdapter struct {
	service      string
	operation    Operationer
	cacheStorage CacheStorage
	mu           sync.Mutex
}

func NewCacheOperationAdapter(service string, operation Operationer, storage CacheStorage) Operationer {
	return &CacheOperationAdapter{
		service:      service,
		operation:    operation,
		cacheStorage: storage,
	}
}

func (a *CacheOperationAdapter) WithParseConfig(parseConfig *ParseConfig) Operationer {
	a.operation.WithParseConfig(parseConfig)
	return a
}

func (a *CacheOperationAdapter) ID() string {
	return a.operation.ID()
}

func (a *CacheOperationAdapter) GetParameters() OpenAPIParameters {
	a.mu.Lock()
	defer a.mu.Unlock()

	key := a.key("parameters")
	if cached, ok := a.cacheStorage.Get(key); ok {
		return cached.(OpenAPIParameters)
	}

	value := a.operation.GetParameters()
	if err := a.cacheStorage.Set(key, value); err != nil {
		log.Printf("Failed to set cache parameters for %s: %s\n", key, err.Error())
	}

	return value
}

func (a *CacheOperationAdapter) GetRequestBody() (*Schema, string) {
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

func (a *CacheOperationAdapter) GetResponse() *OpenAPIResponse {
	a.mu.Lock()
	defer a.mu.Unlock()

	key := a.key("response")
	if cached, ok := a.cacheStorage.Get(key); ok {
		return cached.(*OpenAPIResponse)
	}

	value := a.operation.GetResponse()
	if err := a.cacheStorage.Set(key, value); err != nil {
		log.Printf("Failed to set cache response for %s: %s\n", key, err.Error())
	}

	return value
}

func (a *CacheOperationAdapter) key(typ string) string {
	return a.service + ":" + a.operation.ID() + ":" + typ
}
