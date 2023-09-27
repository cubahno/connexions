//go:build !integration

package connexions

import (
	"github.com/getkin/kin-openapi/openapi3"
	assert2 "github.com/stretchr/testify/assert"
	"net/http"
	"path/filepath"
	"testing"
)

func TestNewKinDocumentFromFile(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	t.Run("file-not-found", func(t *testing.T) {
		doc, err := NewKinDocumentFromFile(filepath.Join("non-existent.yml"))
		assert.Nil(doc)
		assert.Error(err)
	})
}

func TestKinOperation(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	t.Run("ID", func(t *testing.T) {
		operation := &KinOperation{Operation: &openapi3.Operation{
			OperationID: "findNice",
		}}
		res := operation.ID()
		assert.Equal("findNice", res)
	})

	t.Run("GetParameters-nil-case", func(t *testing.T) {
		operation := &KinOperation{Operation: &openapi3.Operation{
			Parameters: openapi3.Parameters{
				{
					Value: &openapi3.Parameter{Name: "name"},
				},
				{
					Value: nil,
				},
			},
		}}
		res := operation.GetParameters()
		expected := OpenAPIParameters{
			{Name: "name"},
		}
		assert.Equal(expected, res)
	})

	t.Run("GetResponse-headers-with-nil-cases", func(t *testing.T) {
		operation := &KinOperation{Operation: &openapi3.Operation{
			Responses: openapi3.Responses{
				"200": &openapi3.ResponseRef{
					Value: &openapi3.Response{
						Headers: openapi3.Headers{
							"X-Rate-Limit-Limit": &openapi3.HeaderRef{
								Value: &openapi3.Header{
									Parameter: openapi3.Parameter{
										Name: "X-Rate-Limit-Limit",
										Schema: &openapi3.SchemaRef{
											Value: &openapi3.Schema{
												Type: "integer",
											},
										},
									},
								},
							},
							"X-Rate-Limit-Left": &openapi3.HeaderRef{
								Value: nil,
							},
						},
					},
				},
			},
		}}
		res := operation.GetResponse()
		expected := OpenAPIResponse{
			StatusCode: http.StatusOK,
			Headers: OpenAPIHeaders{
				"x-rate-limit-limit": {
					Name: "x-rate-limit-limit",
					In:   ParameterInHeader,
					Schema: &Schema{
						Type: "integer",
					},
				},
			},
		}
		AssertJSONEqual(t, expected, res)
	})

	t.Run("getContent-nil-case", func(t *testing.T) {
		operation := &KinOperation{Operation: &openapi3.Operation{
			Parameters: openapi3.Parameters{
				{
					Value: &openapi3.Parameter{Name: "name"},
				},
				{
					Value: nil,
				},
			},
		}}
		content, contentType := operation.getContent(nil)
		assert.Nil(content)
		assert.Equal("", contentType)
	})
}

func TestNewSchemaFromKin(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	t.Run("nested-all-of", func(t *testing.T) {
		target := openapi3.NewSchema()
		CreateSchemaFromYAMLFile(t, filepath.Join("test_fixtures", "schema-with-nested-all-of.yml"), target)

		res := NewSchemaFromKin(target, nil)
		assert.NotNil(res)

		expected := &Schema{
			Type: TypeObject,
			Properties: map[string]*Schema{
				"name":   {Type: TypeString},
				"age":    {Type: TypeInteger},
				"league": {Type: TypeString},
				"rating": {Type: TypeInteger},
				"tag":    {Type: TypeString},
			},
		}
		a, b := GetJSONPair(expected, res)
		if a != b {
			t.Errorf("expected / actual: \n%s\n%s", a, b)
		}
	})

	t.Run("with-parse-config-applied", func(t *testing.T) {
		target := openapi3.NewSchema()
		CreateSchemaFromYAMLFile(t, filepath.Join("test_fixtures", "document-petstore.yml"), target)

		res := newSchemaFromKin(target, &ParseConfig{MaxLevels: 1}, nil, []string{"user", "id"})
		assert.Nil(res)
	})

	t.Run("with-circular-detected", func(t *testing.T) {
		target := openapi3.NewSchema()
		CreateSchemaFromYAMLFile(t, filepath.Join("test_fixtures", "document-petstore.yml"), target)

		res := newSchemaFromKin(target, &ParseConfig{}, []string{"#/components/User", "#/components/User"}, []string{"user", "id"})
		assert.Nil(res)
	})

	t.Run("min-amount-of-additional-props", func(t *testing.T) {
		js := `
			{
			  "type": "object",
			  "minProperties": 5,
			  "properties": {
				"name": {
				  "type": "string"
				}
			  },
			  "additionalProperties": {
				"type": "string"
			  }
			}
			`
		schema := CreateKinSchemaFromString(t, js)
		assert.NotNil(schema)

		res := NewSchemaFromKin(schema, nil)
		assert.NotNil(res)
		assert.Equal(6, len(res.Properties))
	})
}

