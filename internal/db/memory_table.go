package db

import "sync"

// memoryTable is an in-memory implementation of Table.
type memoryTable struct {
	mu   sync.RWMutex
	data map[string]any
}

// newMemoryTable creates a new in-memory table.
func newMemoryTable() *memoryTable {
	return &memoryTable{
		data: make(map[string]any),
	}
}

// Get retrieves a value by key.
func (t *memoryTable) Get(key string) (any, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	data, ok := t.data[key]
	return data, ok
}

// Set stores a value with the given key.
func (t *memoryTable) Set(key string, value any) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.data[key] = value
}

// Delete removes a value by key.
func (t *memoryTable) Delete(key string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.data, key)
}

// Data returns a copy of all data in the table.
func (t *memoryTable) Data() map[string]any {
	t.mu.RLock()
	defer t.mu.RUnlock()

	cp := make(map[string]any, len(t.data))
	for k, v := range t.data {
		cp[k] = v
	}
	return cp
}

// Clear removes all data from the table.
func (t *memoryTable) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.data = make(map[string]any)
}
