package provider

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cubahno/connexions/openapi"
	"github.com/stretchr/testify/require"
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

	t.Run("base case", func(t *testing.T) {
		requestBody := strings.NewReader(`{"name": "Dawg"}`)

		req, err := http.NewRequest(http.MethodPost, "http://example.com/pets", requestBody)
		if err != nil {
			t.Errorf("Error creating GeneratedRequest: %v", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		op := doc.FindOperation(&openapi.OperationDescription{Resource: "/pets", Method: http.MethodPost})
		errs := validator.ValidateRequest(&openapi.GeneratedRequest{
			ContentType: req.Header.Get("Content-Type"),
			Operation:   op,
			Request:     req,
		})

		assert.Nil(errs)
	})

	t.Run("invalid type doc", func(t *testing.T) {
		requestBody := strings.NewReader(`{"name": 1}`)

		req, err := http.NewRequest(http.MethodPost, "http://example.com/pets", requestBody)
		if err != nil {
			t.Errorf("Error creating GeneratedRequest: %v", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		op := doc.FindOperation(&openapi.OperationDescription{Resource: "/pets", Method: http.MethodPost})
		errs := validator.ValidateRequest(&openapi.GeneratedRequest{
			ContentType: req.Header.Get("Content-Type"),
			Operation:   op,
			Request:     req,
		})
		expectedErrors := []string{`value must be a string`}

		assert.Equal(t, len(expectedErrors), len(errs))
		for i, expectedErr := range expectedErrors {
			assert.Contains(t, errs[i].Error(), expectedErr)
		}
	})

	t.Run("missing required", func(t *testing.T) {
		requestBody := strings.NewReader(`{"foo": "bar"}`)

		req, err := http.NewRequest(http.MethodPost, "http://example.com/pets", requestBody)
		if err != nil {
			t.Errorf("Error creating GeneratedRequest: %v", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		op := doc.FindOperation(&openapi.OperationDescription{Resource: "/pets", Method: http.MethodPost})
		errs := validator.ValidateRequest(&openapi.GeneratedRequest{
			ContentType: req.Header.Get("Content-Type"),
			Operation:   op,
			Request:     req,
		})
		expectedErrors := []string{`property "name" is missing`}

		assert.Equal(t, len(expectedErrors), len(errs))
		for i, expectedErr := range expectedErrors {
			assert.Contains(t, errs[i].Error(), expectedErr)
		}
	})
}

func TestValidator_ValidateRequest_NonJSON(t *testing.T) {
	assert := require.New(t)
	testData := filepath.Join("..", "..", "..", "testdata")
	doc, err := NewDocumentFromFile(filepath.Join(testData, "document-connexions.yml"))
	assert.Nil(err)
	validator := NewValidator(doc)

	t.Run("form payload", func(t *testing.T) {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)

		_ = writer.WriteField("path", "petstore")
		_ = writer.Close()

		req, err := http.NewRequest(http.MethodPost, "http://example.com/.ui/import", &body)
		if err != nil {
			t.Errorf("Error creating GeneratedRequest: %v", err)
			return
		}
		req.Header.Set("Content-Type", writer.FormDataContentType())

		op := doc.FindOperation(&openapi.OperationDescription{Resource: "/.ui/import", Method: http.MethodPost})
		errs := validator.ValidateRequest(&openapi.GeneratedRequest{
			ContentType: req.Header.Get("Content-Type"),
			Operation:   op,
			Request:     req,
		})

		assert.Nil(errs)
	})
}

func TestValidator_ValidateResponse(t *testing.T) {
	assert := require.New(t)
	testData := filepath.Join("..", "..", "..", "testdata")
	doc, err := NewDocumentFromFile(filepath.Join(testData, "document-petstore.yml"))
	assert.Nil(err)
	validator := NewValidator(doc)

	t.Run("base case", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "http://example.com/pets", nil)
		op := doc.FindOperation(&openapi.OperationDescription{Resource: "/pets", Method: http.MethodGet})
		res := &openapi.GeneratedResponse{
			StatusCode: http.StatusOK,
			Headers: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Content:     []byte(`[{"id": 1, "name": "Dawg"}]`),
			ContentType: "application/json",
			Request:     req,
			Operation:   op,
		}
		errs := validator.ValidateResponse(res)

		assert.Nil(errs)
	})

	t.Run("invalid type doc", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "http://example.com/pets", nil)
		op := doc.FindOperation(&openapi.OperationDescription{Resource: "/pets", Method: http.MethodGet})
		res := &openapi.GeneratedResponse{
			StatusCode: http.StatusOK,
			Headers: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Content:     []byte(`[{"id": "1", "name": "Dawg"}]`),
			ContentType: "application/json",
			Request:     req,
			Operation:   op,
		}
		errs := validator.ValidateResponse(res)
		expectedErrors := []string{`value must be an integer`}

		assert.Equal(t, len(expectedErrors), len(errs))
		for i, expectedErr := range expectedErrors {
			assert.Contains(t, errs[i].Error(), expectedErr)
		}
	})

	t.Run("invalid type but unsupported response", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "http://example.com/pets", nil)
		op := doc.FindOperation(&openapi.OperationDescription{Resource: "/pets", Method: http.MethodGet})
		res := &openapi.GeneratedResponse{
			StatusCode: http.StatusOK,
			Headers: http.Header{
				"Content-Type": []string{"text/markdown"},
			},
			Content:     []byte(`[{"id": "1", "name": "Dawg"}]`),
			ContentType: "application/json",
			Request:     req,
			Operation:   op,
		}
		errs := validator.ValidateResponse(res)
		expectedErrors := []string{`response header Content-Type has unexpected value: "text/markdown"`}

		assert.Equal(t, len(expectedErrors), len(errs))
		for i, expectedErr := range expectedErrors {
			assert.Contains(t, errs[i].Error(), expectedErr)
		}
	})

	t.Run("empty operation handle", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "http://example.com/pets", nil)
		res := &openapi.GeneratedResponse{
			StatusCode: http.StatusOK,
			Headers: http.Header{
				"Content-Type": []string{"text/markdown"},
			},
			Content:     []byte(`[{"id": "1", "name": "Dawg"}]`),
			ContentType: "text/markdown",
			Request:     req,
		}
		errs := validator.ValidateResponse(res)
		assert.Nil(errs)
	})

	t.Run("no headers", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "http://example.com/pets", nil)
		// op := &KinOperation{Operation: openapi3.NewOperation()}
		op := doc.FindOperation(&openapi.OperationDescription{Resource: "/pets", Method: http.MethodGet})
		res := &openapi.GeneratedResponse{
			StatusCode: http.StatusOK,
			// invalid type
			Content:     []byte(`{"id": "1", "email": "jane.doe@email"}`),
			ContentType: "application/json",
			Request:     req,
			Operation:   op,
		}
		errs := validator.ValidateResponse(res)
		assert.Nil(errs)
	})

	t.Run("no response schema", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "http://example.com/pets", nil)
		// TODO: find the reason in kinOpenAPI
		var op openapi.Operation
		// op := doc.FindOperation(&openapi.OperationDescription{Resource: "/pets", Method: http.MethodGet})

		res := &openapi.GeneratedResponse{
			StatusCode:  http.StatusOK,
			ContentType: "application/json",
			Headers: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Request:   req,
			Operation: op,
		}
		errs := validator.ValidateResponse(res)
		assert.Nil(errs)
	})
}