func TestMergeKinSubSchemas(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	t.Run("implied-string-type", func(t *testing.T) {
		target := &openapi3.Schema{
			OneOf: openapi3.SchemaRefs{
				{Value: &openapi3.Schema{Type: "string"}},
			},
			Not: &openapi3.SchemaRef{
				Value: &openapi3.Schema{Enum: []any{"doggie"}},
			},
		}

		schema, ref := mergeKinSubSchemas(target)
		assert.Nil(schema.OneOf)
		assert.Equal("", ref)
		assert.Equal(TypeString, schema.Type)
	})

	t.Run("implied-array-type", func(t *testing.T) {
		target := &openapi3.Schema{
			OneOf: openapi3.SchemaRefs{
				{Value: &openapi3.Schema{
					Items: &openapi3.SchemaRef{
						Value: &openapi3.Schema{Type: "string"},
					},
				}},
			},
		}

		schema, ref := mergeKinSubSchemas(target)
		assert.Nil(schema.OneOf)
		assert.Equal("", ref)
		assert.Equal(TypeArray, schema.Type)
	})

	t.Run("implied-lastly-object-type", func(t *testing.T) {
		target := &openapi3.Schema{
			OneOf: openapi3.SchemaRefs{
				{Value: &openapi3.Schema{}},
			},
		}

		schema, ref := mergeKinSubSchemas(target)
		assert.Nil(schema.OneOf)
		assert.Equal("", ref)
		assert.Equal(TypeObject, schema.Type)
	})

	t.Run("inferred-from-enum", func(t *testing.T) {
		target := &openapi3.Schema{
			Enum: []any{1, 2, 3},
		}

		schema, _ := mergeKinSubSchemas(target)
		assert.Equal(TypeInteger, schema.Type)
	})
}

func TestPickKinSchemaProxy(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	t.Run("happy-path", func(t *testing.T) {
		items := []*openapi3.SchemaRef{
			nil,
			{Value: &openapi3.Schema{Type: "string"}},
			{Value: &openapi3.Schema{Type: "integer"}},
		}
		res := pickKinSchemaProxy(items)
		assert.Equal(items[1], res)
	})

	t.Run("prefer-reference", func(t *testing.T) {
		items := []*openapi3.SchemaRef{
			nil,
			{Value: &openapi3.Schema{Type: "string"}},
			{Ref: "#ref"},
		}
		res := pickKinSchemaProxy(items)
		assert.Equal(items[2], res)
	})
}

func TestGetKinAdditionalProperties(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	t.Run("empty-case", func(t *testing.T) {
		res := getKinAdditionalProperties(openapi3.AdditionalProperties{})
		assert.Nil(res)
	})

	t.Run("has-case", func(t *testing.T) {
		has := new(bool)
		*has = true
		res := getKinAdditionalProperties(openapi3.AdditionalProperties{Has: has})
		expected := &openapi3.Schema{
			Type: TypeString,
		}
		assert.Equal(expected, res)
	})

	t.Run("inlined-object", func(t *testing.T) {
		source := openapi3.AdditionalProperties{
			Schema: &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type: TypeObject,
					Properties: map[string]*openapi3.SchemaRef{
						"name": {
							Value: &openapi3.Schema{
								Type: TypeString,
							},
						},
						"age": {
							Value: &openapi3.Schema{
								Type: TypeInteger,
							},
						},
					},
				},
			},
		}

		expected := &openapi3.Schema{
			Type: TypeObject,
			Properties: map[string]*openapi3.SchemaRef{
				"name": {
					Value: &openapi3.Schema{Type: TypeString},
				},
				"age": {
					Value: &openapi3.Schema{Type: TypeInteger},
				},
			},
		}

		res := getKinAdditionalProperties(source)

		assert.Equal(expected, res)
	})
}
