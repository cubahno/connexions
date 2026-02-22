package db

import (
	"context"
	"testing"
	"time"

	assert2 "github.com/stretchr/testify/assert"
)

func TestMemoryTable_Get(t *testing.T) {
	assert := assert2.New(t)
	ctx := context.Background()

	t.Run("get existing key", func(t *testing.T) {
		table := newMemoryTable()
		table.Set(ctx, "foo", "bar", 0)

		val, ok := table.Get(ctx, "foo")
		assert.True(ok)
		assert.Equal("bar", val)
	})

	t.Run("get non-existing key", func(t *testing.T) {
		table := newMemoryTable()

		val, ok := table.Get(ctx, "foo")
		assert.False(ok)
		assert.Nil(val)
	})

	t.Run("get expired key", func(t *testing.T) {
		table := newMemoryTable()
		table.Set(ctx, "foo", "bar", 1*time.Millisecond)

		time.Sleep(5 * time.Millisecond)

		val, ok := table.Get(ctx, "foo")
		assert.False(ok)
		assert.Nil(val)

		// Key should be deleted after Get
		_, exists := table.data["foo"]
		assert.False(exists)
	})
}

func TestMemoryTable_Set(t *testing.T) {
	assert := assert2.New(t)
	ctx := context.Background()

	t.Run("set new key", func(t *testing.T) {
		table := newMemoryTable()
		table.Set(ctx, "foo", "bar", 0)

		entry := table.data["foo"]
		assert.Equal("bar", entry.value)
		assert.True(entry.expiresAt.IsZero())
	})

	t.Run("set with TTL", func(t *testing.T) {
		table := newMemoryTable()
		table.Set(ctx, "foo", "bar", 1*time.Hour)

		entry := table.data["foo"]
		assert.Equal("bar", entry.value)
		assert.False(entry.expiresAt.IsZero())
		assert.True(entry.expiresAt.After(time.Now()))
	})

	t.Run("overwrite existing key", func(t *testing.T) {
		table := newMemoryTable()
		table.Set(ctx, "foo", "bar", 0)
		table.Set(ctx, "foo", "baz", 0)

		assert.Equal("baz", table.data["foo"].value)
	})
}

func TestMemoryTable_Delete(t *testing.T) {
	assert := assert2.New(t)
	ctx := context.Background()

	t.Run("delete existing key", func(t *testing.T) {
		table := newMemoryTable()
		table.Set(ctx, "foo", "bar", 0)

		table.Delete(ctx, "foo")

		_, ok := table.data["foo"]
		assert.False(ok)
	})

	t.Run("delete non-existing key", func(t *testing.T) {
		table := newMemoryTable()

		// Should not panic
		table.Delete(ctx, "foo")
	})
}

func TestMemoryTable_Data(t *testing.T) {
	assert := assert2.New(t)
	ctx := context.Background()

	t.Run("returns copy of data", func(t *testing.T) {
		table := newMemoryTable()
		table.Set(ctx, "foo", "bar", 0)
		table.Set(ctx, "baz", 123, 0)

		data := table.Data(ctx)

		assert.Len(data, 2)
		assert.Equal("bar", data["foo"])
		assert.Equal(123, data["baz"])

		// Modifying returned map should not affect original
		data["foo"] = "modified"
		assert.Equal("bar", table.data["foo"].value)
	})

	t.Run("excludes expired entries", func(t *testing.T) {
		table := newMemoryTable()
		table.Set(ctx, "valid", "value", 0)
		table.Set(ctx, "expired", "old", 1*time.Millisecond)

		time.Sleep(5 * time.Millisecond)

		data := table.Data(ctx)

		assert.Len(data, 1)
		assert.Equal("value", data["valid"])
		_, hasExpired := data["expired"]
		assert.False(hasExpired)
	})

	t.Run("empty table", func(t *testing.T) {
		table := newMemoryTable()

		data := table.Data(ctx)

		assert.Empty(data)
	})
}

func TestMemoryTable_Clear(t *testing.T) {
	assert := assert2.New(t)
	ctx := context.Background()

	table := newMemoryTable()
	table.Set(ctx, "foo", "bar", 0)
	table.Set(ctx, "baz", 123, 0)

	table.Clear(ctx)

	assert.Empty(table.data)
}
