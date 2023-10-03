//go:build !integration

package connexions

import (
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	assert2 "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

func TestDocument(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	kinDoc, err := NewKinDocumentFromFile(filepath.Join("testdata", "document-petstore.yml"))
	assert.Nil(err)
	libDoc, err := NewLibOpenAPIDocumentFromFile(filepath.Join("testdata", "document-petstore.yml"))
	assert.Nil(err)

	docs := []Document{
		kinDoc,
		libDoc,
	}

	for _, doc := range docs {
		t.Run("GetVersion", func(t *testing.T) {
			assert.Equal("3.0.0", doc.GetVersion())
		})

		t.Run("GetResources", func(t *testing.T) {
			res := doc.GetResources()
			expected := map[string][]string{
				"/pets":      {"GET", "POST"},
				"/pets/{id}": {"GET", "DELETE"},
			}
			assert.ElementsMatch(expected["/pets"], res["/pets"])
			assert.ElementsMatch(expected["/pets/{id}"], res["/pets/{id}"])
		})

		t.Run("FindOperation", func(t *testing.T) {
			op := doc.FindOperation(&OperationDescription{"", "/pets", "GET"})
			assert.NotNil(op)

			assert.Equal(2, len(op.GetParameters()))
		})

		t.Run("FindOperation-res-not-found", func(t *testing.T) {
			op := doc.FindOperation(&OperationDescription{"", "/pets2", "GET"})
			assert.Nil(op)
		})

		t.Run("FindOperation-method-not-found", func(t *testing.T) {
			op := doc.FindOperation(&OperationDescription{"", "/pets", "PATCH"})
			assert.Nil(op)
		})
	}
}

func TestOperation(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	petStorePath := filepath.Join("testdata", "document-petstore.yml")
	withFriendsPath := filepath.Join("testdata", "document-person-with-friends.yml")

	kinDoc, err := NewKinDocumentFromFile(petStorePath)
	assert.Nil(err)
	kinDocWithFriends, err := NewKinDocumentFromFile(withFriendsPath)
	assert.Nil(err)

	libDoc, err := NewLibOpenAPIDocumentFromFile(petStorePath)
	assert.Nil(err)
	libDocWithFriends, err := NewLibOpenAPIDocumentFromFile(withFriendsPath)
	assert.Nil(err)

	for _, doc := range []Document{kinDoc, libDoc} {
		t.Run("FindOperation-with-no-options", func(t *testing.T) {
			op := doc.FindOperation(nil)
			assert.Nil(op)
		})

		t.Run("GetParameters", func(t *testing.T) {
			op := doc.FindOperation(&OperationDescription{"", "/pets", "GET"})
			params := op.GetParameters()

			expected := OpenAPIParameters{
				{
					Name:     "limit",
					In:       ParameterInQuery,
					Required: false,
					Schema: &Schema{
						Type:   TypeInteger,
						Format: "int32",
					},
				},
				{
					Name:     "tags",
					In:       ParameterInQuery,
					Required: false,
					Schema: &Schema{
						Type: "array",
						Items: &Schema{
							Type: "string",
						},
					},
				},
			}

			a, b := GetJSONPair(expected, params)
			if a != b {
				t.Errorf("doc %s: \nexpected / actual: \n%s\n%s", doc.Provider(), a, b)
			}
		})

		t.Run("GetRequestBody", func(t *testing.T) {
			op := doc.FindOperation(&OperationDescription{"", "/pets", "POST"})
			body, contentType := op.GetRequestBody()

			expectedBody := &Schema{
				Type: "object",
				Properties: map[string]*Schema{
					"name": {
						Type: "string",
					},
					"tag": {
						Type: "string",
					},
				},
				Required: []string{"name"},
			}
			assert.NotNil(body)
			assert.Equal("application/json", contentType)

			a, b := GetJSONPair(expectedBody, body)
			if a != b {
				t.Errorf("doc %s: \nexpected / actual: \n%s\n%s", doc.Provider(), a, b)
			}
		})

		t.Run("GetResponse", func(t *testing.T) {
			op := doc.FindOperation(&OperationDescription{"", "/pets", "GET"})
			res := op.GetResponse()
			content := res.Content
			contentType := res.ContentType

			var props []string
			for name := range content.Items.Properties {
				props = append(props, name)
			}

			assert.Equal("application/json", contentType)
			assert.NotNil(content.Items)
			assert.Equal("array", content.Type)
			assert.Equal("object", content.Items.Type)
			assert.ElementsMatch([]string{"name", "tag", "id"}, props)
		})
	}

	for _, docWithFriends := range []Document{
		kinDocWithFriends,
		libDocWithFriends,
	} {
		t.Run("GetRequestBody-empty", func(t *testing.T) {
			op := docWithFriends.FindOperation(&OperationDescription{"", "/person/{id}/find", "POST"})
			body, contentType := op.GetRequestBody()

			assert.Nil(body)
			assert.Equal("", contentType)
		})

		t.Run("GetRequestBody-empty-content", func(t *testing.T) {
			op := docWithFriends.FindOperation(&OperationDescription{"", "/person/{id}/find", "DELETE"})
			body, contentType := op.GetRequestBody()

			assert.Nil(body)
			assert.Equal("", contentType)
		})

		t.Run("GetRequestBody-with-xml-type", func(t *testing.T) {
			op := docWithFriends.FindOperation(&OperationDescription{"", "/person/{id}/find", "PATCH"})
			body, contentType := op.GetRequestBody()

			expectedBody := &Schema{
				Type: "object",
				Properties: map[string]*Schema{
					"id": {
						Type: TypeInteger,
					},
					"name": {
						Type: TypeString,
					},
				},
			}

			assert.Equal("application/xml", contentType)
			a, b := GetJSONPair(expectedBody, body)
			if a != b {
				t.Errorf("doc %s: \nexpected / actual: \n%s\n%s", docWithFriends.Provider(), a, b)
			}
		})

		t.Run("GetResponse-first-defined-non-default", func(t *testing.T) {
			op := docWithFriends.FindOperation(&OperationDescription{"", "/person/{id}", "GET"})
			assert.NotNil(op)

			res := op.GetResponse()

			expected := &OpenAPIResponse{
				Content: &Schema{
					Type: "object",
					Properties: map[string]*Schema{
						"user": {
							Type: TypeObject,
							Properties: map[string]*Schema{
								"name": {
									Type: TypeString,
								},
							},
						},
					},
				},
				ContentType: "application/json",
				StatusCode:  404,
				Headers: OpenAPIHeaders{
					"x-header": {
						Name:     "x-header",
						In:       ParameterInHeader,
						Required: true,
						Schema: &Schema{
							Type: "string",
						},
					},
					"y-header": {
						Name: "y-header",
						In:   ParameterInHeader,
					},
				},
			}

			a, b := GetJSONPair(expected, res)
			if a != b {
				t.Errorf("doc %s: \nexpected / actual: \n%s\n%s", docWithFriends.Provider(), a, b)
			}
		})

		t.Run("GetResponse-default-used", func(t *testing.T) {
			op := docWithFriends.FindOperation(&OperationDescription{"", "/person/{id}", "PUT"})
			assert.NotNil(op)

			res := op.GetResponse()

			expected := &OpenAPIResponse{
				Content: &Schema{
					Type: "object",
					Properties: map[string]*Schema{
						"code": {
							Type:   TypeInteger,
							Format: "int32",
						},
						"message": {
							Type: TypeString,
						},
					},
					Required: []string{"code", "message"},
				},
				ContentType: "application/json",
				StatusCode:  200,
			}

			a, b := GetJSONPair(expected, res)
			if a != b {
				t.Errorf("doc %s: \nexpected / actual: \n%s\n%s", docWithFriends.Provider(), a, b)
			}
		})

		t.Run("GetResponse-empty", func(t *testing.T) {
			op := docWithFriends.FindOperation(&OperationDescription{"", "/person/{id}", "PATCH"})
			assert.NotNil(op)

			res := op.GetResponse()
			expected := &OpenAPIResponse{StatusCode: http.StatusOK}

			a, b := GetJSONPair(expected, res)
			if a != b {
				t.Errorf("doc %s: \nexpected / actual: \n%s\n%s", docWithFriends.Provider(), a, b)
			}
		})

		t.Run("GetResponse-non-predefined", func(t *testing.T) {
			op := docWithFriends.FindOperation(&OperationDescription{"", "/person/{id}/find", "GET"})
			assert.NotNil(op)

			res := op.GetResponse()

			expected := &OpenAPIResponse{
				Content: &Schema{
					Type: "object",
					Properties: map[string]*Schema{
						"id": {
							Type: TypeInteger,
						},
						"name": {
							Type: TypeString,
						},
					},
					Required: []string{"id"},
				},
				ContentType: "application/xml",
				StatusCode:  200,
			}

			a, b := GetJSONPair(expected, res)
			if a != b {
				t.Errorf("doc %s: \nexpected / actual: \n%s\n%s", docWithFriends.Provider(), a, b)
			}
		})

		t.Run("WithParseConfig", func(t *testing.T) {
			op := docWithFriends.FindOperation(&OperationDescription{"", "/person/{id}/find", "GET"})
			assert.NotNil(op)
			op = op.WithParseConfig(&ParseConfig{OnlyRequired: true})

			res := op.GetResponse()
			expected := &OpenAPIResponse{
				Content: &Schema{
					Type: "object",
					Properties: map[string]*Schema{
						"id": {
							Type: TypeInteger,
						},
					},
					Required: []string{"id"},
				},
				ContentType: "application/xml",
				StatusCode:  200,
			}

			a, b := GetJSONPair(expected, res)
			if a != b {
				t.Errorf("doc %s: \nexpected / actual: \n%s\n%s", docWithFriends.Provider(), a, b)
			}
		})
	}
}

func TestNewDocumentFromFileFactory(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()
	filePath := filepath.Join("testdata", "document-petstore.yml")

	t.Run("KinOpenAPIProvider", func(t *testing.T) {
		res, err := NewDocumentFromFileFactory(KinOpenAPIProvider)(filePath)
		assert.Nil(err)
		assert.Equal(KinOpenAPIProvider, res.Provider())
		assert.Greater(len(res.GetResources()), 0)
	})

	t.Run("LibOpenAPIProvider", func(t *testing.T) {
		res, err := NewDocumentFromFileFactory(LibOpenAPIProvider)(filePath)
		assert.Nil(err)
		assert.Equal(LibOpenAPIProvider, res.Provider())
		assert.Greater(len(res.GetResources()), 0)
	})

	t.Run("unknown-fallbacks-to-LibOpenAPIProvider", func(t *testing.T) {
		res, err := NewDocumentFromFileFactory("unknown")(filePath)
		assert.Nil(err)
		assert.Equal(LibOpenAPIProvider, res.Provider())
		assert.Greater(len(res.GetResources()), 0)
	})
}

func TestFixSchemaTypeTypos(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	type testCase struct {
		name     string
		expected string
	}

	testCases := []testCase{
		{"int", TypeInteger},
		{"float", TypeNumber},
		{"bool", TypeBoolean},
		{"unknown", "unknown"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := FixSchemaTypeTypos(tc.name)
			assert.Equal(tc.expected, res)
		})
	}
}

