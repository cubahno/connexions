package connexions

import (
	assert2 "github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"path/filepath"
	"testing"
)

func CreateLibDocumentFromFile(t *testing.T, filePath string) Document {
	doc, err := NewLibOpenAPIDocumentFromFile(filePath)
	if err != nil {
		t.Errorf("Error loading document: %v", err)
		t.FailNow()
	}
	return doc
}

func GetLibYamlExpectations(t *testing.T, schema *Schema, expected string) (any, any, []byte) {
	assert := assert2.New(t)
	renderedYaml, _ := yaml.Marshal(schema)
	var resYaml any
	err := yaml.Unmarshal(renderedYaml, &resYaml)
	assert.Nil(err)

	var expectedYaml any
	err = yaml.Unmarshal([]byte(expected), &expectedYaml)
	assert.Nil(err)

	return expectedYaml, resYaml, renderedYaml
}

func TestNewLibOpenAPIDocumentFromFile(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	t.Run("file-not-found", func(t *testing.T) {
		res, err := NewLibOpenAPIDocumentFromFile("file-not-found.yml")
		assert.Nil(res)
		assert.NotNil(err)
	})


	t.Run("invalid-yaml", func(t *testing.T) {
		res, err := NewLibOpenAPIDocumentFromFile(filepath.Join("test_fixtures", "document-invalid.yml"))
		assert.Nil(res)
		assert.NotNil(err)
	})
}

func TestLibV3Document(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()
	doc := CreateLibDocumentFromFile(t, filepath.Join("test_fixtures", "document-petstore.yml"))

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
		op := doc.FindOperation(&FindOperationOptions{"", "/pets", "GET", nil})
		assert.NotNil(op)
		libOp, ok := op.(*LibV3Operation)
		assert.True(ok)

		assert.Equal(2, len(op.GetParameters()))
		assert.Equal("findPets", libOp.OperationId)
	})

	t.Run("FindOperation-res-not-found", func(t *testing.T) {
		op := doc.FindOperation(&FindOperationOptions{"", "/pets2", "GET", nil})
		assert.Nil(op)
	})

	t.Run("FindOperation-method-not-found", func(t *testing.T) {
		op := doc.FindOperation(&FindOperationOptions{"", "/pets", "PATCH", nil})
		assert.Nil(op)
	})
}

func TestLibV3Operation(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()
	doc := CreateLibDocumentFromFile(t, filepath.Join("test_fixtures", "document-petstore.yml"))
	docWithFriends := CreateLibDocumentFromFile(t, filepath.Join("test_fixtures", "person-with-friends.yml"))

	t.Run("FindOperation-with-no-options", func(t *testing.T) {
		op := doc.FindOperation(nil)
		assert.Nil(op)
	})

	t.Run("GetParameters", func(t *testing.T) {
		op := doc.FindOperation(&FindOperationOptions{"", "/pets", "GET", nil})
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

		AssertJSONEqual(t, expected, params)
	})

	t.Run("GetRequestBody", func(t *testing.T) {
		op := doc.FindOperation(&FindOperationOptions{"", "/pets", "POST", nil})
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
		body, contentType := op.GetRequestBody()

		assert.Nil(body)
		assert.Equal("", contentType)
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
		op := doc.FindOperation(&FindOperationOptions{"", "/pets", "GET", nil})
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

	t.Run("GetResponse-first-defined-non-default", func(t *testing.T) {
		op := docWithFriends.FindOperation(&FindOperationOptions{"", "/person/{id}", "GET", nil})
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
						Type: TypeInteger,
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
			ContentType: "application/xml",
			StatusCode:  200,
		}

		AssertJSONEqual(t, expected, res)
	})

	t.Run("getContent-empty", func(t *testing.T) {
		op := docWithFriends.FindOperation(&FindOperationOptions{"", "/person/{id}/find", "GET", nil})
		assert.NotNil(op)
		opLib := op.(*LibV3Operation)

		res, contentType := opLib.getContent(nil)

		assert.Nil(res)
		assert.Equal("", contentType)
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
			ContentType: "application/xml",
			StatusCode:  200,
		}

		AssertJSONEqual(t, expected, res)
	})
}

