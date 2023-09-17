//go:build !integration

package connexions

import (
	assert2 "github.com/stretchr/testify/assert"
	"testing"
)

func TestSliceDeleteAtIndex(t *testing.T) {
	assert := assert2.New(t)

	t.Run("base-case", func(t *testing.T) {
		s := []string{"a", "b", "c"}
		res := SliceDeleteAtIndex(s, 1)
		assert.Equal([]string{"a", "c"}, res)
	})

	t.Run("non-ex-ix", func(t *testing.T) {
		s := []string{"a", "b", "c"}
		res := SliceDeleteAtIndex(s, 10)
		assert.Equal([]string{"a", "b", "c"}, res)
	})
}

func TestGetRandomSliceValue(t *testing.T) {
	assert := assert2.New(t)

	t.Run("string", func(t *testing.T) {
		slice := []string{"a", "b", "c"}
		res := GetRandomSliceValue(slice)

		assert.Contains(slice, res)
	})

	t.Run("any", func(t *testing.T) {
		type i int
		slice := []any{"a", "b", "c"}
		res := GetRandomSliceValue(slice)

		assert.Contains(slice, res)
	})

	t.Run("empty", func(t *testing.T) {
		var slice []int
		res := GetRandomSliceValue(slice)
		assert.Equal(0, res)
	})
}

func TestSliceContains(t *testing.T) {
	assert := assert2.New(t)

	t.Run("base", func(t *testing.T) {
		slice := []string{"a", "b", "c"}

		resTrue := SliceContains(slice, "b")
		assert.True(resTrue)

		resFalse := SliceContains(slice, "d")
		assert.False(resFalse)
	})
}

func TestSliceUnique(t *testing.T) {
	assert := assert2.New(t)

	t.Run("string", func(t *testing.T) {
		slice := []string{"a", "b", "c", "a", "b", "c"}
		res := SliceUnique(slice)

		assert.Len(res, 3)
	})

	t.Run("any", func(t *testing.T) {
		slice := []any{"a", "b", "c", "a", "b", "c"}
		res := SliceUnique(slice)

		assert.Len(res, 3)
	})
}

func TestIsSliceUnique(t *testing.T) {
	assert := assert2.New(t)

	t.Run("true", func(t *testing.T) {
		slice := []string{"a", "b", "c"}

		res := IsSliceUnique(slice)
		assert.True(res)
	})

	t.Run("false", func(t *testing.T) {
		slice := []int{1, 2, 2}

		res := IsSliceUnique(slice)
		assert.False(res)
	})

	t.Run("empty", func(t *testing.T) {
		var slice []int

		res := IsSliceUnique(slice)
		assert.True(res)
	})
}

func TestAppendSliceFirstNonEmpty(t *testing.T) {
	assert := assert2.New(t)

	t.Run("has-non-empty-string", func(t *testing.T) {
		slice := []string{"a", "b"}
		res := AppendSliceFirstNonEmpty(slice, "", "c", "d")
		expected := []string{"a", "b", "c"}
		assert.Equal(expected, res)
	})

	t.Run("has-non-empty-int", func(t *testing.T) {
		slice := []int{1, 2, 3}
		res := AppendSliceFirstNonEmpty(slice, 0, 4)
		expected := []int{1, 2, 3, 4}
		assert.Equal(expected, res)
	})

	t.Run("nothing-added", func(t *testing.T) {
		slice := []string{"a", "b"}
		res := AppendSliceFirstNonEmpty(slice, "", "")
		expected := []string{"a", "b"}
		assert.Equal(expected, res)
	})
}
