//go:build !integration

package types

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
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

// TestRequiredStructWithBooleanZeroValue tests the behavior of go-playground/validator
// when validating structs with required tags and boolean fields.
//
// This test documents why we always return `true` for boolean fields in our generator:
// When validator.WithRequiredStructEnabled() is used, a struct field marked as `required`
// is considered missing if the struct is a zero value. For a struct with only a boolean field,
// if that boolean is `false` (the zero value), the entire struct is considered zero and
// fails the `required` validation.
//
// This is the issue we encountered with Stripe's billing portal configuration where
// Features.InvoiceHistory is required, but InvoiceHistory{Enabled: false} would fail validation.
func TestRequiredStructWithBooleanZeroValue(t *testing.T) {
	assert := assert.New(t)

	type InvoiceHistory struct {
		Enabled bool `json:"enabled"`
	}

	type Features struct {
		InvoiceHistory InvoiceHistory `json:"invoice_history" validate:"required"`
	}

	validate := validator.New(validator.WithRequiredStructEnabled())

	t.Run("struct with boolean true passes validation", func(t *testing.T) {
		f := Features{
			InvoiceHistory: InvoiceHistory{Enabled: true},
		}
		err := validate.Struct(f)
		assert.NoError(err, "InvoiceHistory{Enabled: true} should pass required validation")
	})

	t.Run("struct with boolean false fails validation", func(t *testing.T) {
		f := Features{
			InvoiceHistory: InvoiceHistory{Enabled: false},
		}
		err := validate.Struct(f)
		assert.Error(err, "InvoiceHistory{Enabled: false} should fail required validation because it's a zero value")
		assert.Contains(err.Error(), "InvoiceHistory", "error should mention the field name")
		assert.Contains(err.Error(), "required", "error should mention the required tag")
	})

	t.Run("zero value struct fails validation", func(t *testing.T) {
		f := Features{}
		err := validate.Struct(f)
		assert.Error(err, "zero value Features should fail required validation")
	})

	t.Run("JSON unmarshal with enabled false fails validation", func(t *testing.T) {
		jsonStr := `{"invoice_history":{"enabled":false}}`
		var f Features
		err := json.Unmarshal([]byte(jsonStr), &f)
		assert.NoError(err, "JSON unmarshal should succeed")

		err = validate.Struct(f)
		assert.Error(err, "unmarshaled struct with enabled=false should fail validation")
		assert.Equal(false, f.InvoiceHistory.Enabled, "enabled should be false")
	})

	t.Run("JSON unmarshal with enabled true passes validation", func(t *testing.T) {
		jsonStr := `{"invoice_history":{"enabled":true}}`
		var f Features
		err := json.Unmarshal([]byte(jsonStr), &f)
		assert.NoError(err, "JSON unmarshal should succeed")

		err = validate.Struct(f)
		assert.NoError(err, "unmarshaled struct with enabled=true should pass validation")
		assert.Equal(true, f.InvoiceHistory.Enabled, "enabled should be true")
	})
}