func TestNewSchemaFromLibOpenAPI(t *testing.T) {
	t.Parallel()
	assert := assert2.New(t)
	doc := CreateLibDocumentFromFile(t, filepath.Join("test_fixtures", "person-with-friends.yml")).(*LibV3Document)

	t.Run("no-schema", func(t *testing.T) {
		res := NewSchemaFromLibOpenAPI(nil, nil)
		assert.Nil(res)
	})

	t.Run("files", func(t *testing.T) {
		circDoc := CreateLibDocumentFromFile(t, filepath.Join("test_fixtures", "document-files-circular.yml")).(*LibV3Document)
		libSchema := circDoc.Model.Paths.PathItems["/files"].Get.Responses.Codes["200"].Content["application/json"].Schema.Schema()
		res := NewSchemaFromLibOpenAPI(libSchema, nil)

		assert.NotNil(res)
	})

	t.Run("SimpleArray", func(t *testing.T) {
		libSchema := doc.Model.Components.Schemas["SimpleArray"].Schema()
		assert.NotNil(libSchema)

		res := NewSchemaFromLibOpenAPI(libSchema, nil)
		expected := `
type: array
items:
  type: string
`
		expectedYaml, actualYaml, rendered := GetLibYamlExpectations(t, res, expected)
		assert.Greater(len(rendered), 0)
		assert.Equal(expectedYaml, actualYaml)
	})

	t.Run("SimpleArrayWithRef", func(t *testing.T) {
		libSchema := doc.Model.Components.Schemas["SimpleArrayWithRef"].Schema()
		assert.NotNil(libSchema)

		res := NewSchemaFromLibOpenAPI(libSchema, nil)
		expected := `
type: array
items:
  type: object
  properties:
    name:
      type: string
`
		expectedYaml, actualYaml, rendered := GetLibYamlExpectations(t, res, expected)
		assert.Greater(len(rendered), 0)
		assert.Equal(expectedYaml, actualYaml)
	})

	t.Run("SimpleObjectCircular", func(t *testing.T) {
		libSchema := doc.Model.Components.Schemas["SimpleObjectCircular"].Schema()
		assert.NotNil(libSchema)

		res := NewSchemaFromLibOpenAPI(libSchema, nil)
		expected := `
type: object
properties:
    relatives:
        type: array
        items:
            type: object
            properties:
                relatives:
                    type: array
                user:
                    type: object
                    properties:
                        name:
                            type: string
    user:
        type: object
        properties:
            name:
                type: string
`
		expectedYaml, actualYaml, rendered := GetLibYamlExpectations(t, res, expected)
		assert.Greater(len(rendered), 0)
		assert.Equal(expectedYaml, actualYaml)
	})

	t.Run("SimpleObjectCircularNested", func(t *testing.T) {
		libSchema := doc.Model.Components.Schemas["SimpleObjectCircularNested"].Schema()
		assert.NotNil(libSchema)

		res := NewSchemaFromLibOpenAPI(libSchema, nil)
		expected := `
type: object
properties:
    address:
        type: object
        properties:
            neighbors:
                type: array
                items:
                    type: object
                    properties:
                        address:
                            type: object
                            properties:
                                neighbors:
                                    type: array
                                supervisor: null
                        user:
                            type: object
                            properties:
                                name:
                                    type: string
            supervisor:
                type: object
                properties:
                    address:
                        type: object
                        properties:
                            neighbors:
                                type: array
                            supervisor: null
                    user:
                        type: object
                        properties:
                            name:
                                type: string
    user:
        type: object
        properties:
            name:
                type: string

`
		expectedYaml, actualYaml, rendered := GetLibYamlExpectations(t, res, expected)
		assert.Greater(len(rendered), 0)
		assert.Equal(expectedYaml, actualYaml)
	})

	t.Run("ObjectsWithReferencesAndArrays", func(t *testing.T) {
		libSchema := doc.Model.Components.Schemas["ObjectsWithReferencesAndArrays"].Schema()
		assert.NotNil(libSchema)

		res := NewSchemaFromLibOpenAPI(libSchema, nil)
		expected := `
type: object
properties:
    relatives:
        type: array
        items:
            type: object
            properties:
                name:
                    type: string
    user:
        type: object
        properties:
            friends:
                type: array
                items:
                    type: object
                    properties:
                        name:
                            type: string
            name:
                type: string
`
		expectedYaml, actualYaml, rendered := GetLibYamlExpectations(t, res, expected)
		assert.Greater(len(rendered), 0)
		assert.Equal(expectedYaml, actualYaml)
	})

	t.Run("AddressWithAllOf", func(t *testing.T) {
		libSchema := doc.Model.Components.Schemas["AddressWithAllOf"].Schema()
		assert.NotNil(libSchema)

		res := NewSchemaFromLibOpenAPI(libSchema, nil)
		expected := `
type: object
properties:
  name:
    type: string
  address:
    type: object
    properties:
      name:
        type: string
      abbr:
        type: string
`
		expectedYaml, actualYaml, rendered := GetLibYamlExpectations(t, res, expected)
		assert.Greater(len(rendered), 0)
		assert.Equal(expectedYaml, actualYaml)
	})

	t.Run("ObjectWithAllOfPersonAndEmployee", func(t *testing.T) {
		libSchema := doc.Model.Components.Schemas["ObjectWithAllOfPersonAndEmployee"].Schema()
		assert.NotNil(libSchema)

		res := NewSchemaFromLibOpenAPI(libSchema, nil)
		expected := `
type: object
properties:
  user:
    type: object
    properties:
      name:
        type: string
  employeeId:
    type: integer
`
		expectedYaml, actualYaml, rendered := GetLibYamlExpectations(t, res, expected)
		assert.Greater(len(rendered), 0)
		assert.Equal(expectedYaml, actualYaml)
	})

	t.Run("AddressWithAnyOfObject", func(t *testing.T) {
		libSchema := doc.Model.Components.Schemas["AddressWithAnyOfObject"].Schema()
		assert.NotNil(libSchema)

		res := NewSchemaFromLibOpenAPI(libSchema, nil)
		expected := `
type: object
properties:
  name:
    type: string
  abbr:
    type: string
`
		expectedYaml, actualYaml, rendered := GetLibYamlExpectations(t, res, expected)
		assert.Greater(len(rendered), 0)
		assert.Equal(expectedYaml, actualYaml)
	})

	t.Run("AddressWithAnyOfArray", func(t *testing.T) {
		libSchema := doc.Model.Components.Schemas["AddressWithAnyOfArray"].Schema()
		assert.NotNil(libSchema)

		res := NewSchemaFromLibOpenAPI(libSchema, nil)
		expected := `
type: array
items:
  type: object
  properties:
    name:
      type: string
    abbr:
      type: string
`
		expectedYaml, actualYaml, rendered := GetLibYamlExpectations(t, res, expected)
		assert.Greater(len(rendered), 0)
		assert.Equal(expectedYaml, actualYaml)
	})

	t.Run("AddressWithAnyOfArrayWithoutArrayType", func(t *testing.T) {
		libSchema := doc.Model.Components.Schemas["AddressWithAnyOfArrayWithoutArrayType"].Schema()
		assert.NotNil(libSchema)

		res := NewSchemaFromLibOpenAPI(libSchema, nil)
		expected := `
type: array
items:
  type: object
  properties:
    name:
      type: string
    abbr:
      type: string
`
		expectedYaml, actualYaml, rendered := GetLibYamlExpectations(t, res, expected)
		assert.Greater(len(rendered), 0)
		assert.Equal(expectedYaml, actualYaml)
	})

	t.Run("ArrayOfPersonAndEmployeeWithFriends", func(t *testing.T) {
		libSchema := doc.Model.Components.Schemas["ArrayOfPersonAndEmployeeWithFriends"].Schema()
		assert.NotNil(libSchema)

		res := NewSchemaFromLibOpenAPI(libSchema, nil)
		expected := `
type: array
items:
    type: object
    properties:
        name:
            type: string
        age:
            type: integer
        employeeId:
            type: integer
        address:
            type: object
            properties:
                state:
                    type: string
                city:
                    type: string
        friends:
            type: array
            items:
                type: object
                required: [name]
                properties: 
                    name:   
                        type: string
                    age:    
                        type: integer
                    address:
                        type: object
                        properties:
                            state:
                                type: string
                            city:
                                type: string
    required: [name]
`
		expectedYaml, actualYaml, rendered := GetLibYamlExpectations(t, res, expected)
		assert.Greater(len(rendered), 0)
		assert.Equal(expectedYaml, actualYaml)
	})

	t.Run("PersonFeatures", func(t *testing.T) {
		libSchema := doc.Model.Components.Schemas["PersonFeatures"].Schema()
		assert.NotNil(libSchema)

		res := NewSchemaFromLibOpenAPI(libSchema, nil)
		expected := `
type: object
properties:
    neighbors:
        type: object
        properties:
            houseLeft:
                type: object
                properties:
                    name:
                        type: string
    severity:
        type: string
    address:
        type: object
        properties:
            abbr:
                type: string
            name:
                type: string
    previousAddresses:
        type: array
        items:
            type: object
            properties:
                state:
                    type: string
                city:
                    type: string
`
		expectedYaml, actualYaml, rendered := GetLibYamlExpectations(t, res, expected)
		assert.Greater(len(rendered), 0)
		assert.Equal(expectedYaml, actualYaml)
	})

	t.Run("stripe", func(t *testing.T) {
		doc := CreateLibDocumentFromFile(t, filepath.Join("test_fixtures", "document-psp.yml")).(*LibV3Document)
		libSchema := doc.Model.Components.Schemas["charge"].Schema()
		assert.NotNil(libSchema)

		res := NewSchemaFromLibOpenAPI(libSchema, nil)
		assert.NotNil(res)
	})

	t.Run("WithParseConfig-1-level", func(t *testing.T) {
		libSchema := doc.Model.Components.Schemas["Person"].Schema()
		assert.NotNil(libSchema)

		cfg := &ParseConfig{
			MaxLevels: 1,
		}
		res := NewSchemaFromLibOpenAPI(libSchema, cfg)
		expected := `
type: object
required: [name]
properties:
    address:
        type: object
        properties:
            city: null
            state: null
    age:
        type: integer
    name:
        type: string
`
		expectedYaml, actualYaml, rendered := GetLibYamlExpectations(t, res, expected)
		assert.Greater(len(rendered), 0)
		assert.Equal(expectedYaml, actualYaml)
	})

	t.Run("WithParseConfig-only-required", func(t *testing.T) {
		libSchema := doc.Model.Components.Schemas["Person"].Schema()
		assert.NotNil(libSchema)

		cfg := &ParseConfig{
			OnlyRequired: true,
		}
		res := NewSchemaFromLibOpenAPI(libSchema, cfg)
		expected := `
type: object
properties:
    name:
        type: string
required: [name]
`
		expectedYaml, actualYaml, rendered := GetLibYamlExpectations(t, res, expected)
		assert.Greater(len(rendered), 0)
		assert.Equal(expectedYaml, actualYaml)
	})
}

