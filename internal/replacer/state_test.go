//go:build !integration

package replacer

import (
	"testing"

	assert2 "github.com/stretchr/testify/assert"
)

func TestNewReplaceState(t *testing.T) {
	assert := assert2.New(t)

	t.Run("no-options", func(t *testing.T) {
		res := NewReplaceState()

		assert.Equal([]string{}, res.NamePath)
		assert.Equal(0, res.ElementIndex)
		assert.False(res.IsHeader)
		assert.False(res.IsPathParam)
		assert.Equal("", res.ContentType)
		assert.False(res.IsContentReadOnly)
	})

	t.Run("with-options", func(t *testing.T) {
		res := NewReplaceState(
			WithName("foo"),
			WithElementIndex(1))

		assert.Equal([]string{"foo"}, res.NamePath)
		assert.Equal(1, res.ElementIndex)
		assert.False(res.IsHeader)
		assert.False(res.IsPathParam)
		assert.Equal("", res.ContentType)
		assert.False(res.IsContentReadOnly)
	})

	t.Run("NewFrom does not share NamePath slice", func(t *testing.T) {
		original := NewReplaceState(WithName("parent"))
		assert.Equal([]string{"parent"}, original.NamePath)

		// Create a new state from the original
		child1 := original.NewFrom(original).WithOptions(WithName("child1"))
		assert.Equal([]string{"parent", "child1"}, child1.NamePath)

		// Original should not be modified
		assert.Equal([]string{"parent"}, original.NamePath, "original NamePath should not be modified")

		// Create another child
		child2 := original.NewFrom(original).WithOptions(WithName("child2"))
		assert.Equal([]string{"parent", "child2"}, child2.NamePath)

		// child1 should not be affected
		assert.Equal([]string{"parent", "child1"}, child1.NamePath, "child1 NamePath should not be modified")

		// Original should still not be modified
		assert.Equal([]string{"parent"}, original.NamePath, "original NamePath should still not be modified")
	})

	t.Run("NewFrom with pre-allocated slice capacity", func(t *testing.T) {
		// Create a slice with extra capacity to trigger the append bug
		namePath := make([]string, 1, 10)
		namePath[0] = "parent"

		original := &ReplaceState{
			NamePath: namePath,
		}
		assert.Equal([]string{"parent"}, original.NamePath)

		// Create children - if NewFrom shares the underlying array, they will interfere
		child1 := original.NewFrom(original).WithOptions(WithName("child1"))
		child2 := original.NewFrom(original).WithOptions(WithName("child2"))

		// All should have correct values
		assert.Equal([]string{"parent"}, original.NamePath, "original should not be modified")
		assert.Equal([]string{"parent", "child1"}, child1.NamePath, "child1 should have correct path")
		assert.Equal([]string{"parent", "child2"}, child2.NamePath, "child2 should have correct path")
	})
}

func TestNewReplaceStateWithName(t *testing.T) {
	assert := assert2.New(t)

	t.Run("empty", func(t *testing.T) {
		res := NewReplaceStateWithName("")

		assert.Equal([]string{""}, res.NamePath)
	})

	t.Run("non-empty", func(t *testing.T) {
		res := NewReplaceStateWithName("foo")

		assert.Equal([]string{"foo"}, res.NamePath)
	})
}

func TestReplaceState(t *testing.T) {
	assert := assert2.New(t)

	t.Run("NewFrom", func(t *testing.T) {
		src := NewReplaceState(WithName("foo"), WithName("bar"))
		got := src.NewFrom(src)

		// Check that NamePath is copied correctly
		assert.Equal([]string{"foo", "bar"}, got.NamePath)

		// Check that SchemaStack is shared (same reference)
		assert.Equal(src.SchemaStack, got.SchemaStack)

		// Check other fields
		assert.Equal(false, got.IsHeader)
		assert.Equal("", got.ContentType)
	})

	t.Run("WithName", func(t *testing.T) {
		// empty initial
		res := (&ReplaceState{}).WithOptions(WithName("foo"))
		assert.Equal([]string{"foo"}, res.NamePath)

		// non-empty initial
		src := &ReplaceState{
			NamePath: []string{"dice", "nice"},
		}

		res = src.WithOptions(WithName("mice"))
		assert.Equal([]string{"dice", "nice", "mice"}, res.NamePath)
	})

	t.Run("WithElementIndex", func(t *testing.T) {
		src := &ReplaceState{}

		res := src.WithOptions(WithElementIndex(10))
		assert.Equal(10, res.ElementIndex)
	})

	t.Run("WithHeader", func(t *testing.T) {
		src := &ReplaceState{}

		res := src.WithOptions(WithHeader())
		assert.True(res.IsHeader)
	})

	t.Run("WithPath", func(t *testing.T) {
		src := &ReplaceState{}

		res := src.WithOptions(WithPath())
		assert.True(res.IsPathParam)
	})

	t.Run("WithContentType", func(t *testing.T) {
		src := &ReplaceState{}

		res := src.WithOptions(WithContentType("application/json"))
		assert.Equal("application/json", res.ContentType)
	})

	t.Run("WithReadOnly", func(t *testing.T) {
		src := &ReplaceState{}

		res := src.WithOptions(WithReadOnly())
		assert.True(res.IsContentReadOnly)
	})

	t.Run("WithWriteOnly", func(t *testing.T) {
		src := &ReplaceState{}

		res := src.WithOptions(WithWriteOnly())
		assert.True(res.IsContentWriteOnly)
	})
}
