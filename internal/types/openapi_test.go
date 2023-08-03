package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGoTypeToOpenAPIType(t *testing.T) {
	tests := []struct {
		name     string
		goType   string
		expected string
	}{
		{
			name:     "string type",
			goType:   "string",
			expected: TypeString,
		},
		{
			name:     "bool type",
			goType:   "bool",
			expected: TypeBoolean,
		},
		{
			name:     "int type",
			goType:   "int",
			expected: TypeInteger,
		},
		{
			name:     "int8 type",
			goType:   "int8",
			expected: TypeInteger,
		},
		{
			name:     "int16 type",
			goType:   "int16",
			expected: TypeInteger,
		},
		{
			name:     "int32 type",
			goType:   "int32",
			expected: TypeInteger,
		},
		{
			name:     "int64 type",
			goType:   "int64",
			expected: TypeInteger,
		},
		{
			name:     "uint type",
			goType:   "uint",
			expected: TypeInteger,
		},
		{
			name:     "uint8 type",
			goType:   "uint8",
			expected: TypeInteger,
		},
		{
			name:     "uint16 type",
			goType:   "uint16",
			expected: TypeInteger,
		},
		{
			name:     "uint32 type",
			goType:   "uint32",
			expected: TypeInteger,
		},
		{
			name:     "uint64 type",
			goType:   "uint64",
			expected: TypeInteger,
		},
		{
			name:     "float32 type",
			goType:   "float32",
			expected: TypeNumber,
		},
		{
			name:     "float64 type",
			goType:   "float64",
			expected: TypeNumber,
		},
		{
			name:     "any type",
			goType:   "any",
			expected: TypeString,
		},
		{
			name:     "interface{} type",
			goType:   "interface{}",
			expected: TypeString,
		},
		{
			name:     "array of strings",
			goType:   "[]string",
			expected: TypeArray,
		},
		{
			name:     "array of custom type",
			goType:   "[]SomeCustomType",
			expected: TypeArray,
		},
		{
			name:     "unknown type defaults to object",
			goType:   "CustomType",
			expected: TypeObject,
		},
		{
			name:     "struct type defaults to object",
			goType:   "struct { Name string }",
			expected: TypeObject,
		},
		{
			name:     "empty struct treated as any",
			goType:   "struct{}",
			expected: "any",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GoTypeToOpenAPIType(tt.goType)
			assert.Equal(t, tt.expected, result)
		})
	}
}