func TestMergeLibOpenAPISubSchemas(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()
	doc := CreateLibDocumentFromFile(t, filepath.Join("test_fixtures", "person-with-friends.yml")).(*LibV3Document)

	t.Run("AddressWithAllOf", func(t *testing.T) {
		libSchema := doc.Model.Components.Schemas["AddressWithAllOf"].Schema()
		assert.NotNil(libSchema)

		res, ref := mergeLibOpenAPISubSchemas(libSchema)
		assert.NotNil(res)
		assert.Equal("#/components/schemas/PersonEmbeddable", ref)
	})

	t.Run("AddressWithAnyOfObject", func(t *testing.T) {
		libSchema := doc.Model.Components.Schemas["AddressWithAnyOfObject"].Schema()
		assert.NotNil(libSchema)

		res, ref := mergeLibOpenAPISubSchemas(libSchema)
		assert.NotNil(res)
		assert.Equal("#/components/schemas/State", ref)
	})

	t.Run("AddressWithAnyOfArray", func(t *testing.T) {
		libSchema := doc.Model.Components.Schemas["AddressWithAnyOfArray"].Schema()
		assert.NotNil(libSchema)

		res, ref := mergeLibOpenAPISubSchemas(libSchema)
		assert.NotNil(res)
		assert.Equal("#/components/schemas/State", ref)
	})

	t.Run("ObjectWithAllOfPersonAndEmployee", func(t *testing.T) {
		libSchema := doc.Model.Components.Schemas["ObjectWithAllOfPersonAndEmployee"].Schema()
		assert.NotNil(libSchema)

		res, ref := mergeLibOpenAPISubSchemas(libSchema)
		assert.NotNil(res)
		assert.Equal("#/components/schemas/Employee", ref)
	})
}

func TestPicklLibOpenAPISchemaProxy(t *testing.T) {

}
