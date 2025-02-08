//go:build !integration

package types

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

func TestToString(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{42, "42"},
		{int64(1234567890), "1234567890"},
		{3.14159265359, "3.14159265359"},
		{uint8(255), "255"},
		{"Hello, world!", "Hello, world!"},
		{true, "true"},
		{nil, ""},
		{map[string]string{"foo": "bar"}, "map[foo:bar]"},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("ToString(%v)", test.input), func(t *testing.T) {
			result := ToString(test.input)
			if result != test.expected {
				t.Errorf("Expected %s, but got %s", test.expected, result)
			}
		})
	}
}

func TestExtractPlaceholders(t *testing.T) {
	tests := []struct {
		url    string
		expect []string
	}{
		{"", []string{}},
		{"/", []string{}},
		{"abc", []string{}},
		{"{user-id}", []string{"{user-id}"}},
		{"{file_id}", []string{"{file_id}"}},
		{"{some_name_1}", []string{"{some_name_1}"}},
		{"{id}/{name}", []string{"{id}", "{name}"}},
		{"/users/{id}/files/{file_id}", []string{"{id}", "{file_id}"}},
		{"{!@#$%^}", []string{"{!@#$%^}"}},
		{"{name:str}/{id}", []string{"{name:str}", "{id}"}},
		{"{name?str}/{id}", []string{"{name?str}", "{id}"}},
		{"{}", []string{"{}"}},
		{"{}/{}", []string{"{}", "{}"}},
	}

	for _, test := range tests {
		result := ExtractPlaceholders(test.url)
		if len(result) != len(test.expect) {
			t.Errorf("For URL Pattern: %s, Expected: %v, Got: %v", test.url, test.expect, result)
		}
		for i, res := range result {
			if res != test.expect[i] {
				t.Errorf("For URL Pattern: %s, Expected: %v, Got: %v", test.url, test.expect, result)
			}
		}
	}
}