func TestGetOpenAPITypeFromValue(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	type testCase struct {
		value    any
		expected string
	}

	testCases := []testCase{
		{1, TypeInteger},
		{3.14, TypeNumber},
		{true, TypeBoolean},
		{"string", TypeString},
		{func() {}, ""},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("case-%v", tc.value), func(t *testing.T) {
			res := GetOpenAPITypeFromValue(tc.value)
			assert.Equal(tc.expected, res)
		})
	}
}

type OtherTestDocument struct {
	Document
}

func (d *OtherTestDocument) Provider() SchemaProvider {
	return "other"
}

func TestNewOpenAPIValidator(t *testing.T) {
	assert := require.New(t)
	t.Run("KinOpenAPIProvider", func(t *testing.T) {
		doc, err := NewKinDocumentFromFile(filepath.Join("testdata", "document-petstore.yml"))
		assert.Nil(err)
		res := NewOpenAPIValidator(doc)
		assert.NotNil(res)
		_, ok := res.(*kinOpenAPIValidator)
		assert.True(ok)
	})

	t.Run("LibOpenAPIProvider", func(t *testing.T) {
		doc, err := NewLibOpenAPIDocumentFromFile(filepath.Join("testdata", "document-petstore.yml"))
		assert.Nil(err)
		res := NewOpenAPIValidator(doc)
		assert.NotNil(res)
		_, ok := res.(*libOpenAPIValidator)
		assert.True(ok)
	})

	t.Run("unknown", func(t *testing.T) {
		doc, _ := NewKinDocumentFromFile(filepath.Join("testdata", "document-petstore.yml"))
		kin, _ := doc.(*KinDocument)

		other := &OtherTestDocument{Document: kin}

		res := NewOpenAPIValidator(other)
		assert.Nil(res)
	})
}

