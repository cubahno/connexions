//go:build !integration

package connexions

import (
	"testing"
)

func TestToSnakeCase(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"convertToSnakeCase", "convert_to_snake_case"},
		{"HTTPStatusCode", "http_status_code"},
		{"camelCase", "camel_case"},
		{"snake_case", "snake_case"},
		{"__double__underscore__", "double_underscore"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			actual := ToSnakeCase(tc.input)
			if actual != tc.expected {
				t.Errorf("Expected: %s, Got: %s", tc.expected, actual)
			}
		})
	}
}

func TestMaybeRegexPattern(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"\\d+", true},
		{"word\\b", true},
		{"[a-z]", true},
		{"(", true},
		{"_id$", true},
		{"abc", false},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := MaybeRegexPattern(test.input)
			if result != test.expected {
				t.Errorf("For input '%s', expected %v but got %v", test.input, test.expected, result)
			}
		})
	}
}
