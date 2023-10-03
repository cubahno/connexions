package openapi

import (
	"fmt"
	assert2 "github.com/stretchr/testify/assert"
	"testing"
)

func TestFixSchemaTypeTypos(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	type testCase struct {
		name     string
		expected string
	}

	testCases := []testCase{
		{"int", TypeInteger},
		{"float", TypeNumber},
		{"bool", TypeBoolean},
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
		{1, TypeInteger},
		{3.14, TypeNumber},
		{true, TypeBoolean},
		{"string", TypeString},
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
