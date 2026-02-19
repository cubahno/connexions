package db

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/cubahno/connexions/v2/pkg/config"
	assert2 "github.com/stretchr/testify/assert"
)

func TestNewRedisDB(t *testing.T) {
	assert := assert2.New(t)

	t.Run("nil config returns error", func(t *testing.T) {
		db, err := NewRedisDB(nil, "test-service", 100*time.Second)
		assert.Nil(db)
		assert.Error(err)
		assert.Contains(err.Error(), "redis config is nil")
	})

	t.Run("invalid address returns error", func(t *testing.T) {
		cfg := &config.RedisConfig{
			Address: "invalid:99999",
		}
		db, err := NewRedisDB(cfg, "test-service", 100*time.Second)
		assert.Nil(db)
		assert.Error(err)
		assert.Contains(err.Error(), "failed to connect to redis")
	})

	t.Run("successful connection", func(t *testing.T) {
		mr := miniredis.RunT(t)
		cfg := &config.RedisConfig{Address: mr.Addr()}

		db, err := NewRedisDB(cfg, "test-service", 100*time.Second)
		assert.NoError(err)
		assert.NotNil(db)
		assert.NotNil(db.History())
		db.Close()
	})
}

func TestRedisDB_History(t *testing.T) {
	assert := assert2.New(t)
	mr := miniredis.RunT(t)
	cfg := &config.RedisConfig{Address: mr.Addr()}

	db, err := NewRedisDB(cfg, "test-service", 0)
	assert.NoError(err)
	defer db.Close()

	history := db.History()
	assert.NotNil(history)

	// Test that history works
	req := &http.Request{Method: "GET", URL: &url.URL{Path: "/test"}}
	result := history.Set("/test", req, nil)

	assert.Equal("/test", result.Resource)
	assert.NotNil(result.Storage)
}

