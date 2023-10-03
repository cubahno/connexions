//go:build !integration

package connexions

import (
	"github.com/cubahno/connexions/config"
	"github.com/cubahno/connexions/openapi"
	"github.com/cubahno/connexions/openapi/providers/kin"
	"github.com/cubahno/connexions/openapi/providers/lib"
	assert2 "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"path/filepath"
	"testing"
)

func TestDocument(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	kinDoc, err := kin.NewDocumentFromFile(filepath.Join("testdata", "document-petstore.yml"))
	assert.Nil(err)
	libDoc, err := lib.NewDocumentFromFile(filepath.Join("testdata", "document-petstore.yml"))
	assert.Nil(err)

	docs := []openapi.Document{
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
			op := doc.FindOperation(&openapi.OperationDescription{Resource: "/pets", Method: "GET"})
			assert.NotNil(op)

			assert.Equal(2, len(op.GetParameters()))
		})

		t.Run("FindOperation-res-not-found", func(t *testing.T) {
			op := doc.FindOperation(&openapi.OperationDescription{Resource: "/pets2", Method: "GET"})
			assert.Nil(op)
		})

		t.Run("FindOperation-method-not-found", func(t *testing.T) {
			op := doc.FindOperation(&openapi.OperationDescription{Resource: "/pets", Method: "PATCH"})
			assert.Nil(op)
		})
	}
}

