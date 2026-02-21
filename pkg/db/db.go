// Package db provides a per-service database abstraction with support for
// multiple storage backends (memory, redis) and named tables.
package db

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/cubahno/connexions/v2/pkg/config"
	"github.com/sony/gobreaker/v2"
)

// DB is a per-service database that provides access to named tables.
// Each service gets its own isolated DB instance.
type DB interface {
	// History returns the history table with typed methods for request/response tracking.
	History() HistoryTable

	// Table returns a generic key-value table by name.
	// Tables are created lazily on first access.
	Table(name string) Table

	// CircuitBreakerStore returns a store for distributed circuit breaker state.
	// Implements gobreaker.SharedDataStore interface.
	CircuitBreakerStore() gobreaker.SharedDataStore

	// Close releases any resources held by the database.
	Close()
}

// Table is a generic key-value store with optional TTL support.
type Table interface {
	// Get retrieves a value by key.
	Get(key string) (any, bool)

	// Set stores a value with the given key.
	Set(key string, value any)

	// Delete removes a value by key.
	Delete(key string)

	// Data returns a copy of all data in the table.
	Data() map[string]any

	// Clear removes all data from the table.
	Clear()
}

// HistoryTable provides typed access to request/response history.
type HistoryTable interface {
	// Get retrieves a request record by the HTTP request.
	Get(req *http.Request) (*RequestedResource, bool)

	// Set stores a request record.
	Set(resource string, req *http.Request, response *Response) *RequestedResource

	// SetResponse updates the response for an existing request record.
	SetResponse(req *http.Request, response *Response)

	// Data returns all request records.
	Data() map[string]*RequestedResource

	// Clear removes all history records.
	Clear()
}

// RequestedResource represents the current request being processed.
// Resource is the openapi resource path, i.e. /pets, /pets/{id}
// Body is the request body if method is not GET
// Response is the current response if present
// Request is the current http request
// Storage is the thread-safe storage for this request's service.
type RequestedResource struct {
	Resource string
	Body     []byte
	Response *Response
	Request  *http.Request
	Storage  Table
}

// Response represents the response that was generated or received from the server.
// Data is the response body
// StatusCode is the HTTP status code returned
// ContentType is the Content-Type header of the response
// IsFromUpstream is true if the response was received from the upstream server
type Response struct {
	Data           []byte
	StatusCode     int
	ContentType    string
	IsFromUpstream bool
}

// NewDB creates a new database for a service based on the storage configuration.
// If storageCfg is nil or type is memory, returns an in-memory database.
// If type is redis, returns a Redis-backed database.
func NewDB(serviceName string, historyDuration time.Duration, storageCfg *config.StorageConfig) DB {
	// Default to memory if no config or memory type
	if storageCfg == nil || storageCfg.Type == "" || storageCfg.Type == config.StorageTypeMemory {
		return NewMemoryDB(serviceName, historyDuration)
	}

	if storageCfg.Type == config.StorageTypeRedis {
		db, err := NewRedisDB(storageCfg.Redis, serviceName, historyDuration)
		if err != nil {
			slog.Error("Failed to create Redis DB, falling back to memory", "error", err, "service", serviceName)
			return NewMemoryDB(serviceName, historyDuration)
		}
		return db
	}

	// Unknown type, fall back to memory
	slog.Warn("Unknown storage type, falling back to memory", "type", storageCfg.Type, "service", serviceName)
	return NewMemoryDB(serviceName, historyDuration)
}
