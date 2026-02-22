package db

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMemoryStorage_NewDB(t *testing.T) {
	ctx := context.Background()
	storage := newMemoryStorage()
	defer storage.Close()

	db1 := storage.NewDB("service1", 5*time.Minute)
	db2 := storage.NewDB("service2", 5*time.Minute)

	assert.NotNil(t, db1)
	assert.NotNil(t, db2)

	// Verify they have separate tables
	db1.Table("users").Set(ctx, "key", "value1", 0)
	db2.Table("users").Set(ctx, "key", "value2", 0)

	val1, _ := db1.Table("users").Get(ctx, "key")
	val2, _ := db2.Table("users").Get(ctx, "key")

	assert.Equal(t, "value1", val1)
	assert.Equal(t, "value2", val2)
}

func TestMemoryStorage_SharedBackend(t *testing.T) {
	ctx := context.Background()
	storage := newMemoryStorage()
	defer storage.Close()

	// Get the same service twice - should share underlying tables
	db1 := storage.NewDB("myservice", 5*time.Minute)
	db2 := storage.NewDB("myservice", 5*time.Minute)

	db1.Table("data").Set(ctx, "key", "shared_value", 0)

	// Both should see the same data since they share the storage
	val1, ok1 := db1.Table("data").Get(ctx, "key")
	val2, ok2 := db2.Table("data").Get(ctx, "key")

	assert.True(t, ok1)
	assert.True(t, ok2)
	assert.Equal(t, "shared_value", val1)
	assert.Equal(t, "shared_value", val2)
}

func TestMemoryStorage_CircuitBreakerStoreShared(t *testing.T) {
	storage := newMemoryStorage()
	defer storage.Close()

	db1 := storage.NewDB("service1", 5*time.Minute)
	db2 := storage.NewDB("service2", 5*time.Minute)

	// Both services should share the same circuit breaker store
	store1 := db1.CircuitBreakerStore()
	store2 := db2.CircuitBreakerStore()

	assert.NotNil(t, store1)
	assert.NotNil(t, store2)

	// Lock from one service should be visible to the other
	err := store1.Lock("shared-breaker")
	assert.NoError(t, err)

	// Unlock from the other service
	err = store2.Unlock("shared-breaker")
	assert.NoError(t, err)
}

func TestMemoryStorage_History(t *testing.T) {
	ctx := context.Background()
	storage := newMemoryStorage()
	defer storage.Close()

	db := storage.NewDB("testservice", 0)
	history := db.History()

	assert.NotNil(t, history)

	req := &http.Request{Method: "GET", URL: &url.URL{Path: "/test"}}
	result := history.Set(ctx, "/test", req, nil)

	assert.Equal(t, "/test", result.Resource)
}

func TestMemoryStorage_Close(t *testing.T) {
	ctx := context.Background()
	storage := newMemoryStorage()

	// Create some service DBs
	db1 := storage.NewDB("service1", 50*time.Millisecond)
	db2 := storage.NewDB("service2", 50*time.Millisecond)

	// Add some data
	req := &http.Request{Method: "GET", URL: &url.URL{Path: "/test"}}
	db1.History().Set(ctx, "/test1", req, nil)
	db2.History().Set(ctx, "/test2", req, nil)

	// Close individual service DBs
	db1.Close()
	db2.Close()

	// Close storage (should be safe even after service DBs are closed)
	storage.Close()
}
