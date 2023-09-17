//go:build unit

package connexions

import (
	assert2 "github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestReplaceState(t *testing.T) {
	assert := assert2.New(t)

	t.Run("NewFrom", func(t *testing.T) {
		src := &ReplaceState{
			NamePath: []string{"foo", "bar"},
		}
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
		res := (&ReplaceState{}).WithName("foo")
		assert.Equal([]string{"foo"}, res.NamePath)

		// non-empty initial
		src := &ReplaceState{
			NamePath: []string{"dice", "nice"},
		}

		res = src.WithName("mice")
		assert.Equal([]string{"dice", "nice", "mice"}, res.NamePath)
	})

	t.Run("WithElementIndex", func(t *testing.T) {
		src := &ReplaceState{}

		res := src.WithElementIndex(10)
		assert.Equal(10, res.ElementIndex)
	})

	t.Run("WithHeader", func(t *testing.T) {
		src := &ReplaceState{}

		res := src.WithHeader()
		assert.True(res.IsHeader)
	})

	t.Run("WithPathParam", func(t *testing.T) {
		src := &ReplaceState{}

		res := src.WithPathParam()
		assert.True(res.IsPathParam)
	})

	t.Run("SetAPIResponseContentType", func(t *testing.T) {
		src := &ReplaceState{}

		res := src.WithContentType("application/json")
		assert.Equal("application/json", res.ContentType)
	})
}
