//go:build !integration

package connexions

import (
	assert2 "github.com/stretchr/testify/assert"
	"testing"
)

func TestIsMap(t *testing.T) {
	assert := assert2.New(t)

	m1 := map[string]any{}
	assert.True(IsMap(m1))

	assert.False(IsMap("hello"))
	assert.False(IsMap(123))
	assert.False(IsMap(123.456))
	assert.False(IsMap(true))
}

func TestGetValueByDottedPath(t *testing.T) {
	data := map[string]any{
		"foo": map[string]any{
			"bar": map[string]any{
				"var": "hello",
			},
		},
		"rice": "nice",
		"mice": []string{"dice"},
	}

	tests := []struct {
		dottedPath string
		expected   interface{}
	}{
		{"foo.bar.var", "hello"},
		{"foo.bar.nonexistent", nil},
		{"foo.nonexistent.var", nil},
		{"nonexistent.bar.var", nil},
		{"rice", "nice"},
		{"mice.dice", nil},
	}

	for _, test := range tests {
		result := GetValueByDottedPath(data, test.dottedPath)
		if result != test.expected {
			t.Errorf("For path %s, expected %v but got %v", test.dottedPath, test.expected, result)
		}
	}
}

func TestSetValueByDottedPath(t *testing.T) {
	assert := assert2.New(t)

	tests := []struct {
		dottedPath       string
		value            any
		src              map[string]any
		expectedNewData  any
		expectedNewValue any
	}{
		{"nice.dice.quite", "rice",
			map[string]any{
				"nice": map[string]any{
					"dice": map[string]any{
						"other": "mice",
						"quite": "not rice"}}},
			map[string]any{
				"nice": map[string]any{
					"dice": map[string]any{
						"other": "mice",
						"quite": "rice"}}},
			"rice",
		},
		{
			"foo.bar", "new value",
			map[string]any{
				"foo": []string{"bar", "car"},
			},
			map[string]any{
				"foo": []string{"bar", "car"},
			},
			nil,
		},
		{"foo.bar.var", "new value",
			map[string]any{},
			map[string]any{
				"foo": map[string]any{
					"bar": map[string]any{
						"var": "new value"}}},
			"new value",
		},
	}

	for _, test := range tests {
		data := test.src
		SetValueByDottedPath(data, test.dottedPath, test.value)

		result := GetValueByDottedPath(data, test.dottedPath)

		if result != test.expectedNewValue {
			t.Errorf("For path %s, expected %v but got %v", test.dottedPath, test.expectedNewValue, result)
		}
		assert.Equal(test.expectedNewData, data)
	}
}

func TestGetRandomKeyFromMap(t *testing.T) {
	assert := assert2.New(t)

	t.Run("happy-path", func(t *testing.T) {
		myMap := map[string]bool{
			"key1": true,
			"key2": true,
			"key3": true,
		}

		randomKey := GetRandomKeyFromMap(myMap)

		if randomKey == "" {
			t.Errorf("Expected a non-empty random key, but got an empty key")
		}

		if _, exists := myMap[randomKey]; !exists {
			t.Errorf("Random key %s does not exist in the original map", randomKey)
		}
	})

	t.Run("empty-map", func(t *testing.T) {
		randomKey := GetRandomKeyFromMap[int](nil)
		assert.Equal("", randomKey)
	})
}

func TestGetSortedMapKeys(t *testing.T) {
	assert := assert2.New(t)
	myMap := map[string]bool{
		"key1": true,
		"key2": true,
		"key3": true,
	}

	res := GetSortedMapKeys(myMap)

	assert.Equal([]string{"key1", "key2", "key3"}, res)
}
