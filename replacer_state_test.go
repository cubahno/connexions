//go:build !integration

package connexions

import (
	assert2 "github.com/stretchr/testify/assert"
	"reflect"
	"testing"
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
		wanted := &ReplaceState{
			NamePath:    []string{"foo", "bar"},
			IsHeader:    false,
			ContentType: "",
		}
		if got := src.NewFrom(src); !reflect.DeepEqual(got, wanted) {
			t.Errorf("NewFrom() = %v, expected %v", got, wanted)
		}
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

	t.Run("WithReadOnly", func(t *testing.T) {
		src := &ReplaceState{}

		res := src.WithOptions(WithReadOnly())
		assert.True(res.IsContentReadOnly)
	})
}