func TestOpenAPIValidator_ValidateRequest(t *testing.T) {
	assert := require.New(t)

	filePath := filepath.Join("testdata", "document-petstore.yml")
	kinDoc, err := NewKinDocumentFromFile(filePath)
	assert.Nil(err)

	libDoc, err := NewLibOpenAPIDocumentFromFile(filePath)
	assert.Nil(err)

	kinValidator := NewKinOpenAPIValidator(kinDoc)
	libValidator := NewLibOpenAPIValidator(libDoc)

	type testCase struct {
		doc            Document
		validator      OpenAPIValidator
		expectedErrors []string
	}

	for _, tc := range []testCase{
		{kinDoc, kinValidator, nil},
		{libDoc, libValidator, nil},
	} {
		t.Run(fmt.Sprintf("base-case-doc-%s", tc.doc.Provider()), func(t *testing.T) {
			doc := tc.doc
			validator := tc.validator

			requestBody := strings.NewReader(`{"name": "Dawg"}`)

			req, err := http.NewRequest(http.MethodPost, "http://example.com/pets", requestBody)
			if err != nil {
				t.Errorf("Error creating request: %v", err)
				return
			}
			req.Header.Set("Content-Type", "application/json")

			op := doc.FindOperation(&OperationDescription{"", "/pets", http.MethodPost})
			errs := validator.ValidateRequest(&Request{
				operation: op,
				request:   req,
			})
			assert.Equal(len(tc.expectedErrors), len(errs))
		})
	}

	for _, tc := range []testCase{
		{kinDoc, kinValidator, []string{`value must be a string`}},
		{libDoc, libValidator, []string{`expected string, but got number`}},
	} {
		t.Run(fmt.Sprintf("invalid-type-doc-%s", tc.doc.Provider()), func(t *testing.T) {
			doc := tc.doc
			validator := tc.validator

			requestBody := strings.NewReader(`{"name": 1}`)

			req, err := http.NewRequest(http.MethodPost, "http://example.com/pets", requestBody)
			if err != nil {
				t.Errorf("Error creating request: %v", err)
				return
			}
			req.Header.Set("Content-Type", "application/json")

			op := doc.FindOperation(&OperationDescription{"", "/pets", http.MethodPost})
			errs := validator.ValidateRequest(&Request{
				ContentType: req.Header.Get("Content-Type"),
				operation:   op,
				request:     req,
			})

			assert.Equal(len(tc.expectedErrors), len(errs))
			for i, expectedErr := range tc.expectedErrors {
				assert.Contains(errs[i].Error(), expectedErr)
			}
		})
	}

	for _, tc := range []testCase{
		{kinDoc, kinValidator, []string{`property "name" is missing`}},
		{libDoc, libValidator, []string{`missing properties: 'name'`}},
	} {
		t.Run(fmt.Sprintf("missing-required-doc-%s", tc.doc.Provider()), func(t *testing.T) {
			doc := tc.doc
			validator := tc.validator

			requestBody := strings.NewReader(`{"foo": "bar"}`)

			req, err := http.NewRequest(http.MethodPost, "http://example.com/pets", requestBody)
			if err != nil {
				t.Errorf("Error creating request: %v", err)
				return
			}
			req.Header.Set("Content-Type", "application/json")

			op := doc.FindOperation(&OperationDescription{"", "/pets", http.MethodPost})
			errs := validator.ValidateRequest(&Request{
				ContentType: req.Header.Get("Content-Type"),
				operation:   op,
				request:     req,
			})

			assert.Equal(len(tc.expectedErrors), len(errs))
			for i, expectedErr := range tc.expectedErrors {
				assert.Contains(errs[i].Error(), expectedErr)
			}
		})
	}
}