func TestValidator_ValidateResponse_NonJSON(t *testing.T) {
	assert := require.New(t)
	testData := filepath.Join("..", "..", "..", "testdata")
	doc, err := NewDocumentFromFile(filepath.Join(testData, "document-with-other-responses.yml"))
	assert.Nil(err)
	validator := NewValidator(doc)

	t.Run("plain text", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "http://example.com/about", nil)
		op := doc.FindOperation(&openapi.OperationDescription{Resource: "/about", Method: http.MethodGet})

		res := &openapi.GeneratedResponse{
			StatusCode:  http.StatusOK,
			ContentType: "text/plain",
			Content:     []byte(`Hallo, Welt!`),
			Headers: http.Header{
				"Content-Type": []string{"text/plain"},
			},
			Request:   req,
			Operation: op,
		}
		errs := validator.ValidateResponse(res)

		assert.Nil(errs)
	})
}

func TestValidator_ValidateResponse_NoSchema(t *testing.T) {
	assert := require.New(t)
	testData := filepath.Join("..", "..", "..", "testdata")
	doc, err := NewDocumentFromFile(filepath.Join(testData, "document-without-response.yml"))
	assert.Nil(err)
	validator := NewValidator(doc)

	t.Run("no schema response", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "http://example.com/", nil)
		op := doc.FindOperation(&openapi.OperationDescription{Resource: "/", Method: http.MethodGet})

		res := &openapi.GeneratedResponse{
			StatusCode:  http.StatusOK,
			ContentType: "application/json",
			Content:     nil,
			Headers: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Request:   req,
			Operation: op,
		}
		errs := validator.ValidateResponse(res)
		assert.Nil(errs)
	})
}
