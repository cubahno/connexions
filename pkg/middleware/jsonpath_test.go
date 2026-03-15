package middleware

import (
	"testing"

	assert2 "github.com/stretchr/testify/assert"
)

func TestExtractJSONPath(t *testing.T) {
	assert := assert2.New(t)

	body := []byte(`{
		"data": {
			"name": "Jane",
			"address": {
				"zip": "12345",
				"city": "Springfield"
			},
			"items": [
				{"id": 1, "label": "first"},
				{"id": 2, "label": "second"}
			],
			"count": 42,
			"active": true
		}
	}`)

	tests := []struct {
		name     string
		path     string
		expected any
	}{
		{"simple nested", "data.name", "Jane"},
		{"deep nested", "data.address.zip", "12345"},
		{"array index 0", "data.items[0].label", "first"},
		{"array index 1", "data.items[1].id", float64(2)},
		{"array wildcard", "data.items.label", "first"},
		{"number value", "data.count", float64(42)},
		{"bool value", "data.active", true},
		{"top-level key", "data", map[string]any{
			"name": "Jane",
			"address": map[string]any{
				"zip":  "12345",
				"city": "Springfield",
			},
			"items": []any{
				map[string]any{"id": float64(1), "label": "first"},
				map[string]any{"id": float64(2), "label": "second"},
			},
			"count":  float64(42),
			"active": true,
		}},
		{"non-existent key", "data.missing", nil},
		{"non-existent nested", "data.address.state", nil},
		{"non-existent top", "missing", nil},
		{"array out of bounds", "data.items[5].label", nil},
		{"non-existent deep path", "data.address.state.code", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSONPath(body, tt.path)
			assert.Equal(tt.expected, result)
		})
	}

	t.Run("invalid JSON returns nil", func(t *testing.T) {
		result := extractJSONPath([]byte("not json"), "data.name")
		assert.Nil(result)
	})

	t.Run("nil body returns nil", func(t *testing.T) {
		result := extractJSONPath(nil, "data.name")
		assert.Nil(result)
	})
}

func TestParseDottedPath(t *testing.T) {
	assert := assert2.New(t)

	t.Run("simple path", func(t *testing.T) {
		segments := parseDottedPath("data.name")
		assert.Len(segments, 2)
		assert.Equal("data", segments[0].key)
		assert.Equal(-1, segments[0].index)
		assert.Equal("name", segments[1].key)
	})

	t.Run("array index", func(t *testing.T) {
		segments := parseDottedPath("data.items[0].name")
		assert.Len(segments, 3)
		assert.Equal("items", segments[1].key)
		assert.Equal(0, segments[1].index)
		assert.True(segments[1].isArr)
	})

	t.Run("empty path", func(t *testing.T) {
		segments := parseDottedPath("")
		assert.Empty(segments)
	})
}

func TestFormatValue(t *testing.T) {
	assert := assert2.New(t)

	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"string", "hello", "hello"},
		{"integer float", float64(42), "42"},
		{"decimal float", 3.14, "3.14"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"nil", nil, "<nil>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(tt.expected, formatValue(tt.input))
		})
	}
}
