package kin

import (
	"github.com/cubahno/connexions/openapi/providers"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"testing"
)

func TestNewValidator(t *testing.T) {
	assert := require.New(t)

	testData := filepath.Join("..", "..", "..", "testdata")
	doc, err := NewDocumentFromFile(filepath.Join(testData, "document-petstore.yml"))
	assert.Nil(err)
	inst := NewValidator(doc)
	assert.NotNil(inst)
}

func TestValidator_ValidateRequest(t *testing.T) {
	assert := require.New(t)
	testData := filepath.Join("..", "..", "..", "testdata")
	doc, err := NewDocumentFromFile(filepath.Join(testData, "document-petstore.yml"))
	assert.Nil(err)
	validator := NewValidator(doc)

	tc := &providers.RequestValidatorTestCase{
		Doc:       doc,
		Validator: validator,
	}

	tc.BaseCase(t, nil)
	tc.InvalidTypeDoc(t, []string{`value must be a string`})
	tc.MissingRequired(t, []string{`property "name" is missing`})
}

func TestValidator_ValidateResponse(t *testing.T) {
	assert := require.New(t)
	testData := filepath.Join("..", "..", "..", "testdata")
	doc, err := NewDocumentFromFile(filepath.Join(testData, "document-petstore.yml"))
	assert.Nil(err)
	validator := NewValidator(doc)

	tc := &providers.ResponseValidatorTestCase{
		Doc:       doc,
		Validator: validator,
	}

	tc.BaseCase(t, nil)
	tc.InvalidTypeDoc(t, []string{`value must be an integer`})
	tc.InvalidTypeButUnsupportedResponse(t, []string{`response header Content-Type has unexpected value: "text/markdown"`})
	tc.EmptyOperationHandle(t, nil)
	tc.NoHeaders(t, nil)
	tc.NoResponseSchema(t, nil)
}

func TestValidator_NonJSON_ValidateResponse(t *testing.T) {
	assert := require.New(t)
	testData := filepath.Join("..", "..", "..", "testdata")
	doc, err := NewDocumentFromFile(filepath.Join(testData, "document-with-other-responses.yml"))
	assert.Nil(err)
	validator := NewValidator(doc)

	tc := &providers.ResponseValidatorNonJsonTestCase{
		Doc:       doc,
		Validator: validator,
	}

	tc.PlainText(t, nil)
}
