package db

import (
	"sync"
	"time"

	"github.com/sony/gobreaker/v2"
)

// Ensure memoryDB implements DB interface.
var _ DB = (*memoryDB)(nil)

// memoryDB is an in-memory implementation of DB.
type memoryDB struct {
	serviceName     string
	historyDuration time.Duration

	mu      sync.RWMutex
	tables  map[string]*memoryTable
	history *memoryHistoryTable
	cbStore *memoryCircuitBreakerStore
}

// NewMemoryDB creates a new in-memory database for a service.
// historyDuration specifies how often the history table is cleared (0 means no auto-clear).
func NewMemoryDB(serviceName string, historyDuration time.Duration) DB {
	db := &memoryDB{
		serviceName:     serviceName,
		historyDuration: historyDuration,
		tables:          make(map[string]*memoryTable),
		cbStore:         newMemoryCircuitBreakerStore(),
	}

	// Create the default storage table for the service
	storageTable := newMemoryTable()
	db.tables["_storage"] = storageTable

	// Create history table with reference to storage
	db.history = newMemoryHistoryTable(serviceName, storageTable, historyDuration)

	return db
}

// History returns the history table.
func (db *memoryDB) History() HistoryTable {
	return db.history
}

// Table returns a table by name, creating it if it doesn't exist.
func (db *memoryDB) Table(name string) Table {
	db.mu.RLock()
	t, ok := db.tables[name]
	db.mu.RUnlock()

	if ok {
		return t
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	// Double-check after acquiring write lock
	if t, ok = db.tables[name]; ok {
		return t
	}

	t = newMemoryTable()
	db.tables[name] = t
	return t
}

// CircuitBreakerStore returns the circuit breaker store.
func (db *memoryDB) CircuitBreakerStore() gobreaker.SharedDataStore {
	return db.cbStore
}

// Close releases resources held by the database.
func (db *memoryDB) Close() {
	db.history.cancel()
}
