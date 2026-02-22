// Package db provides a shared storage abstraction with support for
// multiple backends (memory, redis) and per-service isolated views.
package db

import (
	"log/slog"
	"time"

	"github.com/cubahno/connexions/v2/pkg/config"
)

// Storage is the shared storage backend that can provide per-service DB instances.
// There should be only one Storage instance per application.
type Storage interface {
	// NewDB returns a DB scoped to a specific service.
	// The returned DB shares the underlying storage but isolates data via key prefixing.
	NewDB(serviceName string, historyDuration time.Duration) DB

	// Close releases any resources held by the storage backend.
	Close()
}

// NewStorage creates a shared storage backend based on configuration.
// If storageCfg is nil or type is memory, returns an in-memory storage.
// If type is redis, returns a Redis-backed storage.
func NewStorage(storageCfg *config.StorageConfig) Storage {
	// Default to memory if no config or memory type
	if storageCfg == nil || storageCfg.Type == "" || storageCfg.Type == config.StorageTypeMemory {
		return newMemoryStorage()
	}

	if storageCfg.Type == config.StorageTypeRedis {
		storage, err := newRedisStorage(storageCfg.Redis)
		if err != nil {
			slog.Error("Failed to create Redis storage, falling back to memory", "error", err)
			return newMemoryStorage()
		}
		return storage
	}

	// Unknown type, fall back to memory
	slog.Warn("Unknown storage type, falling back to memory", "type", storageCfg.Type)
	return newMemoryStorage()
}
