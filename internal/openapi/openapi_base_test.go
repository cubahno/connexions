package openapi

import (
	"fmt"
	"testing"

	"github.com/cubahno/connexions/internal/types"
	assert2 "github.com/stretchr/testify/assert"
)

func TestFixSchemaTypeTypos(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	type testCase struct {
		name     string
		expected string
	}

	testCases := []testCase{
		{"int", types.TypeInteger},
		{"float", types.TypeNumber},
		{"bool", types.TypeBoolean},
		{"unknown", "unknown"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := FixSchemaTypeTypos(tc.name)
			assert.Equal(tc.expected, res)
		})
	}
}

func TestGetOpenAPITypeFromValue(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	type testCase struct {
		value    any
		expected string
	}

	testCases := []testCase{
		{1, types.TypeInteger},
		{3.14, types.TypeNumber},
		{true, types.TypeBoolean},
		{"string", types.TypeString},
		{func() {}, ""},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("case-%v", tc.value), func(t *testing.T) {
			res := GetOpenAPITypeFromValue(tc.value)
			assert.Equal(tc.expected, res)
		})
	}
}

func TestTransformHTTPCode(t *testing.T) {
	assert := assert2.New(t)

	type tc struct {
		name     string
		expected int
	}
	testCases := []tc{
		{"200", 200},
		{"2xx", 200},
		{"2XX", 200},
		{"default", 200},
		{"20x", 200},
		{"201", 201},
		{"*", 200},
		{"unknown", 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(tc.expected, TransformHTTPCode(tc.name))
		})
	}
}