func TestOpenAPIValidator_ValidateResponse(t *testing.T) {
	assert := require.New(t)

	filePath := filepath.Join("testdata", "document-petstore.yml")
	kinDoc, err := NewKinDocumentFromFile(filePath)
	assert.Nil(err)

	libDoc, err := NewLibOpenAPIDocumentFromFile(filePath)
	assert.Nil(err)

	kinValidator := NewKinOpenAPIValidator(kinDoc)
	libValidator := NewLibOpenAPIValidator(libDoc)

	type testCase struct {
		doc            Document
		validator      OpenAPIValidator
		expectedErrors []string
	}

	for _, tc := range []testCase{
		{kinDoc, kinValidator, nil},
		{libDoc, libValidator, nil},
	} {
		t.Run(fmt.Sprintf("base-case-doc-%s", tc.doc.Provider()), func(t *testing.T) {
			doc := tc.doc
			validator := tc.validator

			req, _ := http.NewRequest(http.MethodGet, "http://example.com/pets", nil)
			op := doc.FindOperation(&OperationDescription{"", "/pets", http.MethodGet})
			res := &Response{
				StatusCode: http.StatusOK,
				Headers: http.Header{
					"Content-Type": []string{"application/json"},
				},
				Content:     []byte(`[{"id": 1, "name": "Dawg"}]`),
				ContentType: "application/json",
				request:     req,
				operation:   op,
			}
			errs := validator.ValidateResponse(res)

			assert.Equal(len(tc.expectedErrors), len(errs))
			for i, expectedErr := range tc.expectedErrors {
				assert.Contains(errs[i].Error(), expectedErr)
			}
		})
	}

	for _, tc := range []testCase{
		{kinDoc, kinValidator, []string{`value must be an integer`}},
		{libDoc, libValidator, []string{`allOf failed`, `expected integer, but got string`}},
	} {
		t.Run(fmt.Sprintf("invalid-type-doc-%s", tc.doc.Provider()), func(t *testing.T) {
			doc := tc.doc
			validator := tc.validator

			req, _ := http.NewRequest(http.MethodGet, "http://example.com/pets", nil)
			op := doc.FindOperation(&OperationDescription{"", "/pets", http.MethodGet})
			res := &Response{
				StatusCode: http.StatusOK,
				Headers: http.Header{
					"Content-Type": []string{"application/json"},
				},
				Content:     []byte(`[{"id": "1", "name": "Dawg"}]`),
				ContentType: "application/json",
				request:     req,
				operation:   op,
			}
			errs := validator.ValidateResponse(res)

			assert.Equal(len(tc.expectedErrors), len(errs))
			for i, expectedErr := range tc.expectedErrors {
				assert.Contains(errs[i].Error(), expectedErr)
			}
		})
	}

	for _, tc := range []testCase{
		{kinDoc, kinValidator, []string{`response header Content-Type has unexpected value: "text/markdown"`}},
		{libDoc, libValidator, []string{`GET / 200 operation response content type 'text/markdown' does not exist`}},
	} {
		t.Run(fmt.Sprintf("invalid-type-but-unsupported-response-type-doc-%s", tc.doc.Provider()), func(t *testing.T) {
			doc := tc.doc
			validator := tc.validator

			req, _ := http.NewRequest(http.MethodGet, "http://example.com/pets", nil)
			op := doc.FindOperation(&OperationDescription{"", "/pets", http.MethodGet})
			res := &Response{
				StatusCode: http.StatusOK,
				Headers: http.Header{
					"Content-Type": []string{"text/markdown"},
				},
				Content:     []byte(`[{"id": "1", "name": "Dawg"}]`),
				ContentType: "application/json",
				request:     req,
				operation:   op,
			}
			errs := validator.ValidateResponse(res)

			assert.Equal(len(tc.expectedErrors), len(errs))
			for i, expectedErr := range tc.expectedErrors {
				assert.Contains(errs[i].Error(), expectedErr)
			}
		})
	}

	for _, tc := range []testCase{
		{kinDoc, kinValidator, nil},
		{libDoc, libValidator, []string{`GET / 200 operation response content type 'text/markdown' does not exist`}},
	} {
		t.Run(fmt.Sprintf("empty-operation-handle-doc-%s", tc.doc.Provider()), func(t *testing.T) {
			validator := tc.validator

			req, _ := http.NewRequest(http.MethodGet, "http://example.com/pets", nil)
			res := &Response{
				StatusCode: http.StatusOK,
				Headers: http.Header{
					"Content-Type": []string{"text/markdown"},
				},
				Content:     []byte(`[{"id": "1", "name": "Dawg"}]`),
				ContentType: "text/markdown",
				request:     req,
			}
			errs := validator.ValidateResponse(res)

			assert.Equal(len(tc.expectedErrors), len(errs))
			for i, expectedErr := range tc.expectedErrors {
				assert.Contains(errs[i].Error(), expectedErr)
			}
		})
	}

	for _, tc := range []testCase{
		{kinDoc, kinValidator, nil},
		{libDoc, libValidator, []string{`GET / 200 operation response content type '' does not exist`}},
	} {
		t.Run(fmt.Sprintf("no-headers-doc-%s", tc.doc.Provider()), func(t *testing.T) {
			doc := tc.doc
			validator := tc.validator

			req, _ := http.NewRequest(http.MethodGet, "http://example.com/pets", nil)
			// op := &KinOperation{Operation: openapi3.NewOperation()}
			op := doc.FindOperation(&OperationDescription{"", "/pets", http.MethodGet})
			res := &Response{
				StatusCode: http.StatusOK,
				// invalid type
				Content:     []byte(`{"id": "1", "email": "jane.doe@email"}`),
				ContentType: "application/json",
				request:     req,
				operation:   op,
			}
			errs := validator.ValidateResponse(res)

			assert.Equal(len(tc.expectedErrors), len(errs))
			for i, expectedErr := range tc.expectedErrors {
				assert.Contains(errs[i].Error(), expectedErr)
			}
		})
	}

	for _, tc := range []testCase{
		{kinDoc, kinValidator, nil},
		{libDoc, libValidator, nil},
	} {
		t.Run(fmt.Sprintf("no-response-schema-doc-%s", tc.doc.Provider()), func(t *testing.T) {
			doc := tc.doc
			validator := tc.validator

			req, _ := http.NewRequest(http.MethodGet, "http://example.com/pets", nil)
			op := doc.FindOperation(&OperationDescription{"", "/pets", http.MethodGet})
			if doc.Provider() == KinOpenAPIProvider {
				op = &KinOperation{Operation: openapi3.NewOperation()}
			}

			res := &Response{
				StatusCode:  http.StatusOK,
				ContentType: "application/json",
				Headers: http.Header{
					"Content-Type": []string{"application/json"},
				},
				request:   req,
				operation: op,
			}
			errs := validator.ValidateResponse(res)

			assert.Equal(len(tc.expectedErrors), len(errs))
			for i, expectedErr := range tc.expectedErrors {
				assert.Contains(errs[i].Error(), expectedErr)
			}
		})
	}
}