func TestOperation(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	petStorePath := filepath.Join("testdata", "document-petstore.yml")
	withFriendsPath := filepath.Join("testdata", "document-person-with-friends.yml")

	kinDoc, err := kin.NewDocumentFromFile(petStorePath)
	assert.Nil(err)
	kinDocWithFriends, err := kin.NewDocumentFromFile(withFriendsPath)
	assert.Nil(err)

	libDoc, err := lib.NewDocumentFromFile(petStorePath)
	assert.Nil(err)
	libDocWithFriends, err := lib.NewDocumentFromFile(withFriendsPath)
	assert.Nil(err)

	for _, doc := range []openapi.Document{kinDoc, libDoc} {
		t.Run("FindOperation-with-no-options", func(t *testing.T) {
			op := doc.FindOperation(nil)
			assert.Nil(op)
		})

		t.Run("GetParameters", func(t *testing.T) {
			op := doc.FindOperation(&openapi.OperationDescription{Resource: "/pets", Method: "GET"})
			params := op.GetParameters()

			expected := openapi.Parameters{
				{
					Name:     "limit",
					In:       openapi.ParameterInQuery,
					Required: false,
					Schema: &openapi.Schema{
						Type:   openapi.TypeInteger,
						Format: "int32",
					},
				},
				{
					Name:     "tags",
					In:       openapi.ParameterInQuery,
					Required: false,
					Schema: &openapi.Schema{
						Type: "array",
						Items: &openapi.Schema{
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
			op := doc.FindOperation(&openapi.OperationDescription{Resource: "/pets", Method: "POST"})
			body, contentType := op.GetRequestBody()

			expectedBody := &openapi.Schema{
				Type: "object",
				Properties: map[string]*openapi.Schema{
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
			op := doc.FindOperation(&openapi.OperationDescription{Resource: "/pets", Method: "GET"})
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

	for _, docWithFriends := range []openapi.Document{
		kinDocWithFriends,
		libDocWithFriends,
	} {
		t.Run("GetRequestBody-empty", func(t *testing.T) {
			op := docWithFriends.FindOperation(&openapi.OperationDescription{Resource: "/person/{id}/find", Method: "POST"})
			body, contentType := op.GetRequestBody()

			assert.Nil(body)
			assert.Equal("", contentType)
		})

		t.Run("GetRequestBody-empty-content", func(t *testing.T) {
			op := docWithFriends.FindOperation(&openapi.OperationDescription{Resource: "/person/{id}/find", Method: "DELETE"})
			body, contentType := op.GetRequestBody()

			assert.Nil(body)
			assert.Equal("", contentType)
		})

		t.Run("GetRequestBody-with-xml-type", func(t *testing.T) {
			op := docWithFriends.FindOperation(&openapi.OperationDescription{Resource: "/person/{id}/find", Method: "PATCH"})
			body, contentType := op.GetRequestBody()

			expectedBody := &openapi.Schema{
				Type: "object",
				Properties: map[string]*openapi.Schema{
					"id": {
						Type: openapi.TypeInteger,
					},
					"name": {
						Type: openapi.TypeString,
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
			op := docWithFriends.FindOperation(&openapi.OperationDescription{Resource: "/person/{id}", Method: "GET"})
			assert.NotNil(op)

			res := op.GetResponse()

			expected := &openapi.Response{
				Content: &openapi.Schema{
					Type: "object",
					Properties: map[string]*openapi.Schema{
						"user": {
							Type: openapi.TypeObject,
							Properties: map[string]*openapi.Schema{
								"name": {
									Type: openapi.TypeString,
								},
							},
						},
					},
				},
				ContentType: "application/json",
				StatusCode:  404,
				Headers: openapi.Headers{
					"x-header": {
						Name:     "x-header",
						In:       openapi.ParameterInHeader,
						Required: true,
						Schema: &openapi.Schema{
							Type: "string",
						},
					},
					"y-header": {
						Name: "y-header",
						In:   openapi.ParameterInHeader,
					},
				},
			}

			a, b := GetJSONPair(expected, res)
			if a != b {
				t.Errorf("doc %s: \nexpected / actual: \n%s\n%s", docWithFriends.Provider(), a, b)
			}
		})

		t.Run("GetResponse-default-used", func(t *testing.T) {
			op := docWithFriends.FindOperation(&openapi.OperationDescription{Resource: "/person/{id}", Method: "PUT"})
			assert.NotNil(op)

			res := op.GetResponse()

			expected := &openapi.Response{
				Content: &openapi.Schema{
					Type: "object",
					Properties: map[string]*openapi.Schema{
						"code": {
							Type:   openapi.TypeInteger,
							Format: "int32",
						},
						"message": {
							Type: openapi.TypeString,
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
			op := docWithFriends.FindOperation(&openapi.OperationDescription{Resource: "/person/{id}", Method: "PATCH"})
			assert.NotNil(op)

			res := op.GetResponse()
			expected := &openapi.Response{StatusCode: http.StatusOK}

			a, b := GetJSONPair(expected, res)
			if a != b {
				t.Errorf("doc %s: \nexpected / actual: \n%s\n%s", docWithFriends.Provider(), a, b)
			}
		})

		t.Run("GetResponse-non-predefined", func(t *testing.T) {
			op := docWithFriends.FindOperation(&openapi.OperationDescription{Resource: "/person/{id}/find", Method: "GET"})
			assert.NotNil(op)

			res := op.GetResponse()

			expected := &openapi.Response{
				Content: &openapi.Schema{
					Type: "object",
					Properties: map[string]*openapi.Schema{
						"id": {
							Type: openapi.TypeInteger,
						},
						"name": {
							Type: openapi.TypeString,
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
			op := docWithFriends.FindOperation(&openapi.OperationDescription{Resource: "/person/{id}/find", Method: "GET"})
			assert.NotNil(op)
			op = op.WithParseConfig(&config.ParseConfig{OnlyRequired: true})

			res := op.GetResponse()
			expected := &openapi.Response{
				Content: &openapi.Schema{
					Type: "object",
					Properties: map[string]*openapi.Schema{
						"id": {
							Type: openapi.TypeInteger,
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
		res, err := NewDocumentFromFileFactory(config.KinOpenAPIProvider)(filePath)
		assert.Nil(err)
		assert.Equal(config.KinOpenAPIProvider, res.Provider())
		assert.Greater(len(res.GetResources()), 0)
	})

	t.Run("LibOpenAPIProvider", func(t *testing.T) {
		res, err := NewDocumentFromFileFactory(config.LibOpenAPIProvider)(filePath)
		assert.Nil(err)
		assert.Equal(config.LibOpenAPIProvider, res.Provider())
		assert.Greater(len(res.GetResources()), 0)
	})

	t.Run("unknown-fallbacks-to-LibOpenAPIProvider", func(t *testing.T) {
		res, err := NewDocumentFromFileFactory("unknown")(filePath)
		assert.Nil(err)
		assert.Equal(config.LibOpenAPIProvider, res.Provider())
		assert.Greater(len(res.GetResources()), 0)
	})
}

type OtherTestDocument struct {
	openapi.Document
}

func (d *OtherTestDocument) Provider() config.SchemaProvider {
	return "other"
}

func TestNewOpenAPIValidator(t *testing.T) {
	assert := require.New(t)
	t.Run("KinOpenAPIProvider", func(t *testing.T) {
		doc, err := kin.NewDocumentFromFile(filepath.Join("testdata", "document-petstore.yml"))
		assert.Nil(err)
		res := NewOpenAPIValidator(doc)
		assert.NotNil(res)
		_, ok := res.(*kin.Validator)
		assert.True(ok)
	})

	t.Run("LibOpenAPIProvider", func(t *testing.T) {
		doc, err := lib.NewDocumentFromFile(filepath.Join("testdata", "document-petstore.yml"))
		assert.Nil(err)
		res := NewOpenAPIValidator(doc)
		assert.NotNil(res)
		_, ok := res.(*lib.Validator)
		assert.True(ok)
	})

	t.Run("unknown", func(t *testing.T) {
		doc, _ := kin.NewDocumentFromFile(filepath.Join("testdata", "document-petstore.yml"))
		kinDoc, _ := doc.(*kin.Document)

		other := &OtherTestDocument{Document: kinDoc}

		res := NewOpenAPIValidator(other)
		assert.Nil(res)
	})
}
