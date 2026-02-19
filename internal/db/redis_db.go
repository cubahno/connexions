package db

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cubahno/connexions/v2/pkg/config"
	"github.com/redis/go-redis/v9"
	"github.com/sony/gobreaker/v2"
)

// Ensure redisDB implements DB interface.
var _ DB = (*redisDB)(nil)

// redisDB is a Redis-backed implementation of DB.
type redisDB struct {
	client          *redis.Client
	serviceName     string
	historyDuration time.Duration

	mu      sync.RWMutex
	tables  map[string]*redisTable
	history *redisHistoryTable
	cbStore *redisCircuitBreakerStore
}

// NewRedisDB creates a new Redis-backed database for a service.
// historyDuration specifies the TTL for history records (0 means no expiry).
func NewRedisDB(cfg *config.RedisConfig, serviceName string, historyDuration time.Duration) (DB, error) {
	if cfg == nil {
		return nil, fmt.Errorf("redis config is nil")
	}

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Address,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	db := &redisDB{
		client:          client,
		serviceName:     serviceName,
		historyDuration: historyDuration,
		tables:          make(map[string]*redisTable),
		cbStore:         newRedisCircuitBreakerStore(client),
	}

	// Create the default storage table for the service
	storageTable := newRedisTable(client, serviceName, "_storage", 0) // no TTL for storage
	db.tables["_storage"] = storageTable

	// Create history table with reference to storage
	db.history = newRedisHistoryTable(client, serviceName, storageTable, historyDuration)

	return db, nil
}

// History returns the history table.
func (db *redisDB) History() HistoryTable {
	return db.history
}

// Table returns a table by name, creating it if it doesn't exist.
func (db *redisDB) Table(name string) Table {
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

	t = newRedisTable(db.client, db.serviceName, name, 0) // no TTL by default
	db.tables[name] = t
	return t
}

// CircuitBreakerStore returns the circuit breaker store.
func (db *redisDB) CircuitBreakerStore() gobreaker.SharedDataStore {
	return db.cbStore
}

// Close releases resources held by the database.
func (db *redisDB) Close() {
	_ = db.client.Close()
}
