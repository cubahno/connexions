//go:build !integration

package connexions

import (
	assert2 "github.com/stretchr/testify/assert"
	"net/http"
	"path/filepath"
	"testing"
)

func TestDocument(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	kinDoc, err := NewKinDocumentFromFile(filepath.Join("test_fixtures", "document-petstore.yml"))
	assert.Nil(err)
	libDoc, err := NewLibOpenAPIDocumentFromFile(filepath.Join("test_fixtures", "document-petstore.yml"))
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

	petStorePath := filepath.Join("test_fixtures", "document-petstore.yml")
	withFriendsPath := filepath.Join("test_fixtures", "document-person-with-friends.yml")

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
			for name, _ := range content.Items.Properties {
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
	filePath := filepath.Join("test_fixtures", "document-petstore.yml")

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
