package xs

import (
	"fmt"
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

func TestValidateStringWithPattern(t *testing.T) {
	tests := []struct {
		input    string
		pattern  string
		expected bool
	}{
		{"hello", "^h.*", true},
		{"world", "^w.*", true},
		{"123", "^[0-9]*$", true},
		{"purchase_count", "^.*_count", true},
		{"purchase_count", "_count$", true},
		{"abc", "^d.*", false},
		{"123a", "^[0-9]*$", false},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s_%s", test.input, test.pattern), func(t *testing.T) {
			result := ValidateStringWithPattern(test.input, test.pattern)
			if result != test.expected {
				t.Errorf("For input '%s' and pattern '%s', expected %v but got %v", test.input, test.pattern, test.expected, result)
			}
		})
	}
}

func TestMightBeRegexPattern(t *testing.T) {
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
			result := MightBeRegexPattern(test.input)
			if result != test.expected {
				t.Errorf("For input '%s', expected %v but got %v", test.input, test.expected, result)
			}
		})
	}
}
