package db

import (
	"testing"

	assert2 "github.com/stretchr/testify/assert"
)

func TestMemoryTable_Get(t *testing.T) {
	assert := assert2.New(t)

	t.Run("get existing key", func(t *testing.T) {
		table := newMemoryTable()
		table.data["foo"] = "bar"

		val, ok := table.Get("foo")
		assert.True(ok)
		assert.Equal("bar", val)
	})

	t.Run("get non-existing key", func(t *testing.T) {
		table := newMemoryTable()

		val, ok := table.Get("foo")
		assert.False(ok)
		assert.Nil(val)
	})
}

func TestMemoryTable_Set(t *testing.T) {
	assert := assert2.New(t)

	t.Run("set new key", func(t *testing.T) {
		table := newMemoryTable()
		table.Set("foo", "bar")

		assert.Equal("bar", table.data["foo"])
	})

	t.Run("overwrite existing key", func(t *testing.T) {
		table := newMemoryTable()
		table.Set("foo", "bar")
		table.Set("foo", "baz")

		assert.Equal("baz", table.data["foo"])
	})
}

func TestMemoryTable_Delete(t *testing.T) {
	assert := assert2.New(t)

	t.Run("delete existing key", func(t *testing.T) {
		table := newMemoryTable()
		table.data["foo"] = "bar"

		table.Delete("foo")

		_, ok := table.data["foo"]
		assert.False(ok)
	})

	t.Run("delete non-existing key", func(t *testing.T) {
		table := newMemoryTable()

		// Should not panic
		table.Delete("foo")
	})
}

func TestMemoryTable_Data(t *testing.T) {
	assert := assert2.New(t)

	t.Run("returns copy of data", func(t *testing.T) {
		table := newMemoryTable()
		table.data["foo"] = "bar"
		table.data["baz"] = 123

		data := table.Data()

		assert.Len(data, 2)
		assert.Equal("bar", data["foo"])
		assert.Equal(123, data["baz"])

		// Modifying returned map should not affect original
		data["foo"] = "modified"
		assert.Equal("bar", table.data["foo"])
	})

	t.Run("empty table", func(t *testing.T) {
		table := newMemoryTable()

		data := table.Data()

		assert.Empty(data)
	})
}

func TestMemoryTable_Clear(t *testing.T) {
	assert := assert2.New(t)

	table := newMemoryTable()
	table.data["foo"] = "bar"
	table.data["baz"] = 123

	table.Clear()

	assert.Empty(table.data)
}
