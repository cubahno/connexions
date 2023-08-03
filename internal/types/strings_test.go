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
		{"navigationservice.e-spirit.cloud", "navigationservice_e_spirit_cloud"},
		{"my-service", "my_service"},
		{"service.name-with.dots-and-hyphens", "service_name_with_dots_and_hyphens"},
		{"IP Push Notification_sandbox", "ip_push_notification_sandbox"},
		{"Payment Service v49", "payment_service_v49"},
		{"My API Service", "my_api_service"},
		{"1password.com", "n_1password_com"}, // Starts with digit
		{"2fa-service", "n_2fa_service"},     // Starts with digit
		{"123test", "n_123test"},             // Starts with digit
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

func TestDeduplicatePathParams(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no placeholders",
			input:    "/users",
			expected: "/users",
		},
		{
			name:     "single placeholder",
			input:    "/users/{id}",
			expected: "/users/{id}",
		},
		{
			name:     "different placeholders",
			input:    "/users/{userId}/posts/{postId}",
			expected: "/users/{userId}/posts/{postId}",
		},
		{
			name:     "duplicate placeholder",
			input:    "/foo/{id}/bar/{id}",
			expected: "/foo/{id}/bar/{id_2}",
		},
		{
			name:     "triple duplicate",
			input:    "/a/{id}/b/{id}/c/{id}",
			expected: "/a/{id}/b/{id_2}/c/{id_3}",
		},
		{
			name:     "multiple different duplicates",
			input:    "/a/{id}/b/{name}/c/{id}/d/{name}",
			expected: "/a/{id}/b/{name}/c/{id_2}/d/{name_2}",
		},
		{
			name:     "empty path",
			input:    "",
			expected: "",
		},
		{
			name:     "root path",
			input:    "/",
			expected: "/",
		},
		{
			name:     "complex path with duplicates",
			input:    "/v1/accounts/{account}/cards/{card}/transactions/{account}",
			expected: "/v1/accounts/{account}/cards/{card}/transactions/{account_2}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DeduplicatePathParams(tt.input)
			if result != tt.expected {
				t.Errorf("DeduplicatePathParams(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestBase64Encode(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "aGVsbG8="},
		{"", ""},
		{"test123", "dGVzdDEyMw=="},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := Base64Encode(tt.input)
			if result != tt.expected {
				t.Errorf("Base64Encode(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizePathForChi(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no wildcards",
			input:    "/foo/bar",
			expected: "/foo/bar",
		},
		{
			name:     "trailing wildcard",
			input:    "/health/*",
			expected: "/health/*",
		},
		{
			name:     "double trailing wildcard",
			input:    "/health/**",
			expected: "/health/*",
		},
		{
			name:     "middle wildcard",
			input:    "/foo/*/bar",
			expected: "/foo/{wildcard}/bar",
		},
		{
			name:     "middle double wildcard",
			input:    "/foo/**/bar",
			expected: "/foo/{wildcard}/bar",
		},
		{
			name:     "multiple middle wildcards",
			input:    "/foo/*/bar/*/baz",
			expected: "/foo/{wildcard}/bar/{wildcard_2}/baz",
		},
		{
			name:     "middle and trailing wildcard",
			input:    "/foo/*/bar/*",
			expected: "/foo/{wildcard}/bar/*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizePathForChi(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizePathForChi(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
