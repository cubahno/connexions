package db

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	assert2 "github.com/stretchr/testify/assert"
)

func TestNewMemoryDB(t *testing.T) {
	assert := assert2.New(t)

	db := NewMemoryDB("test-service", 5*time.Minute)
	defer db.Close()

	assert.NotNil(db)
	assert.NotNil(db.History())
	assert.NotNil(db.Table("_storage"))
}

func TestMemoryDB_History(t *testing.T) {
	assert := assert2.New(t)

	db := NewMemoryDB("test-service", 0)
	defer db.Close()

	history := db.History()

	assert.NotNil(history)

	// Test that history works
	req := &http.Request{Method: "GET", URL: &url.URL{Path: "/test"}}
	result := history.Set("/test", req, nil)

	assert.Equal("/test", result.Resource)
	assert.NotNil(result.Storage)
}

func TestMemoryDB_Table(t *testing.T) {
	assert := assert2.New(t)

	t.Run("create new table", func(t *testing.T) {
		db := NewMemoryDB("test-service", 0)
		defer db.Close()

		table := db.Table("psp_operations")

		assert.NotNil(table)
	})

	t.Run("get existing table", func(t *testing.T) {
		db := NewMemoryDB("test-service", 0)
		defer db.Close()

		table1 := db.Table("psp_operations")
		table2 := db.Table("psp_operations")

		assert.Same(table1, table2)
	})

	t.Run("different tables are independent", func(t *testing.T) {
		db := NewMemoryDB("test-service", 0)
		defer db.Close()

		table1 := db.Table("table1")
		table2 := db.Table("table2")

		table1.Set("key", "value1")
		table2.Set("key", "value2")

		val1, _ := table1.Get("key")
		val2, _ := table2.Get("key")

		assert.Equal("value1", val1)
		assert.Equal("value2", val2)
	})
}

func TestMemoryDB_CircuitBreakerStore(t *testing.T) {
	assert := assert2.New(t)

	db := NewMemoryDB("test-service", 0)
	defer db.Close()

	store := db.CircuitBreakerStore()
	assert.NotNil(store)

	// Verify it works
	err := store.Lock("test")
	assert.NoError(err)

	err = store.Unlock("test")
	assert.NoError(err)
}

func TestMemoryDB_Close(t *testing.T) {
	assert := assert2.New(t)

	db := NewMemoryDB("test-service", 50*time.Millisecond)

	req := &http.Request{Method: "GET", URL: &url.URL{Path: "/test"}}
	db.History().Set("/test", req, nil)

	// Close should stop the history auto-clear
	db.Close()

	// Wait past the clear timeout
	time.Sleep(100 * time.Millisecond)

	// Data should still be there since ticker was cancelled
	assert.Len(db.History().Data(), 1)
}

func TestMemoryDB_ImplementsInterface(t *testing.T) {
	// Compile-time check that memoryDB implements DB
	var _ DB = (*memoryDB)(nil)
}
