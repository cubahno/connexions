//go:build !integration

package connexions

import (
	"github.com/pb33f/libopenapi/datamodel/high/base"
	assert2 "github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"path/filepath"
	"testing"
)

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

	t.Run("circular-swagger", func(t *testing.T) {
		res, err := NewLibOpenAPIDocumentFromFile(filepath.Join("test_fixtures", "document-circular-with-references-v2.yml"))
		assert.NotNil(res)
		assert.NoError(err)
	})

	t.Run("error-swagger", func(t *testing.T) {
		res, err := NewLibOpenAPIDocumentFromFile(filepath.Join("test_fixtures", "document-invalid-v2.yml"))
		assert.Nil(res)
		assert.Error(err)
	})
}

func TestNewSchemaFromLibOpenAPI(t *testing.T) {
	t.Parallel()
	assert := assert2.New(t)
	libDoc, err := NewLibOpenAPIDocumentFromFile(filepath.Join("test_fixtures", "document-person-with-friends.yml"))
	assert.Nil(err)
	doc := libDoc.(*LibV3Document)

	t.Run("no-schema", func(t *testing.T) {
		res := NewSchemaFromLibOpenAPI(nil, nil)
		assert.Nil(res)
	})

	t.Run("files", func(t *testing.T) {
		libDoc1, err := NewLibOpenAPIDocumentFromFile(filepath.Join("test_fixtures", "document-files-circular.yml"))
		assert.Nil(err)
		circDoc := libDoc1.(*LibV3Document)
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
                relatives: null
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
                                neighbors: null
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
                            neighbors: null
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

	t.Run("StateWithoutAbbr", func(t *testing.T) {
		libSchema := doc.Model.Components.Schemas["StateWithoutAbbr"].Schema()
		assert.NotNil(libSchema)

		res := NewSchemaFromLibOpenAPI(libSchema, nil)
		expected := `
type: object
properties:
    abbr:
        type: string
    name:
        type: string
not:
    type: object
    properties:
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
		libDoc, err := NewLibOpenAPIDocumentFromFile(filepath.Join("test_fixtures", "document-psp.yml"))
		assert.Nil(err)
		doc := libDoc.(*LibV3Document)
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
	libDoc, err := NewLibOpenAPIDocumentFromFile(filepath.Join("test_fixtures", "document-person-with-friends.yml"))
	assert.Nil(err)
	doc := libDoc.(*LibV3Document)

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

	t.Run("StateWithoutAbbr", func(t *testing.T) {
		libSchema := doc.Model.Components.Schemas["StateWithoutAbbr"].Schema()
		assert.NotNil(libSchema)

		res, ref := mergeLibOpenAPISubSchemas(libSchema)
		assert.NotNil(res)
		assert.Equal("#/components/schemas/State", ref)
	})

	t.Run("ImpliedTypeResolved", func(t *testing.T) {
		libSchema := doc.Model.Components.Schemas["ImpliedType"].Schema()
		assert.NotNil(libSchema)

		res, _ := mergeLibOpenAPISubSchemas(libSchema)
		assert.NotNil(res)
		assert.Equal(TypeObject, res.Type[0])
	})

	t.Run("EmptyPolymorphic", func(t *testing.T) {
		libSchema := doc.Model.Components.Schemas["EmptyPolymorphic"].Schema()
		assert.NotNil(libSchema)

		res, _ := mergeLibOpenAPISubSchemas(libSchema)
		assert.NotNil(res)
		assert.NotNil(res.Type)
		assert.Equal(TypeObject, res.Type[0])
	})

	t.Run("RecursiveCall", func(t *testing.T) {
		libSchema := doc.Model.Components.Schemas["Connexions"].Schema()
		assert.NotNil(libSchema)

		expectedProps := []string{
			"name",
			"age",
			"address",
			"employeeId",
			"neighbors",
			"severity",
			"previousAddresses",
		}

		res, _ := mergeLibOpenAPISubSchemas(libSchema)
		assert.NotNil(res)
		assert.NotNil(res.Type)
		assert.Equal(TypeObject, res.Type[0])

		var props []string
		for k := range res.Properties {
			props = append(props, k)
		}
		assert.ElementsMatch(expectedProps, props)
	})
}

func TestPickLibOpenAPISchemaProxy(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()
	libDoc, err := NewLibOpenAPIDocumentFromFile(filepath.Join("test_fixtures", "document-person-with-friends.yml"))
	assert.Nil(err)
	doc := libDoc.(*LibV3Document)

	t.Run("skips-empty-returns-ref", func(t *testing.T) {
		libSchema := doc.Model.Components.Schemas["StateWithoutAbbr"]
		assert.NotNil(libSchema)

		schemaProxies := []*base.SchemaProxy{
			nil,
			libSchema,
			base.CreateSchemaProxyRef("#/components/schemas/State"),
		}

		res := pickLibOpenAPISchemaProxy(schemaProxies)
		assert.NotNil(res)
		assert.Equal("#/components/schemas/State", res.GetReference())
	})

	t.Run("fst-not-empty-without-ref", func(t *testing.T) {
		libSchema := doc.Model.Components.Schemas["StateWithoutAbbr"]
		assert.NotNil(libSchema)

		schemaProxies := []*base.SchemaProxy{
			nil,
			libSchema,
			nil,
		}

		res := pickLibOpenAPISchemaProxy(schemaProxies)
		assert.NotNil(res)
	})
}

func TestGetLibAdditionalProperties(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	t.Run("unknown-case", func(t *testing.T) {
		res := getLibAdditionalProperties("schema")
		assert.Nil(res)
	})

	t.Run("nil-case", func(t *testing.T) {
		res := getLibAdditionalProperties(nil)
		assert.Nil(res)
	})

	t.Run("false-case", func(t *testing.T) {
		res := getLibAdditionalProperties(false)
		assert.Nil(res)
	})

	t.Run("true-case", func(t *testing.T) {
		res := getLibAdditionalProperties(true)
		expected := &base.Schema{Type: []string{TypeString}}
		assert.Equal(expected, res)
	})

	t.Run("inlined-case", func(t *testing.T) {
		schema := `
type: object
properties:
  name:
    type: string
  age:
    type: integer
`
		source := CreateLibSchemaFromString(t, schema)

		res := getLibAdditionalProperties(source)

		assert.NotNil(res)
		assert.Equal(TypeObject, res.Type[0])
		assert.Equal(TypeString, res.Properties["name"].Schema().Type[0])
		assert.Equal(TypeInteger, res.Properties["age"].Schema().Type[0])
		assert.Nil(res.AdditionalProperties)
	})
}
