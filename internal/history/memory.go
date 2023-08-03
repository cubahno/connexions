package history

import "sync"

type MemoryStorage struct {
	mu   sync.Mutex
	data map[string]any
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		data: make(map[string]any),
	}
}

func (s *MemoryStorage) Get(key string) (any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, ok := s.data[key]
	return data, ok
}

func (s *MemoryStorage) Set(key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
}

func (s *MemoryStorage) Data() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	cp := make(map[string]any, len(s.data))
	for k, v := range s.data {
		cp[k] = v
	}
	return cp
}
