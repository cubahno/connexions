package db

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func newTestRedisTable(t *testing.T) (*redisTable, *miniredis.Miniredis) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	table := newRedisTable(client, "test", "users")
	return table, mr
}

func TestRedisTable_Get(t *testing.T) {
	ctx := context.Background()

	t.Run("get existing key", func(t *testing.T) {
		table, _ := newTestRedisTable(t)
		table.Set(ctx, "foo", "bar", 0)

		val, ok := table.Get(ctx, "foo")
		assert.True(t, ok)
		assert.Equal(t, "bar", val)
	})

	t.Run("get non-existing key", func(t *testing.T) {
		table, _ := newTestRedisTable(t)

		val, ok := table.Get(ctx, "foo")
		assert.False(t, ok)
		assert.Nil(t, val)
	})

	t.Run("get with invalid json returns false", func(t *testing.T) {
		table, mr := newTestRedisTable(t)
		// Set invalid JSON directly in Redis
		_ = mr.Set("test:users:bad", "not-json{")

		val, ok := table.Get(ctx, "bad")
		assert.False(t, ok)
		assert.Nil(t, val)
	})
}

func TestRedisTable_Set(t *testing.T) {
	ctx := context.Background()

	t.Run("set new key", func(t *testing.T) {
		table, _ := newTestRedisTable(t)
		table.Set(ctx, "foo", "bar", 0)

		val, ok := table.Get(ctx, "foo")
		assert.True(t, ok)
		assert.Equal(t, "bar", val)
	})

	t.Run("set with TTL", func(t *testing.T) {
		table, mr := newTestRedisTable(t)
		table.Set(ctx, "foo", "bar", 1*time.Hour)

		val, ok := table.Get(ctx, "foo")
		assert.True(t, ok)
		assert.Equal(t, "bar", val)

		// Verify TTL was set in Redis
		ttl := mr.TTL("test:users:foo")
		assert.True(t, ttl > 0)
	})

	t.Run("overwrite existing key", func(t *testing.T) {
		table, _ := newTestRedisTable(t)
		table.Set(ctx, "foo", "bar", 0)
		table.Set(ctx, "foo", "baz", 0)

		val, _ := table.Get(ctx, "foo")
		assert.Equal(t, "baz", val)
	})

	t.Run("set complex value", func(t *testing.T) {
		table, _ := newTestRedisTable(t)
		table.Set(ctx, "obj", map[string]any{"name": "test", "count": float64(42)}, 0)

		val, ok := table.Get(ctx, "obj")
		assert.True(t, ok)
		m, ok := val.(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, "test", m["name"])
		assert.Equal(t, float64(42), m["count"])
	})
}

func TestRedisTable_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("delete existing key", func(t *testing.T) {
		table, _ := newTestRedisTable(t)
		table.Set(ctx, "foo", "bar", 0)

		table.Delete(ctx, "foo")

		_, ok := table.Get(ctx, "foo")
		assert.False(t, ok)
	})

	t.Run("delete non-existing key", func(t *testing.T) {
		table, _ := newTestRedisTable(t)
		// Should not panic
		table.Delete(ctx, "nonexistent")
	})
}

func TestRedisTable_Data(t *testing.T) {
	ctx := context.Background()

	t.Run("returns all data", func(t *testing.T) {
		table, _ := newTestRedisTable(t)
		table.Set(ctx, "foo", "bar", 0)
		table.Set(ctx, "baz", float64(123), 0)

		data := table.Data(ctx)

		assert.Len(t, data, 2)
		assert.Equal(t, "bar", data["foo"])
		assert.Equal(t, float64(123), data["baz"])
	})

	t.Run("empty table", func(t *testing.T) {
		table, _ := newTestRedisTable(t)

		data := table.Data(ctx)

		assert.Empty(t, data)
	})

	t.Run("skips invalid json entries", func(t *testing.T) {
		table, mr := newTestRedisTable(t)
		table.Set(ctx, "valid", "value", 0)

		// Inject invalid JSON directly
		_ = mr.Set("test:users:invalid", "not-valid-json{")

		data := table.Data(ctx)

		// Should only contain the valid entry
		assert.Len(t, data, 1)
		assert.Equal(t, "value", data["valid"])
	})
}

func TestRedisTable_Clear(t *testing.T) {
	ctx := context.Background()

	t.Run("clears all data", func(t *testing.T) {
		table, _ := newTestRedisTable(t)
		table.Set(ctx, "foo", "bar", 0)
		table.Set(ctx, "baz", "qux", 0)

		table.Clear(ctx)

		data := table.Data(ctx)
		assert.Empty(t, data)
	})

	t.Run("clear empty table", func(t *testing.T) {
		table, _ := newTestRedisTable(t)
		// Should not panic
		table.Clear(ctx)
	})
}
