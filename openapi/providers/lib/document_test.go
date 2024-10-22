//go:build !integration

package lib

import (
	"github.com/cubahno/connexions/config"
	"github.com/cubahno/connexions/openapi"
	"github.com/cubahno/connexions/openapi/providers"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	assert2 "github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"path/filepath"
	"testing"
)

func GetLibYamlExpectations(t *testing.T, schema *openapi.Schema, expected string) (any, any, []byte) {
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

func TestNewDocumentFromFile(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	testData := filepath.Join("..", "..", "..", "testdata")

	t.Run("file-not-found", func(t *testing.T) {
		res, err := NewDocumentFromFile("file-not-found.yml")
		assert.Nil(res)
		assert.NotNil(err)
	})

	t.Run("invalid-yaml", func(t *testing.T) {
		res, err := NewDocumentFromFile(filepath.Join(testData, "document-invalid.yml"))
		assert.Nil(res)
		assert.NotNil(err)
	})

	t.Run("circular-swagger", func(t *testing.T) {
		res, err := NewDocumentFromFile(filepath.Join(testData, "document-circular-with-references-v2.yml"))
		assert.NotNil(res)
		assert.NoError(err)
	})

	t.Run("error-swagger", func(t *testing.T) {
		res, err := NewDocumentFromFile(filepath.Join(testData, "document-invalid-v2.yml"))
		assert.Nil(res)
		assert.Error(err)
	})
}

func TestDocument(t *testing.T) {
	tc := &providers.DocumentTestCase{
		DocFactory: NewDocumentFromFile,
	}
	tc.Run(t)
}

func TestOperation(t *testing.T) {
	tc := &providers.OperationTestCase{
		DocFactory: NewDocumentFromFile,
	}
	tc.Run(t)
}

func TestNewSchema(t *testing.T) {
	t.Parallel()
	assert := assert2.New(t)

	testData := filepath.Join("..", "..", "..", "testdata")
	libDoc, err := NewDocumentFromFile(filepath.Join(testData, "document-person-with-friends.yml"))
	assert.Nil(err)
	doc := libDoc.(*V3Document)

	t.Run("NewSchemaSuite", func(t *testing.T) {
		getSchema := func(t *testing.T, fileName, componentID string, parseConfig *config.ParseConfig) *openapi.Schema {
			t.Helper()
			d, err := NewDocumentFromFile(filepath.Join(testData, fileName))
			assert.Nil(err)
			v3Doc := d.(*V3Document)
			libSchemaProxy, _ := v3Doc.Model.Components.Schemas.Get(componentID)
			libSchema := libSchemaProxy.Schema()
			assert.NotNil(libSchema)

			return NewSchema(libSchema, parseConfig)
		}
		tc := &providers.NewSchemaTestSuite{
			SchemaFactory: getSchema,
		}

		tc.Run(t)
	})

	t.Run("no-schema", func(t *testing.T) {
		res := NewSchema(nil, nil)
		assert.Nil(res)
	})

	t.Run("files", func(t *testing.T) {
		libDoc1, err := NewDocumentFromFile(filepath.Join(testData, "document-files-circular.yml"))
		assert.Nil(err)
		circDoc := libDoc1.(*V3Document)
		files, _ := circDoc.Model.Paths.PathItems.Get("/files")
		code, _ := files.Get.Responses.Codes.Get("200")
		js, _ := code.Content.Get("application/json")
		libSchema := js.Schema.Schema()
		res := NewSchema(libSchema, nil)

		assert.NotNil(res)
	})

	t.Run("SimpleArray", func(t *testing.T) {
		libSchema := doc.Model.Components.Schemas.GetOrZero("SimpleArray").Schema()
		assert.NotNil(libSchema)

		res := NewSchema(libSchema, nil)
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
		libSchema := doc.Model.Components.Schemas.GetOrZero("SimpleArrayWithRef").Schema()
		assert.NotNil(libSchema)

		res := NewSchema(libSchema, nil)
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
		libSchema := doc.Model.Components.Schemas.GetOrZero("SimpleObjectCircular").Schema()
		assert.NotNil(libSchema)

		res := NewSchema(libSchema, nil)
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
		libSchema := doc.Model.Components.Schemas.GetOrZero("SimpleObjectCircularNested").Schema()
		assert.NotNil(libSchema)

		res := NewSchema(libSchema, nil)
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
		libSchema := doc.Model.Components.Schemas.GetOrZero("ObjectsWithReferencesAndArrays").Schema()
		assert.NotNil(libSchema)

		res := NewSchema(libSchema, nil)
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
		libSchema := doc.Model.Components.Schemas.GetOrZero("AddressWithAllOf").Schema()
		assert.NotNil(libSchema)

		res := NewSchema(libSchema, nil)
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
		libSchema := doc.Model.Components.Schemas.GetOrZero("ObjectWithAllOfPersonAndEmployee").Schema()
		assert.NotNil(libSchema)

		res := NewSchema(libSchema, nil)
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
		libSchema := doc.Model.Components.Schemas.GetOrZero("AddressWithAnyOfObject").Schema()
		assert.NotNil(libSchema)

		res := NewSchema(libSchema, nil)
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
		libSchema := doc.Model.Components.Schemas.GetOrZero("AddressWithAnyOfArray").Schema()
		assert.NotNil(libSchema)

		res := NewSchema(libSchema, nil)
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
		libSchema := doc.Model.Components.Schemas.GetOrZero("StateWithoutAbbr").Schema()
		assert.NotNil(libSchema)

		res := NewSchema(libSchema, nil)
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
		libSchema := doc.Model.Components.Schemas.GetOrZero("AddressWithAnyOfArrayWithoutArrayType").Schema()
		assert.NotNil(libSchema)

		res := NewSchema(libSchema, nil)
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
		libSchema := doc.Model.Components.Schemas.GetOrZero("ArrayOfPersonAndEmployeeWithFriends").Schema()
		assert.NotNil(libSchema)

		res := NewSchema(libSchema, nil)
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
		libSchema := doc.Model.Components.Schemas.GetOrZero("PersonFeatures").Schema()
		assert.NotNil(libSchema)

		res := NewSchema(libSchema, nil)
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
		libDoc, err := NewDocumentFromFile(filepath.Join(testData, "document-psp.yml"))
		assert.Nil(err)
		doc := libDoc.(*V3Document)
		libSchema := doc.Model.Components.Schemas.GetOrZero("charge").Schema()
		assert.NotNil(libSchema)

		res := NewSchema(libSchema, nil)
		assert.NotNil(res)
	})

	t.Run("WithParseConfig-1-level", func(t *testing.T) {
		libSchema := doc.Model.Components.Schemas.GetOrZero("Person").Schema()
		assert.NotNil(libSchema)

		cfg := &config.ParseConfig{
			MaxLevels: 1,
		}
		res := NewSchema(libSchema, cfg)
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
		libSchema := doc.Model.Components.Schemas.GetOrZero("Person").Schema()
		assert.NotNil(libSchema)

		cfg := &config.ParseConfig{
			OnlyRequired: true,
		}
		res := NewSchema(libSchema, cfg)
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

	t.Run("min-amount-of-additional-props", func(t *testing.T) {
		yml := `
type: object
minProperties: 5
properties:
  name:
    type: string
additionalProperties:
  type: string
`
		schema := CreateLibSchemaFromString(t, yml)
		assert.NotNil(schema)

		res := NewSchema(schema.Schema(), nil)
		assert.NotNil(res)
		assert.Equal(6, len(res.Properties))
	})
}

func TestMergeLibOpenAPISubSchemas(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()
	testData := filepath.Join("..", "..", "..", "testdata")

	libDoc, err := NewDocumentFromFile(filepath.Join(testData, "document-person-with-friends.yml"))
	assert.Nil(err)
	doc := libDoc.(*V3Document)

	t.Run("AddressWithAllOf", func(t *testing.T) {
		libSchema := doc.Model.Components.Schemas.GetOrZero("AddressWithAllOf").Schema()
		assert.NotNil(libSchema)

		res, ref := mergeSubSchemas(libSchema)
		assert.NotNil(res)
		assert.Equal("#/components/schemas/PersonEmbeddable", ref)
	})

	t.Run("AddressWithAnyOfObject", func(t *testing.T) {
		libSchema := doc.Model.Components.Schemas.GetOrZero("AddressWithAnyOfObject").Schema()
		assert.NotNil(libSchema)

		res, ref := mergeSubSchemas(libSchema)
		assert.NotNil(res)
		assert.Equal("#/components/schemas/State", ref)
	})

	t.Run("AddressWithAnyOfArray", func(t *testing.T) {
		libSchema := doc.Model.Components.Schemas.GetOrZero("AddressWithAnyOfArray").Schema()
		assert.NotNil(libSchema)

		res, ref := mergeSubSchemas(libSchema)
		assert.NotNil(res)
		assert.Equal("#/components/schemas/State", ref)
	})

	t.Run("ObjectWithAllOfPersonAndEmployee", func(t *testing.T) {
		libSchema := doc.Model.Components.Schemas.GetOrZero("ObjectWithAllOfPersonAndEmployee").Schema()
		assert.NotNil(libSchema)

		res, ref := mergeSubSchemas(libSchema)
		assert.NotNil(res)
		assert.Equal("#/components/schemas/Employee", ref)
	})

	t.Run("StateWithoutAbbr", func(t *testing.T) {
		libSchema := doc.Model.Components.Schemas.GetOrZero("StateWithoutAbbr").Schema()
		assert.NotNil(libSchema)

		res, ref := mergeSubSchemas(libSchema)
		assert.NotNil(res)
		assert.Equal("#/components/schemas/State", ref)
	})

	t.Run("ImpliedTypeResolved", func(t *testing.T) {
		libSchema := doc.Model.Components.Schemas.GetOrZero("ImpliedType").Schema()
		assert.NotNil(libSchema)

		res, _ := mergeSubSchemas(libSchema)
		assert.NotNil(res)
		assert.Equal(openapi.TypeObject, res.Type[0])
	})

	t.Run("EmptyPolymorphic", func(t *testing.T) {
		libSchema := doc.Model.Components.Schemas.GetOrZero("EmptyPolymorphic").Schema()
		assert.NotNil(libSchema)

		res, _ := mergeSubSchemas(libSchema)
		assert.NotNil(res)
		assert.NotNil(res.Type)
		assert.Equal(openapi.TypeObject, res.Type[0])
	})

	t.Run("RecursiveCall", func(t *testing.T) {
		libSchema := doc.Model.Components.Schemas.GetOrZero("Connexions").Schema()
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

		res, _ := mergeSubSchemas(libSchema)
		assert.NotNil(res)
		assert.NotNil(res.Type)
		assert.Equal(openapi.TypeObject, res.Type[0])

		var props []string
		for k := range res.Properties.KeysFromOldest() {
			props = append(props, k)
		}
		assert.ElementsMatch(expectedProps, props)
	})

	t.Run("inferred-from-enum", func(t *testing.T) {
		t.Skip("Fix this test with correct types")
		target := &base.Schema{
			Enum: nil, // []any{1, 2, 3}
		}

		schema, _ := mergeSubSchemas(target)
		assert.Equal([]string{openapi.TypeInteger}, schema.Type)
	})
}

func TestPickLibOpenAPISchemaProxy(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()
	testData := filepath.Join("..", "..", "..", "testdata")
	libDoc, err := NewDocumentFromFile(filepath.Join(testData, "document-person-with-friends.yml"))
	assert.Nil(err)
	doc := libDoc.(*V3Document)

	t.Run("skips-empty-returns-ref", func(t *testing.T) {
		libSchema := doc.Model.Components.Schemas.GetOrZero("StateWithoutAbbr")
		assert.NotNil(libSchema)

		schemaProxies := []*base.SchemaProxy{
			nil,
			libSchema,
			base.CreateSchemaProxyRef("#/components/schemas/State"),
		}

		res := pickSchemaProxy(schemaProxies)
		assert.NotNil(res)
		assert.Equal("#/components/schemas/State", res.GetReference())
	})

	t.Run("fst-not-empty-without-ref", func(t *testing.T) {
		libSchema := doc.Model.Components.Schemas.GetOrZero("StateWithoutAbbr")
		assert.NotNil(libSchema)

		schemaProxies := []*base.SchemaProxy{
			nil,
			libSchema,
			nil,
		}

		res := pickSchemaProxy(schemaProxies)
		assert.NotNil(res)
	})
}

func TestGetLibAdditionalProperties(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	t.Run("unknown-case", func(t *testing.T) {
		res := getAdditionalProperties("schema")
		assert.Nil(res)
	})

	t.Run("nil-case", func(t *testing.T) {
		res := getAdditionalProperties(nil)
		assert.Nil(res)
	})

	t.Run("false-case", func(t *testing.T) {
		res := getAdditionalProperties(false)
		assert.Nil(res)
	})

	t.Run("true-case", func(t *testing.T) {
		res := getAdditionalProperties(true)
		expected := &base.Schema{Type: []string{openapi.TypeString}}
		assert.Equal(expected, res)
	})

	t.Run("inlined-case", func(t *testing.T) {
		schema := `
type: object
minProperties: 2
properties:
  name:
    type: string
  age:
    type: integer
  city:
    type: string
`
		source := CreateLibSchemaFromString(t, schema)

		res := getAdditionalProperties(source)

		assert.NotNil(res)
		assert.Equal(openapi.TypeObject, res.Type[0])
		assert.Equal(openapi.TypeString, res.Properties.GetOrZero("name").Schema().Type[0])
		assert.Equal(openapi.TypeInteger, res.Properties.GetOrZero("age").Schema().Type[0])
		assert.Nil(res.AdditionalProperties)
	})
}