func TestRedisDB_Table(t *testing.T) {
	assert := assert2.New(t)
	mr := miniredis.RunT(t)
	cfg := &config.RedisConfig{Address: mr.Addr()}

	db, err := NewRedisDB(cfg, "test-service", 100*time.Second)
	assert.NoError(err)
	defer db.Close()

	t.Run("get table creates new table", func(t *testing.T) {
		table := db.Table("my-table")
		assert.NotNil(table)
	})

	t.Run("get same table returns same instance", func(t *testing.T) {
		table1 := db.Table("same-table")
		table2 := db.Table("same-table")
		assert.Same(table1, table2)
	})

	t.Run("different tables are independent", func(t *testing.T) {
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

func TestRedisTable(t *testing.T) {
	assert := assert2.New(t)
	mr := miniredis.RunT(t)
	cfg := &config.RedisConfig{Address: mr.Addr()}

	db, err := NewRedisDB(cfg, "test-service", 100*time.Second)
	assert.NoError(err)
	defer db.Close()

	table := db.Table("test-table")

	t.Run("set and get", func(t *testing.T) {
		table.Set("key1", "value1")
		val, ok := table.Get("key1")
		assert.True(ok)
		assert.Equal("value1", val)
	})

	t.Run("get non-existent key", func(t *testing.T) {
		val, ok := table.Get("non-existent")
		assert.False(ok)
		assert.Nil(val)
	})

	t.Run("delete", func(t *testing.T) {
		table.Set("to-delete", "value")
		table.Delete("to-delete")
		_, ok := table.Get("to-delete")
		assert.False(ok)
	})

	t.Run("data returns all entries", func(t *testing.T) {
		// Clear first
		table.Clear()
		table.Set("a", "1")
		table.Set("b", "2")

		data := table.Data()
		assert.Len(data, 2)
		assert.Equal("1", data["a"])
		assert.Equal("2", data["b"])
	})

	t.Run("clear removes all entries", func(t *testing.T) {
		table.Set("x", "y")
		table.Clear()
		data := table.Data()
		assert.Empty(data)
	})

	t.Run("get with corrupted json returns false", func(t *testing.T) {
		// Manually set corrupted data
		_ = mr.Set("test-service:test-table:corrupted", "not-valid-json")
		_, ok := table.Get("corrupted")
		assert.False(ok)
	})

	t.Run("data skips corrupted entries", func(t *testing.T) {
		table.Clear()
		table.Set("valid", "value")
		_ = mr.Set("test-service:test-table:corrupted2", "not-valid-json")

		data := table.Data()
		assert.Len(data, 1)
		assert.Equal("value", data["valid"])
	})

	t.Run("set with unmarshalable value does nothing", func(t *testing.T) {
		// Channels cannot be marshaled to JSON
		ch := make(chan int)
		table.Set("unmarshalable", ch)

		_, ok := table.Get("unmarshalable")
		assert.False(ok)
	})
}

func TestRedisHistoryTable(t *testing.T) {
	assert := assert2.New(t)
	mr := miniredis.RunT(t)
	cfg := &config.RedisConfig{Address: mr.Addr()}

	db, err := NewRedisDB(cfg, "test-service", 100*time.Second)
	assert.NoError(err)
	defer db.Close()

	history := db.History()

	t.Run("set and get", func(t *testing.T) {
		req := &http.Request{
			Method: http.MethodGet,
			URL:    &url.URL{Path: "/test"},
		}
		resp := &Response{
			Data:       []byte("response data"),
			StatusCode: 200,
		}

		rec := history.Set("/test", req, resp)
		assert.NotNil(rec)
		assert.Equal("/test", rec.Resource)
		assert.Equal(200, rec.Response.StatusCode)

		got, ok := history.Get(req)
		assert.True(ok)
		assert.Equal("/test", got.Resource)
	})

	t.Run("set with body", func(t *testing.T) {
		body := strings.NewReader(`{"foo":"bar"}`)
		req, _ := http.NewRequest(http.MethodPost, "/with-body", body)

		rec := history.Set("/with-body", req, nil)
		assert.NotNil(rec)
		assert.Equal([]byte(`{"foo":"bar"}`), rec.Body)
	})

	t.Run("set response updates existing record", func(t *testing.T) {
		req := &http.Request{
			Method: http.MethodGet,
			URL:    &url.URL{Path: "/update-resp"},
		}
		history.Set("/update-resp", req, nil)

		resp := &Response{
			Data:       []byte("updated"),
			StatusCode: 201,
		}
		history.SetResponse(req, resp)

		got, ok := history.Get(req)
		assert.True(ok)
		assert.Equal(201, got.Response.StatusCode)
	})

	t.Run("data returns all records", func(t *testing.T) {
		history.Clear()

		req1 := &http.Request{Method: http.MethodGet, URL: &url.URL{Path: "/a"}}
		req2 := &http.Request{Method: http.MethodGet, URL: &url.URL{Path: "/b"}}

		history.Set("/a", req1, nil)
		history.Set("/b", req2, nil)

		data := history.Data()
		assert.Len(data, 2)
	})

	t.Run("clear removes all records", func(t *testing.T) {
		req := &http.Request{Method: http.MethodGet, URL: &url.URL{Path: "/clear-test"}}
		history.Set("/clear-test", req, nil)

		history.Clear()
		data := history.Data()
		assert.Empty(data)
	})

	t.Run("set response for non-existent record", func(t *testing.T) {
		history.Clear()
		req := &http.Request{Method: http.MethodGet, URL: &url.URL{Path: "/non-existent"}}

		// Should not panic, just log
		history.SetResponse(req, &Response{Data: []byte("test"), StatusCode: 200})

		// Record should not exist
		_, ok := history.Get(req)
		assert.False(ok)
	})

	t.Run("get with corrupted data returns false", func(t *testing.T) {
		// Manually set corrupted data in Redis
		_ = mr.Set("test-service:history:GET:/corrupted", "not-valid-json")

		req := &http.Request{Method: http.MethodGet, URL: &url.URL{Path: "/corrupted"}}
		_, ok := history.Get(req)
		assert.False(ok)
	})

	t.Run("set response with corrupted existing data", func(t *testing.T) {
		// Manually set corrupted data in Redis
		_ = mr.Set("test-service:history:GET:/corrupted2", "not-valid-json")

		req := &http.Request{Method: http.MethodGet, URL: &url.URL{Path: "/corrupted2"}}
		// Should not panic
		history.SetResponse(req, &Response{Data: []byte("test"), StatusCode: 200})
	})

	t.Run("reuse body from existing record", func(t *testing.T) {
		history.Clear()
		body := strings.NewReader(`{"existing":"body"}`)
		req1, _ := http.NewRequest(http.MethodPost, "/reuse-body", body)
		history.Set("/reuse-body", req1, nil)

		// Second request without body
		req2 := &http.Request{Method: http.MethodPost, URL: &url.URL{Path: "/reuse-body"}}
		rec := history.Set("/reuse-body", req2, nil)
		assert.Equal([]byte(`{"existing":"body"}`), rec.Body)
	})

	t.Run("data skips corrupted entries", func(t *testing.T) {
		history.Clear()
		req := &http.Request{Method: http.MethodGet, URL: &url.URL{Path: "/valid-entry"}}
		history.Set("/valid-entry", req, nil)

		// Manually set corrupted data
		_ = mr.Set("test-service:history:GET:/corrupted-data", "not-valid-json")

		data := history.Data()
		assert.Len(data, 1)
		assert.Contains(data, "GET:/valid-entry")
	})
}

func TestNewDB_Factory(t *testing.T) {
	assert := assert2.New(t)

	t.Run("nil config returns memory DB", func(t *testing.T) {
		db := NewDB("test", 100*time.Second, nil)
		assert.NotNil(db)
		_, isMemory := db.(*memoryDB)
		assert.True(isMemory)
		db.Close()
	})

	t.Run("memory type returns memory DB", func(t *testing.T) {
		cfg := &config.StorageConfig{Type: config.StorageTypeMemory}
		db := NewDB("test", 100*time.Second, cfg)
		assert.NotNil(db)
		_, isMemory := db.(*memoryDB)
		assert.True(isMemory)
		db.Close()
	})

	t.Run("redis type with valid config returns redis DB", func(t *testing.T) {
		mr := miniredis.RunT(t)
		cfg := &config.StorageConfig{
			Type:  config.StorageTypeRedis,
			Redis: &config.RedisConfig{Address: mr.Addr()},
		}
		db := NewDB("test", 100*time.Second, cfg)
		assert.NotNil(db)
		_, isRedis := db.(*redisDB)
		assert.True(isRedis)
		db.Close()
	})

	t.Run("redis type with invalid config falls back to memory", func(t *testing.T) {
		cfg := &config.StorageConfig{
			Type:  config.StorageTypeRedis,
			Redis: &config.RedisConfig{Address: "invalid:99999"},
		}
		db := NewDB("test", 100*time.Second, cfg)
		assert.NotNil(db)
		_, isMemory := db.(*memoryDB)
		assert.True(isMemory)
		db.Close()
	})

	t.Run("unknown storage type falls back to memory", func(t *testing.T) {
		cfg := &config.StorageConfig{
			Type: "unknown-type",
		}
		db := NewDB("test", 100*time.Second, cfg)
		assert.NotNil(db)
		_, isMemory := db.(*memoryDB)
		assert.True(isMemory)
		db.Close()
	})
}

func TestRedisDB_CircuitBreakerStore(t *testing.T) {
	assert := assert2.New(t)
	mr := miniredis.RunT(t)
	cfg := &config.RedisConfig{Address: mr.Addr()}

	db, err := NewRedisDB(cfg, "test-service", 100*time.Second)
	assert.NoError(err)
	defer db.Close()

	store := db.CircuitBreakerStore()
	assert.NotNil(store)

	// Verify it works
	err = store.Lock("test")
	assert.NoError(err)

	err = store.Unlock("test")
	assert.NoError(err)
}

func TestRedisDB_Close(t *testing.T) {
	assert := assert2.New(t)
	mr := miniredis.RunT(t)
	cfg := &config.RedisConfig{Address: mr.Addr()}

	db, err := NewRedisDB(cfg, "test-service", 100*time.Second)
	assert.NoError(err)

	req := &http.Request{Method: "GET", URL: &url.URL{Path: "/test"}}
	db.History().Set("/test", req, nil)

	// Close should work without error
	db.Close()

	// Data should still be in Redis (persisted)
	// Reconnect to verify
	db2, err := NewRedisDB(cfg, "test-service", 100*time.Second)
	assert.NoError(err)
	defer db2.Close()

	got, ok := db2.History().Get(req)
	assert.True(ok)
	assert.Equal("/test", got.Resource)
}

func TestRedisDB_ImplementsInterface(t *testing.T) {
	// Compile-time check that redisDB implements DB
	var _ DB = (*redisDB)(nil)
}
