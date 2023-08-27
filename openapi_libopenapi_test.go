package connexions

import (
    "github.com/pb33f/libopenapi/datamodel/high/base"
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

func AssertLibYamlEqual(t *testing.T, schema *base.Schema, expected string) {
    assert := assert2.New(t)
    renderedYaml, _ := schema.RenderInline()
    var resYaml any
    _ = yaml.Unmarshal(renderedYaml, &resYaml)

    var expectedYaml any
    err := yaml.Unmarshal([]byte(expected), &expectedYaml)
    assert.Nil(err)

    assert.Equal(expectedYaml, resYaml)
}

func GetLibYamlExpectations(t *testing.T, schema *base.Schema, expected string) (any, any, []byte) {
    assert := assert2.New(t)
    renderedYaml, _ := schema.RenderInline()
    var resYaml any
    _ = yaml.Unmarshal(renderedYaml, &resYaml)

    var expectedYaml any
    err := yaml.Unmarshal([]byte(expected), &expectedYaml)
    assert.Nil(err)

    return expectedYaml, resYaml, renderedYaml
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
            "/pets": {"GET", "POST"},
            "/pets/{id}": {"GET", "DELETE"},
        }
        assert.Equal(expected, res)
    })

    t.Run("FindOperation", func(t *testing.T) {
        op := doc.FindOperation("/pets", "GET")
        assert.NotNil(op)
        libOp, ok := op.(*LibV3Operation)
        assert.True(ok)

        assert.Equal(2, len(op.GetParameters()))
        assert.Equal("findPets", libOp.OperationId)
    })

    t.Run("FindOperation-res-not-found", func(t *testing.T) {
        op := doc.FindOperation("/pets2", "GET")
        assert.Nil(op)
    })

    t.Run("FindOperation-method-not-found", func(t *testing.T) {
        op := doc.FindOperation("/pets", "PATCH")
        assert.Nil(op)
    })
}

func TestLibV3Operation(t *testing.T) {
    assert := assert2.New(t)
    t.Parallel()
    doc := CreateLibDocumentFromFile(t, filepath.Join("test_fixtures", "document-petstore.yml"))

    t.Run("GetResponse", func(t *testing.T) {
        op := doc.FindOperation("/pets", "GET")
        res, code := op.GetResponse()
        assert.Equal(200, code)
        println(res)
    })
}

func TestLibV3Response(t *testing.T) {
    assert := assert2.New(t)
    t.Parallel()
    doc := CreateLibDocumentFromFile(t, filepath.Join("test_fixtures", "document-petstore.yml"))

    t.Run("GetContent", func(t *testing.T) {
        op := doc.FindOperation("/pets", "GET")
        res, _ := op.GetResponse()
        content, contentType := res.GetContent()

        assert.Equal("application/json", contentType)
        assert.NotNil(content.Items)
        assert.Equal("array", content.Type)
        assert.Equal("object", content.Items.Type)
        assert.Equal([]string{"name", "id"}, content.Items.Required)
    })
}

func TestNewSchemaFromLibOpenAPI(t *testing.T) {
    assert := assert2.New(t)
    t.Parallel()
    doc := CreateLibDocumentFromFile(t, filepath.Join("test_fixtures", "document-files-circular.yml")).(*LibV3Document)
    libSchema := doc.Model.Paths.PathItems["/files"].Get.Responses.Codes["200"].Content["application/json"].Schema.Schema()

    res := NewSchemaFromLibOpenAPI(NormalizeLibOpenAPISchema(libSchema, nil))

    assert.NotNil(res)
    assert.True(true)
}

func TestNewLibOpenAPIDocumentFromFile(t *testing.T) {

}

func TestCollectLibObjects(t *testing.T) {
    assert := assert2.New(t)
    t.Parallel()
    doc := CreateLibDocumentFromFile(t, filepath.Join("test_fixtures", "person-with-friends.yml")).(*LibV3Document)

    t.Run("SimpleArray", func(t *testing.T) {
        libSchema := doc.Model.Components.Schemas["SimpleArray"].Schema()
        assert.NotNil(libSchema)

        res := collectLibObjects(libSchema, nil)
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

        res := collectLibObjects(libSchema, nil)
        expected := `
type: array
items:
  type: object
  properties:
    name:
      type: string
`
        AssertLibYamlEqual(t, res, expected)
    })

    t.Run("SimpleObjectCircular", func(t *testing.T) {
        libSchema := doc.Model.Components.Schemas["SimpleObjectCircular"].Schema()
        assert.NotNil(libSchema)

        res := collectLibObjects(libSchema, nil)
        expected := `
type: object
properties:
    user:
        type: object
        properties:
            name:
                type: string
    relatives:
        type: array
        items:
            type: object
            properties:
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

        res := collectLibObjects(libSchema, nil)
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
        AssertLibYamlEqual(t, res, expected)
    })

    t.Run("AddressWithAllOf", func(t *testing.T) {
        libSchema := doc.Model.Components.Schemas["AddressWithAllOf"].Schema()
        assert.NotNil(libSchema)

        res := collectLibObjects(libSchema, nil)
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

        res := collectLibObjects(libSchema, nil)
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
}

func TestCollectLibArrays(t *testing.T) {
    assert := assert2.New(t)
    t.Parallel()
    doc := CreateLibDocumentFromFile(t, filepath.Join("test_fixtures", "person-with-friends.yml")).(*LibV3Document)

    t.Run("ArrayOfPersonAndEmployeeWithFriends", func(t *testing.T) {
        libSchema := doc.Model.Components.Schemas["ArrayOfPersonAndEmployeeWithFriends"].Schema()
        assert.NotNil(libSchema)

        res := collectLibArrays(libSchema, nil)
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
`
        expectedYaml, actualYaml, rendered := GetLibYamlExpectations(t, res, expected)
        assert.Greater(len(rendered), 0)
        assert.Equal(expectedYaml, actualYaml)
    })
}

func TestPicklLibOpenAPISchemaProxy(t *testing.T) {

}
