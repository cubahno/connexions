package providers

import (
	"github.com/cubahno/connexions/config"
	"github.com/cubahno/connexions/openapi"
	"github.com/stretchr/testify/assert"
	"net/http"
	"strings"
	"testing"
)

type ResponseValidatorTestCase struct {
	Doc       openapi.Document
	Validator openapi.Validator
}

func (tc *ResponseValidatorTestCase) BaseCase(t *testing.T, expectedErrors []string) {
	t.Helper()
	doc := tc.Doc
	validator := tc.Validator

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

	assert.Equal(t, len(expectedErrors), len(errs))
	for i, expectedErr := range expectedErrors {
		assert.Contains(t, errs[i].Error(), expectedErr)
	}
}

func (tc *ResponseValidatorTestCase) InvalidTypeDoc(t *testing.T, expectedErrors []string) {
	doc := tc.Doc
	validator := tc.Validator

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

	assert.Equal(t, len(expectedErrors), len(errs))
	for i, expectedErr := range expectedErrors {
		assert.Contains(t, errs[i].Error(), expectedErr)
	}
}

func (tc *ResponseValidatorTestCase) InvalidTypeButUnsupportedResponse(t *testing.T, expectedErrors []string) {
	doc := tc.Doc
	validator := tc.Validator

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

	assert.Equal(t, len(expectedErrors), len(errs))
	for i, expectedErr := range expectedErrors {
		assert.Contains(t, errs[i].Error(), expectedErr)
	}
}

func (tc *ResponseValidatorTestCase) EmptyOperationHandle(t *testing.T, expectedErrors []string) {
	validator := tc.Validator

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

	assert.Equal(t, len(expectedErrors), len(errs))
	for i, expectedErr := range expectedErrors {
		assert.Contains(t, errs[i].Error(), expectedErr)
	}
}

func (tc *ResponseValidatorTestCase) NoHeaders(t *testing.T, expectedErrors []string) {
	doc := tc.Doc
	validator := tc.Validator

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

	assert.Equal(t, len(expectedErrors), len(errs))
	for i, expectedErr := range expectedErrors {
		assert.Contains(t, errs[i].Error(), expectedErr)
	}
}

func (tc *ResponseValidatorTestCase) NoResponseSchema(t *testing.T, expectedErrors []string) {
	doc := tc.Doc
	validator := tc.Validator

	req, _ := http.NewRequest(http.MethodGet, "http://example.com/pets", nil)
	op := doc.FindOperation(&openapi.OperationDescription{Resource: "/pets", Method: http.MethodGet})
	if doc.Provider() == config.KinOpenAPIProvider {
		op = nil
	}

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

	assert.Equal(t, len(expectedErrors), len(errs))
	for i, expectedErr := range expectedErrors {
		assert.Contains(t, errs[i].Error(), expectedErr)
	}
}

type ResponseValidatorNonJsonTestCase struct {
	Doc       openapi.Document
	Validator openapi.Validator
}

func (tc *ResponseValidatorNonJsonTestCase) PlainText(t *testing.T, expectedErrors []string) {
	doc := tc.Doc
	validator := tc.Validator

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

	assert.Equal(t, len(expectedErrors), len(errs))
	for i, expectedErr := range expectedErrors {
		assert.Contains(t, errs[i].Error(), expectedErr)
	}
}

type RequestValidatorTestCase struct {
	Doc       openapi.Document
	Validator openapi.Validator
}

func (tc *RequestValidatorTestCase) BaseCase(t *testing.T, expectedErrors []string) {
	doc := tc.Doc
	validator := tc.Validator

	requestBody := strings.NewReader(`{"name": "Dawg"}`)

	req, err := http.NewRequest(http.MethodPost, "http://example.com/pets", requestBody)
	if err != nil {
		t.Errorf("Error creating GeneratedRequest: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	op := doc.FindOperation(&openapi.OperationDescription{Resource: "/pets", Method: http.MethodPost})
	errs := validator.ValidateRequest(&openapi.GeneratedRequest{
		Operation: op,
		Request:   req,
	})

	assert.Equal(t, len(expectedErrors), len(errs))
	for i, expectedErr := range expectedErrors {
		assert.Contains(t, errs[i].Error(), expectedErr)
	}
}

func (tc *RequestValidatorTestCase) InvalidTypeDoc(t *testing.T, expectedErrors []string) {
	doc := tc.Doc
	validator := tc.Validator

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

	assert.Equal(t, len(expectedErrors), len(errs))
	for i, expectedErr := range expectedErrors {
		assert.Contains(t, errs[i].Error(), expectedErr)
	}
}

func (tc *RequestValidatorTestCase) MissingRequired(t *testing.T, expectedErrors []string) {
	doc := tc.Doc
	validator := tc.Validator

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

	assert.Equal(t, len(expectedErrors), len(errs))
	for i, expectedErr := range expectedErrors {
		assert.Contains(t, errs[i].Error(), expectedErr)
	}
}
