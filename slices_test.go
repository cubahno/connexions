package xs

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

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
