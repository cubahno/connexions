package connexions

import (
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestReplaceState(t *testing.T) {
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
		src := &ReplaceState{
			NamePath: []string{"dice", "nice"},
		}

		res := src.WithName("mice")
		assert.Equal(t, []string{"dice", "nice", "mice"}, res.NamePath)
	})

	t.Run("WithElementIndex", func(t *testing.T) {
		src := &ReplaceState{}

		res := src.WithElementIndex(10)
		assert.Equal(t, 10, res.ElementIndex)
	})

	t.Run("WithHeader", func(t *testing.T) {
		src := &ReplaceState{}

		res := src.WithHeader()
		assert.True(t, res.IsHeader)
	})

	t.Run("WithContentType", func(t *testing.T) {
		src := &ReplaceState{}

		res := src.WithContentType("application/json")
		assert.Equal(t, "application/json", res.ContentType)
	})
}
