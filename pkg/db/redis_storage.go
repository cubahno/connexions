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

// Ensure redisStorage implements Storage interface.
var _ Storage = (*redisStorage)(nil)

// redisStorage is a shared Redis-backed storage.
type redisStorage struct {
	client  *redis.Client
	cbStore *redisCircuitBreakerStore

	mu       sync.RWMutex
	services map[string]*redisServiceDB // track service DBs for cleanup
}

// newRedisStorage creates a new shared Redis storage.
func newRedisStorage(cfg *config.RedisConfig) (*redisStorage, error) {
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

	return &redisStorage{
		client:   client,
		cbStore:  newRedisCircuitBreakerStore(client),
		services: make(map[string]*redisServiceDB),
	}, nil
}

// NewDB returns a DB scoped to a specific service.
func (s *redisStorage) NewDB(serviceName string, historyDuration time.Duration) DB {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if we already have a DB for this service
	if db, ok := s.services[serviceName]; ok {
		return db
	}

	db := &redisServiceDB{
		storage:         s,
		serviceName:     serviceName,
		historyDuration: historyDuration,
		tables:          make(map[string]*redisTable),
		history:         newRedisHistoryTable(s.client, serviceName+":history", historyDuration),
	}

	s.services[serviceName] = db
	return db
}

// Close releases resources held by the storage.
func (s *redisStorage) Close() {
	_ = s.client.Close()
}

// redisServiceDB is a service-scoped view into shared Redis storage.
type redisServiceDB struct {
	storage         *redisStorage
	serviceName     string
	historyDuration time.Duration

	mu      sync.RWMutex
	tables  map[string]*redisTable
	history *redisHistoryTable
}

// History returns the history table.
func (db *redisServiceDB) History() HistoryTable {
	return db.history
}

// Table returns a table by name, creating it if it doesn't exist.
func (db *redisServiceDB) Table(name string) Table {
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

	t = newRedisTable(db.storage.client, db.serviceName, name)
	db.tables[name] = t
	return t
}

// CircuitBreakerStore returns the shared circuit breaker store.
func (db *redisServiceDB) CircuitBreakerStore() gobreaker.SharedDataStore {
	return db.storage.cbStore
}

// Close releases resources (no-op for Redis service DB, storage handles cleanup).
func (db *redisServiceDB) Close() {
	// Nothing to close - the shared storage owns the Redis client
}
