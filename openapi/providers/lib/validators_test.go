package lib

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

	inst = NewValidator(nil)
	assert.Nil(inst)
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
	tc.InvalidTypeDoc(t, []string{`expected string, but got number`})
	tc.MissingRequired(t, []string{`missing properties: 'name'`})
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
	tc.InvalidTypeDoc(t, []string{`allOf failed`, `expected integer, but got string`})
	tc.InvalidTypeButUnsupportedResponse(t, []string{`GET / 200 operation response content type 'text/markdown' does not exist`})
	tc.EmptyOperationHandle(t, []string{`GET / 200 operation response content type 'text/markdown' does not exist`})
	tc.NoHeaders(t, []string{`GET / 200 operation response content type '' does not exist`})
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
