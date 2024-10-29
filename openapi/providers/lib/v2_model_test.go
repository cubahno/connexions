//go:build !integration

package lib

import (
	"github.com/cubahno/connexions/config"
	"github.com/cubahno/connexions/openapi"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	v2high "github.com/pb33f/libopenapi/datamodel/high/v2"
	assert2 "github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

func TestLibV2Document(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()
	testData := filepath.Join("..", "..", "..", "testdata")
	doc, err := NewDocumentFromFile(filepath.Join(testData, "document-petstore-v2.yml"))
	assert.Nil(err)

	t.Run("GetVersion", func(t *testing.T) {
		assert.Equal("2.0", doc.GetVersion())
	})

	t.Run("Provider", func(t *testing.T) {
		assert.Equal(config.LibOpenAPIProvider, doc.Provider())
	})

	t.Run("GetSecurity", func(t *testing.T) {
		res := doc.GetSecurity()
		expected := openapi.SecurityComponents{
			"api_key": &openapi.SecurityComponent{
				Type:   openapi.AuthTypeApiKey,
				Scheme: openapi.AuthSchemeBearer,
				In:     openapi.AuthLocationHeader,
				Name:   "api_key",
			},
		}
		assert.Equal(expected, res)
	})

	t.Run("GetResources", func(t *testing.T) {
		res := doc.GetResources()
		expected := map[string][]string{
			"/pet":                     {"POST", "PUT"},
			"/pet/findByTags":          {"GET"},
			"/pet/findByStatus":        {"GET"},
			"/pet/{petId}":             {"GET", "POST", "DELETE"},
			"/pet/{petId}/uploadImage": {"POST"},
		}
		assert.ElementsMatch(expected["/pet"], res["/pet"])
		assert.ElementsMatch(expected["/pet/findByTags"], res["/pet/findByTags"])
		assert.ElementsMatch(expected["/pet/findByStatus"], res["/pet/findByStatus"])
		assert.ElementsMatch(expected["/pet/{petId}"], res["/pet/{petId}"])
		assert.ElementsMatch(expected["/pet/{petId}/uploadImage"], res["/pet/{petId}/uploadImage"])
	})

	t.Run("FindOperation", func(t *testing.T) {
		op := doc.FindOperation(&openapi.OperationDescription{Resource: "/pet/findByStatus", Method: "GET"})
		assert.NotNil(op)
		libOp, ok := op.(*V2Operation)
		assert.True(ok)
		assert.NotNil(libOp)

		assert.Equal(2, len(libOp.getParameters(nil)))
		assert.Equal("findPetsByStatus", libOp.OperationId)
	})

	t.Run("FindOperation-res-not-found", func(t *testing.T) {
		op := doc.FindOperation(&openapi.OperationDescription{Resource: "/pets2", Method: "GET"})
		assert.Nil(op)
	})

	t.Run("FindOperation-method-not-found", func(t *testing.T) {
		op := doc.FindOperation(&openapi.OperationDescription{Resource: "/pet", Method: "PATCH"})
		assert.Nil(op)
	})
}

func TestLibV2Operation(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()
	testData := filepath.Join("..", "..", "..", "testdata")
	doc, err := NewDocumentFromFile(filepath.Join(testData, "document-petstore-v2.yml"))
	assert.Nil(err)
	docWithFriends, err := NewDocumentFromFile(filepath.Join(testData, "document-person-with-friends-v2.yml"))
	assert.Nil(err)

	t.Run("ID", func(t *testing.T) {
		operation := &V2Operation{Operation: &v2high.Operation{OperationId: "findNice"}}
		res := operation.ID()
		assert.Equal("findNice", res)
	})

	t.Run("FindOperation-with-no-options", func(t *testing.T) {
		op := doc.FindOperation(nil)
		assert.Nil(op)
	})

	t.Run("getParameters", func(t *testing.T) {
		op, _ := doc.FindOperation(&openapi.OperationDescription{Resource: "/pet/findByStatus", Method: "GET"}).(*V2Operation)
		params := op.getParameters(nil)

		expected := openapi.Parameters{
			{
				Name: "limit",
				In:   openapi.ParameterInQuery,
				Schema: &openapi.Schema{
					Type: openapi.TypeInteger,
				},
			},
			{
				Name:     "status",
				In:       openapi.ParameterInQuery,
				Required: true,
				Schema: &openapi.Schema{
					Type: openapi.TypeArray,
					Items: &openapi.Schema{
						Type:    openapi.TypeString,
						Default: "available",
						Enum:    []any{"available", "pending", "sold"},
					},
					Default: "available",
				},
			},
		}

		AssertJSONEqual(t, expected, params)
	})

	t.Run("getRequestBody", func(t *testing.T) {
		op, _ := doc.FindOperation(&openapi.OperationDescription{Resource: "/pet", Method: "POST"}).(*V2Operation)
		assert.NotNil(op)
		body, contentType := op.getRequestBody()

		expectedBody := &openapi.Schema{
			Type: "object",
			Properties: map[string]*openapi.Schema{
				"id": {
					Type:   openapi.TypeInteger,
					Format: "int64",
				},
				"name": {
					Type:    "string",
					Example: "doggie",
				},
				"tags": {
					Type: "array",
					Items: &openapi.Schema{
						Type: openapi.TypeObject,
						Properties: map[string]*openapi.Schema{
							"id": {
								Type:   openapi.TypeInteger,
								Format: "int64",
							},
							"name": {
								Type: openapi.TypeString,
							},
						},
					},
				},
				"status": {
					Type: openapi.TypeString,
					Enum: []any{"available", "pending", "sold"},
				},
				"category": {
					Type: openapi.TypeObject,
					Properties: map[string]*openapi.Schema{
						"id": {
							Type:   openapi.TypeInteger,
							Format: "int64",
						},
						"name": {
							Type: openapi.TypeString,
						},
					},
				},
				"photoUrls": {
					Type: openapi.TypeArray,
					Items: &openapi.Schema{
						Type: openapi.TypeString,
					},
				},
			},
			Required: []string{"name", "photoUrls"},
		}
		assert.NotNil(body)
		assert.Equal("application/xml", contentType)
		AssertJSONEqual(t, expectedBody, body)
	})

	t.Run("getRequestBody-empty", func(t *testing.T) {
		op, _ := docWithFriends.FindOperation(&openapi.OperationDescription{Resource: "/person/{id}/find", Method: "POST"}).(*V2Operation)
		body, contentType := op.getRequestBody()

		assert.Nil(body)
		assert.Equal("", contentType)
	})

	t.Run("getRequestBody-empty-content", func(t *testing.T) {
		op, _ := docWithFriends.FindOperation(&openapi.OperationDescription{Resource: "/person/{id}/find", Method: "DELETE"}).(*V2Operation)
		assert.NotNil(op)
		body, contentType := op.getRequestBody()

		assert.Nil(body)
		assert.Equal("application/json", contentType)
	})

	t.Run("getRequestBody-with-xml-type", func(t *testing.T) {
		op, _ := docWithFriends.FindOperation(&openapi.OperationDescription{Resource: "/person/{id}/find", Method: "PATCH"}).(*V2Operation)
		body, contentType := op.getRequestBody()

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

		AssertJSONEqual(t, expectedBody, body)
		assert.Equal("application/xml", contentType)
	})

	t.Run("GetResponse", func(t *testing.T) {
		op := doc.FindOperation(&openapi.OperationDescription{Resource: "/pet/findByStatus", Method: "GET"})
		assert.NotNil(op)
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
		assert.ElementsMatch([]string{"name", "tags", "id", "photoUrls", "category", "status"}, props)
	})

	t.Run("GetResponse-first-defined-non-default", func(t *testing.T) {
		op := docWithFriends.FindOperation(&openapi.OperationDescription{Resource: "/person/{id}", Method: "GET"})
		assert.NotNil(op)

		res := op.GetResponse()

		expected := &openapi.Response{
			Content: &openapi.Schema{
				Type: openapi.TypeObject,
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
					Name: "x-header",
					In:   openapi.ParameterInHeader,
					// required is not there in libopenapi for swagger
					// Required: true,
					Schema: &openapi.Schema{
						Type: "string",
					},
				},
				"y-header": {
					Name: "y-header",
					In:   openapi.ParameterInHeader,
					Schema: &openapi.Schema{
						Type: "string",
					},
				},
			},
		}

		AssertJSONEqual(t, expected, res)
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

		AssertJSONEqual(t, expected, res)
	})

	t.Run("GetResponse-empty", func(t *testing.T) {
		op := docWithFriends.FindOperation(&openapi.OperationDescription{Resource: "/person/{id}", Method: "PATCH"})
		assert.NotNil(op)

		res := op.GetResponse()

		expected := &openapi.Response{}

		AssertJSONEqual(t, expected, res)
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
			ContentType: "application/json",
			StatusCode:  200,
		}

		AssertJSONEqual(t, expected, res)
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
			ContentType: "application/json",
			StatusCode:  200,
		}

		AssertJSONEqual(t, expected, res)
	})

	t.Run("parseParameter", func(t *testing.T) {
		t.Skip("Fix this test with correct types")
		op := docWithFriends.FindOperation(&openapi.OperationDescription{Resource: "/person/{id}/find", Method: "GET"}).(*V2Operation)
		assert.NotNil(op)

		minimum := 1
		maximum := 10
		minItems := 1
		maxItems := 10
		multipleOf := 2
		minLength := 2
		maxLength := 4
		required := true

		libParam := v2high.Parameter{
			Name:       "limit",
			In:         openapi.ParameterInQuery,
			Type:       openapi.TypeInteger,
			Format:     "int64",
			Required:   &required,
			Default:    nil, // 10
			Minimum:    &minimum,
			Maximum:    &maximum,
			Enum:       nil, //[]any{10, 20, 30},
			MinItems:   &minItems,
			MaxItems:   &maxItems,
			MinLength:  &minLength,
			MaxLength:  &maxLength,
			Pattern:    "^[a-zA-Z0-9]+$",
			MultipleOf: &multipleOf,
			Items: &v2high.Items{
				Type: openapi.TypeString,
			},
		}

		expected := &openapi.Schema{
			Type: openapi.TypeInteger,
			Items: &openapi.Schema{
				Type: openapi.TypeString,
			},
			MultipleOf: 2,
			Maximum:    10,
			Minimum:    1,
			MaxLength:  4,
			MinLength:  2,
			Pattern:    "^[a-zA-Z0-9]+$",
			Format:     "int64",
			MaxItems:   10,
			MinItems:   1,
			Enum:       []any{10, 20, 30},
			Default:    10,
		}

		res := op.parseParameter(&libParam)

		assert.Equal(expected, res)
	})

	t.Run("parseParameter-with-schema", func(t *testing.T) {
		op := docWithFriends.FindOperation(&openapi.OperationDescription{Resource: "/person/{id}/find", Method: "GET"}).(*V2Operation)
		assert.NotNil(op)
		libDoc, ok := docWithFriends.(*V2Document)
		assert.True(ok)

		libSchema := libDoc.DocumentModel.Model.Definitions.Definitions.GetOrZero("State").Schema()

		libParam := v2high.Parameter{
			Name:   "limit",
			In:     openapi.ParameterInQuery,
			Type:   openapi.TypeObject,
			Schema: base.CreateSchemaProxy(libSchema),
		}

		res := op.parseParameter(&libParam)
		expected := &openapi.Schema{
			Type: openapi.TypeObject,
			Properties: map[string]*openapi.Schema{
				"name": {
					Type: openapi.TypeString,
				},
				"abbr": {
					Type: openapi.TypeString,
				},
			},
		}
		AssertJSONEqual(t, expected, res)
	})
}
