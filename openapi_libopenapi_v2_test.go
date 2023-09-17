package connexions

import (
	"github.com/pb33f/libopenapi/datamodel/high/base"
	v2high "github.com/pb33f/libopenapi/datamodel/high/v2"
	assert2 "github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

func TestLibV2Document(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()
	doc, err := NewLibOpenAPIDocumentFromFile(filepath.Join("test_fixtures", "document-petstore-v2.yml"))
	assert.Nil(err)

	t.Run("GetVersion", func(t *testing.T) {
		assert.Equal("2.0", doc.GetVersion())
	})

	t.Run("Provider", func(t *testing.T) {
		assert.Equal(LibOpenAPIProvider, doc.Provider())
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
		op := doc.FindOperation(&FindOperationOptions{"", "/pet/findByStatus", "GET", nil})
		assert.NotNil(op)
		libOp, ok := op.(*LibV2Operation)
		assert.True(ok)
		assert.NotNil(libOp)

		assert.Equal(2, len(op.GetParameters()))
		assert.Equal("findPetsByStatus", libOp.OperationId)
	})

	t.Run("FindOperation-res-not-found", func(t *testing.T) {
		op := doc.FindOperation(&FindOperationOptions{"", "/pets2", "GET", nil})
		assert.Nil(op)
	})

	t.Run("FindOperation-method-not-found", func(t *testing.T) {
		op := doc.FindOperation(&FindOperationOptions{"", "/pet", "PATCH", nil})
		assert.Nil(op)
	})
}

func TestLibV2Operation(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()
	doc, err := NewLibOpenAPIDocumentFromFile(filepath.Join("test_fixtures", "document-petstore-v2.yml"))
	assert.Nil(err)
	docWithFriends, err := NewLibOpenAPIDocumentFromFile(filepath.Join("test_fixtures", "document-person-with-friends-v2.yml"))
	assert.Nil(err)

	t.Run("FindOperation-with-no-options", func(t *testing.T) {
		op := doc.FindOperation(nil)
		assert.Nil(op)
	})

	t.Run("GetParameters", func(t *testing.T) {
		op := doc.FindOperation(&FindOperationOptions{"", "/pet/findByStatus", "GET", nil})
		params := op.GetParameters()

		expected := OpenAPIParameters{
			{
				Name: "limit",
				In:   ParameterInQuery,
				Schema: &Schema{
					Type: TypeInteger,
				},
			},
			{
				Name:     "status",
				In:       ParameterInQuery,
				Required: true,
				Schema: &Schema{
					Type: TypeArray,
					Items: &Schema{
						Type:    TypeString,
						Default: "available",
						Enum:    []any{"available", "pending", "sold"},
					},
					Default: "available",
				},
			},
		}

		AssertJSONEqual(t, expected, params)
	})

	t.Run("GetRequestBody", func(t *testing.T) {
		op := doc.FindOperation(&FindOperationOptions{"", "/pet", "POST", nil})
		assert.NotNil(op)
		body, contentType := op.GetRequestBody()

		expectedBody := &Schema{
			Type: "object",
			Properties: map[string]*Schema{
				"id": {
					Type:   TypeInteger,
					Format: "int64",
				},
				"name": {
					Type:    "string",
					Example: "doggie",
				},
				"tags": {
					Type: "array",
					Items: &Schema{
						Type: TypeObject,
						Properties: map[string]*Schema{
							"id": {
								Type:   TypeInteger,
								Format: "int64",
							},
							"name": {
								Type: TypeString,
							},
						},
					},
				},
				"status": {
					Type: TypeString,
					Enum: []any{"available", "pending", "sold"},
				},
				"category": {
					Type: TypeObject,
					Properties: map[string]*Schema{
						"id": {
							Type:   TypeInteger,
							Format: "int64",
						},
						"name": {
							Type: TypeString,
						},
					},
				},
				"photoUrls": {
					Type: TypeArray,
					Items: &Schema{
						Type: TypeString,
					},
				},
			},
			Required: []string{"name", "photoUrls"},
		}
		assert.NotNil(body)
		assert.Equal("application/xml", contentType)
		AssertJSONEqual(t, expectedBody, body)
	})

	t.Run("GetRequestBody-empty", func(t *testing.T) {
		op := docWithFriends.FindOperation(&FindOperationOptions{"", "/person/{id}/find", "POST", nil})
		body, contentType := op.GetRequestBody()

		assert.Nil(body)
		assert.Equal("", contentType)
	})

	t.Run("GetRequestBody-empty-content", func(t *testing.T) {
		op := docWithFriends.FindOperation(&FindOperationOptions{"", "/person/{id}/find", "DELETE", nil})
		assert.NotNil(op)
		body, contentType := op.GetRequestBody()

		assert.Nil(body)
		assert.Equal("application/json", contentType)
	})

	t.Run("GetRequestBody-with-xml-type", func(t *testing.T) {
		op := docWithFriends.FindOperation(&FindOperationOptions{"", "/person/{id}/find", "PATCH", nil})
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

		AssertJSONEqual(t, expectedBody, body)
		assert.Equal("application/xml", contentType)
	})

	t.Run("GetResponse", func(t *testing.T) {
		op := doc.FindOperation(&FindOperationOptions{"", "/pet/findByStatus", "GET", nil})
		assert.NotNil(op)
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
		assert.ElementsMatch([]string{"name", "tags", "id", "photoUrls", "category", "status"}, props)
	})

	t.Run("GetResponse-first-defined-non-default", func(t *testing.T) {
		op := docWithFriends.FindOperation(&FindOperationOptions{"", "/person/{id}", "GET", nil})
		assert.NotNil(op)

		res := op.GetResponse()

		expected := &OpenAPIResponse{
			Content: &Schema{
				Type: TypeObject,
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
					Name: "x-header",
					In:   ParameterInHeader,
					// required is not there in libopenapi for swagger
					// Required: true,
					Schema: &Schema{
						Type: "string",
					},
				},
				"y-header": {
					Name: "y-header",
					In:   ParameterInHeader,
					Schema: &Schema{
						Type: "string",
					},
				},
			},
		}

		AssertJSONEqual(t, expected, res)
	})

	t.Run("GetResponse-default-used", func(t *testing.T) {
		op := docWithFriends.FindOperation(&FindOperationOptions{"", "/person/{id}", "PUT", nil})
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

		AssertJSONEqual(t, expected, res)
	})

	t.Run("GetResponse-empty", func(t *testing.T) {
		op := docWithFriends.FindOperation(&FindOperationOptions{"", "/person/{id}", "PATCH", nil})
		assert.NotNil(op)

		res := op.GetResponse()

		expected := &OpenAPIResponse{}

		AssertJSONEqual(t, expected, res)
	})

	t.Run("GetResponse-non-predefined", func(t *testing.T) {
		op := docWithFriends.FindOperation(&FindOperationOptions{"", "/person/{id}/find", "GET", nil})
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
			ContentType: "application/json",
			StatusCode:  200,
		}

		AssertJSONEqual(t, expected, res)
	})

	t.Run("WithParseConfig", func(t *testing.T) {
		op := docWithFriends.FindOperation(&FindOperationOptions{"", "/person/{id}/find", "GET", nil})
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
			ContentType: "application/json",
			StatusCode:  200,
		}

		AssertJSONEqual(t, expected, res)
	})

	t.Run("parseParameter", func(t *testing.T) {
		op := docWithFriends.FindOperation(&FindOperationOptions{"", "/person/{id}/find", "GET", nil}).(*LibV2Operation)
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
			In:         ParameterInQuery,
			Type:       TypeInteger,
			Format:     "int64",
			Required:   &required,
			Default:    10,
			Minimum:    &minimum,
			Maximum:    &maximum,
			Enum:       []any{10, 20, 30},
			MinItems:   &minItems,
			MaxItems:   &maxItems,
			MinLength:  &minLength,
			MaxLength:  &maxLength,
			Pattern:    "^[a-zA-Z0-9]+$",
			MultipleOf: &multipleOf,
			Items: &v2high.Items{
				Type: TypeString,
			},
		}

		expected := &Schema{
			Type: TypeInteger,
			Items: &Schema{
				Type: TypeString,
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
		op := docWithFriends.FindOperation(&FindOperationOptions{"", "/person/{id}/find", "GET", nil}).(*LibV2Operation)
		assert.NotNil(op)
		libDoc, ok := docWithFriends.(*LibV2Document)
		assert.True(ok)

		libSchema := libDoc.DocumentModel.Model.Definitions.Definitions["State"].Schema()

		libParam := v2high.Parameter{
			Name:   "limit",
			In:     ParameterInQuery,
			Type:   TypeObject,
			Schema: base.CreateSchemaProxy(libSchema),
		}

		res := op.parseParameter(&libParam)
		expected := &Schema{
			Type: TypeObject,
			Properties: map[string]*Schema{
				"name": {
					Type: TypeString,
				},
				"abbr": {
					Type: TypeString,
				},
			},
		}
		AssertJSONEqual(t, expected, res)
	})
}
