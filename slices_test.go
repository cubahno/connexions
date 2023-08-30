package connexions

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSliceUnique(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		slice := []string{"a", "b", "c", "a", "b", "c"}
		res := SliceUnique(slice)

		assert.Len(t, res, 3)
	})

	t.Run("any", func(t *testing.T) {
		slice := []any{"a", "b", "c", "a", "b", "c"}
		res := SliceUnique(slice)

		assert.Len(t, res, 3)
	})
}

func TestGetRandomSliceValue(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		slice := []string{"a", "b", "c"}
		res := GetRandomSliceValue(slice)

		assert.Contains(t, slice, res)
	})

	t.Run("any", func(t *testing.T) {
		type i int
		slice := []any{"a", "b", "c"}
		res := GetRandomSliceValue(slice)

		assert.Contains(t, slice, res)
	})
}
