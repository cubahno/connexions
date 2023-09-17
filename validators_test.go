//go:build !integration

package connexions

import (
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	assert2 "github.com/stretchr/testify/assert"
	"net/http"
	"path/filepath"
	"strings"
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

func TestValidateRequest(t *testing.T) {
	assert := assert2.New(t)
	schema := CreateSchemaFromString(t, `
{"type": "object", 
"required": ["key"],
"properties": 
	{"key": 
		{"type": "string"}
}}`)

	t.Run("base-case", func(t *testing.T) {
		requestBody := strings.NewReader(`{"key": "value"}`)

		req, err := http.NewRequest("POST", "http://example.com/api/resource", requestBody)
		if err != nil {
			t.Errorf("Error creating request: %v", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		err = ValidateRequest(req, schema, "application/json")
		assert.Nil(err)
	})

	t.Run("invalid-type", func(t *testing.T) {
		requestBody := strings.NewReader(`{"key": 1}`)

		req, err := http.NewRequest("POST", "http://example.com/api/resource", requestBody)
		if err != nil {
			t.Errorf("Error creating request: %v", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		err = ValidateRequest(req, schema, "application/json")
		assert.NotNil(err)
		assert.Contains(err.Error(), "value must be a string")
	})

	t.Run("missing-required", func(t *testing.T) {
		requestBody := strings.NewReader(`{"foo": "bar"}`)

		req, err := http.NewRequest("POST", "http://example.com/api/resource", requestBody)
		if err != nil {
			t.Errorf("Error creating request: %v", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		err = ValidateRequest(req, schema, "application/json")
		assert.NotNil(err)
		assert.Contains(err.Error(), `property "key" is missing`)
	})
}

func TestValidateResponse(t *testing.T) {
	assert := assert2.New(t)
	operation := &KinOperation{Operation: openapi3.NewOperation()}
	CreateOperationFromYAMLFile(t, filepath.Join("test_fixtures", "operation-base.yml"), operation)

	t.Run("base-case", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "http://example.com/api/resource", nil)
		res := &Response{
			StatusCode: http.StatusOK,
			Headers: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Content:     []byte(`{"id": 1, "email": "jane.doe@email"}`),
			ContentType: "application/json",
		}

		err := ValidateResponse(req, res, operation)
		assert.Nil(err)
	})

	t.Run("invalid-type", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "http://example.com/api/resource", nil)
		res := &Response{
			StatusCode: http.StatusOK,
			Headers: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Content:     []byte(`{"id": "1", "email": "jane.doe@email"}`),
			ContentType: "application/json",
		}

		err := ValidateResponse(req, res, operation)
		assert.Contains(err.Error(), "value must be an integer")
	})

	t.Run("invalid-type-but-unsupported-response-type", func(t *testing.T) {
		op := &KinOperation{Operation: openapi3.NewOperation()}
		CreateOperationFromYAMLFile(t, filepath.Join("test_fixtures", "operation-base.yml"), op)
		op.Responses["200"].Value.Content["text/markdown"] = op.Responses["200"].Value.Content["application/json"]
		delete(op.Responses["200"].Value.Content, "application/json")

		req, _ := http.NewRequest("GET", "http://example.com/api/resource", nil)
		res := &Response{
			StatusCode: http.StatusOK,
			Headers: http.Header{
				"Content-Type": []string{"text/markdown"},
			},
			Content:     []byte(`{"id": "1", "email": "jane.doe@email"}`),
			ContentType: "text/markdown",
		}

		err := ValidateResponse(req, res, op)
		assert.Nil(err)
	})

	t.Run("no-headers-not-validated", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "http://example.com/api/resource", nil)
		res := &Response{
			StatusCode: http.StatusOK,
			// invalid type
			Content:     []byte(`{"id": "1", "email": "jane.doe@email"}`),
			ContentType: "application/json",
		}

		err := ValidateResponse(req, res, operation)
		assert.Nil(err)
	})

	t.Run("no-response-schema", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "http://example.com/api/resource", nil)
		res := &Response{
			StatusCode:  http.StatusOK,
			ContentType: "application/json",
			Headers: http.Header{
				"Content-Type": []string{"application/json"},
			},
		}

		op := &KinOperation{Operation: openapi3.NewOperation()}
		CreateOperationFromYAMLFile(t, filepath.Join("test_fixtures", "operation-without-response.yml"), op)

		err := ValidateResponse(req, res, op)
		assert.Nil(err)
	})
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
