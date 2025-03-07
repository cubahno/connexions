//go:build !integration

package types

import (
	"fmt"
	"testing"
)

func TestIsValidHTTPVerb(t *testing.T) {
	tests := []struct {
		verb   string
		expect bool
	}{
		{"", false},
		{"GET", true},
		{"get", true},
		{"head", true},
		{"post", true},
		{"put", true},
		{"patch", true},
		{"delete", true},
		{"connect", true},
		{"options", true},
		{"trace", true},
		{"abc", false},
		{"123", false},
		{"!@#$%^", false},
	}

	for _, test := range tests {
		result := IsValidHTTPVerb(test.verb)
		if result != test.expect {
			t.Errorf("For HTTP Verb: %s, Expected: %v, Got: %v", test.verb, test.expect, result)
		}
	}
}

func TestIsValidURLResource(t *testing.T) {
	tests := []struct {
		url    string
		expect bool
	}{
		{"", true},
		{"/", true},
		{"abc", true},
		{"{user-id}", true},
		{"{file_id}", true},
		{"{some_name_1}", true},
		{"{id}/{name}", true},
		{"/users/{id}/files/{file_id}", true},
		{"{!@#$%^}", false},
		{"{name:str}/{id}", false},
		{"{name?str}/{id}", false},
		{"{}", false},
		{"{}/{}", false},
	}

	for _, test := range tests {
		result := IsValidURLResource(test.url)
		if result != test.expect {
			t.Errorf("For URL Pattern: %s, Expected: %v, Got: %v", test.url, test.expect, result)
		}
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

func TestValidateStringWithInvalidPattern(t *testing.T) {
	result := ValidateStringWithPattern("abc", "[a-z")
	if result {
		t.Errorf("Expected false but got true")
	}
}
