package db

import (
	"context"
	"sync"
	"time"
)

// memoryEntry holds a value with optional expiration.
type memoryEntry struct {
	value     any
	expiresAt time.Time // zero value means no expiry
}

// isExpired returns true if the entry has expired.
func (e *memoryEntry) isExpired() bool {
	return !e.expiresAt.IsZero() && time.Now().After(e.expiresAt)
}

// memoryTable is an in-memory implementation of Table.
type memoryTable struct {
	mu   sync.RWMutex
	data map[string]*memoryEntry
}

// newMemoryTable creates a new in-memory table.
func newMemoryTable() *memoryTable {
	return &memoryTable{
		data: make(map[string]*memoryEntry),
	}
}

// Get retrieves a value by key.
// Returns false if the key doesn't exist or has expired.
func (t *memoryTable) Get(_ context.Context, key string) (any, bool) {
	t.mu.RLock()
	entry, ok := t.data[key]
	t.mu.RUnlock()

	if !ok {
		return nil, false
	}

	if entry.isExpired() {
		// Lazy deletion - upgrade to write lock
		t.mu.Lock()
		// Double-check after acquiring write lock
		if entry, ok = t.data[key]; ok && entry.isExpired() {
			delete(t.data, key)
		}
		t.mu.Unlock()
		return nil, false
	}

	return entry.value, true
}

// Set stores a value with the given key.
// If ttl is 0, the value never expires.
func (t *memoryTable) Set(_ context.Context, key string, value any, ttl time.Duration) {
	entry := &memoryEntry{value: value}
	if ttl > 0 {
		entry.expiresAt = time.Now().Add(ttl)
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	t.data[key] = entry
}

// Delete removes a value by key.
func (t *memoryTable) Delete(_ context.Context, key string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.data, key)
}

// Data returns a copy of all non-expired data in the table.
func (t *memoryTable) Data(_ context.Context) map[string]any {
	t.mu.RLock()
	defer t.mu.RUnlock()

	cp := make(map[string]any)
	for k, entry := range t.data {
		if !entry.isExpired() {
			cp[k] = entry.value
		}
	}
	return cp
}

// Clear removes all data from the table.
func (t *memoryTable) Clear(_ context.Context) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.data = make(map[string]*memoryEntry)
}
