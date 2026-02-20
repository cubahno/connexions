package db

import (
	"fmt"
	"sync"

	"github.com/sony/gobreaker/v2"
)

// Ensure memoryCircuitBreakerStore implements gobreaker.SharedDataStore
var _ gobreaker.SharedDataStore = (*memoryCircuitBreakerStore)(nil)

// memoryCircuitBreakerStore is an in-memory implementation of gobreaker.SharedDataStore.
type memoryCircuitBreakerStore struct {
	mu    sync.Mutex
	locks map[string]bool
	data  map[string][]byte
}

// newMemoryCircuitBreakerStore creates a new in-memory circuit breaker store.
func newMemoryCircuitBreakerStore() *memoryCircuitBreakerStore {
	return &memoryCircuitBreakerStore{
		locks: make(map[string]bool),
		data:  make(map[string][]byte),
	}
}

// Lock acquires a lock for the given name.
func (s *memoryCircuitBreakerStore) Lock(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.locks[name] {
		return fmt.Errorf("lock already held for %s", name)
	}
	s.locks[name] = true
	return nil
}

// Unlock releases the lock for the given name.
func (s *memoryCircuitBreakerStore) Unlock(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.locks, name)
	return nil
}

// GetData retrieves circuit breaker state data.
func (s *memoryCircuitBreakerStore) GetData(name string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, ok := s.data[name]
	if !ok {
		return nil, nil
	}
	// Return a copy to prevent mutation
	result := make([]byte, len(data))
	copy(result, data)
	return result, nil
}

// SetData stores circuit breaker state data.
func (s *memoryCircuitBreakerStore) SetData(name string, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Store a copy to prevent mutation
	stored := make([]byte, len(data))
	copy(stored, data)
	s.data[name] = stored
	return nil
}