func TestOpenAPIValidator_ValidateResponse_NonJson(t *testing.T) {
	assert := require.New(t)

	filePath := filepath.Join("testdata", "document-with-other-responses.yml")
	kinDoc, err := NewKinDocumentFromFile(filePath)
	assert.Nil(err)

	libDoc, err := NewLibOpenAPIDocumentFromFile(filePath)
	assert.Nil(err)

	kinValidator := NewKinOpenAPIValidator(kinDoc)
	libValidator := NewLibOpenAPIValidator(libDoc)

	type testCase struct {
		doc            Document
		validator      OpenAPIValidator
		expectedErrors []string
	}

	for _, tc := range []testCase{
		{kinDoc, kinValidator, nil},
		{libDoc, libValidator, nil},
	} {
		t.Run(fmt.Sprintf("text-plain-doc-%s", tc.doc.Provider()), func(t *testing.T) {
			doc := tc.doc
			validator := tc.validator

			req, _ := http.NewRequest(http.MethodGet, "http://example.com/about", nil)
			op := doc.FindOperation(&OperationDescription{"", "/about", http.MethodGet})

			res := &Response{
				StatusCode:  http.StatusOK,
				ContentType: "text/plain",
				Content:     []byte(`Hallo, Welt!`),
				Headers: http.Header{
					"Content-Type": []string{"text/plain"},
				},
				request:   req,
				operation: op,
			}
			errs := validator.ValidateResponse(res)

			assert.Equal(len(tc.expectedErrors), len(errs))
			for i, expectedErr := range tc.expectedErrors {
				assert.Contains(errs[i].Error(), expectedErr)
			}
		})
	}
}
