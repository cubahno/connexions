//go:build !integration

package kin

import (
	"github.com/cubahno/connexions/config"
	"github.com/cubahno/connexions/openapi"
	"github.com/cubahno/connexions/openapi/providers"
	"github.com/getkin/kin-openapi/openapi3"
	assert2 "github.com/stretchr/testify/assert"
	"net/http"
	"path/filepath"
	"testing"
)

func TestNewDocumentFromFile(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	t.Run("file-not-found", func(t *testing.T) {
		doc, err := NewDocumentFromFile(filepath.Join("non-existent.yml"))
		assert.Nil(doc)
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
	assert := assert2.New(t)
	t.Parallel()

	tc := &providers.OperationTestCase{
		DocFactory: NewDocumentFromFile,
	}
	tc.Run(t)

	t.Run("ID", func(t *testing.T) {
		operation := &KinOperation{Operation: &openapi3.Operation{
			OperationID: "findNice",
		}}
		res := operation.ID()
		assert.Equal("findNice", res)
	})

	t.Run("getParameters-nil-case", func(t *testing.T) {
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
		res := operation.getParameters(nil)
		expected := openapi.Parameters{
			{Name: "name"},
		}
		assert.Equal(expected, res)
	})

	t.Run("getParameters has headers", func(t *testing.T) {
		operation := &KinOperation{Operation: &openapi3.Operation{
			Parameters: openapi3.Parameters{
				{
					Value: &openapi3.Parameter{Name: "code", In: "header"},
				},
			},
			Security: &openapi3.SecurityRequirements{
				{
					"HTTPBearer": {},
					"APIKey":     {},
				},
			},
		}}
		securityComponents := openapi.SecurityComponents{
			"HTTPBearer": &openapi.SecurityComponent{
				Type:   openapi.AuthTypeHTTP,
				Scheme: openapi.AuthSchemeBearer,
			},
			"APIKey": &openapi.SecurityComponent{
				Type: openapi.AuthTypeApiKey,
				In:   openapi.AuthLocationQuery,
				Name: "x-api-key",
			},
		}

		expected := openapi.Parameters{
			{
				Name: "code",
				In:   "header",
			},
			{
				Name:     "authorization",
				In:       "header",
				Required: true,
				Schema: &openapi.Schema{
					Type:   "string",
					Format: "bearer",
				},
			},
			{
				Name:     "x-api-key",
				In:       "query",
				Required: true,
				Schema: &openapi.Schema{
					Type: "string",
				},
			},
		}

		res := operation.getParameters(securityComponents)

		assert.Equal(expected, res)
	})

	t.Run("getSecurity if present returned", func(t *testing.T) {
		operation := &KinOperation{Operation: &openapi3.Operation{
			Security: &openapi3.SecurityRequirements{
				{
					"HTTPBearer": {},
					"APIKey":     {},
					"Foo":        {},
				}},
		},
		}
		res := operation.getSecurity()

		expected := []string{"HTTPBearer", "APIKey", "Foo"}
		assert.Equal(expected, res)
	})

	t.Run("getSecurity if not present nil returned", func(t *testing.T) {
		operation := &KinOperation{Operation: &openapi3.Operation{}}
		res := operation.getSecurity()
		assert.Equal(0, len(res))
	})

	t.Run("GetResponse-headers-with-nil-cases", func(t *testing.T) {
		operation := &KinOperation{Operation: &openapi3.Operation{
			Responses: openapi3.NewResponses(
				openapi3.WithStatus(http.StatusOK, &openapi3.ResponseRef{
					Value: &openapi3.Response{
						Headers: openapi3.Headers{
							"X-Rate-Limit-Limit": &openapi3.HeaderRef{
								Value: &openapi3.Header{
									Parameter: openapi3.Parameter{
										Name: "X-Rate-Limit-Limit",
										Schema: &openapi3.SchemaRef{
											Value: &openapi3.Schema{
												Type: &openapi3.Types{"integer"},
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
				}),
			),
		}}
		res := operation.GetResponse()
		expected := openapi.Response{
			StatusCode: http.StatusOK,
			Headers: openapi.Headers{
				"x-rate-limit-limit": {
					Name: "x-rate-limit-limit",
					In:   openapi.ParameterInHeader,
					Schema: &openapi.Schema{
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

	t.Run("nil-schema", func(t *testing.T) {
		operation := &KinOperation{Operation: &openapi3.Operation{
			OperationID: "findNice",
		}}
		mediaTypes := map[string]*openapi3.MediaType{
			"text/plain": {
				Schema: nil,
			},
		}
		content, contentType := operation.getContent(mediaTypes)
		assert.Nil(content)
		assert.Equal("text/plain", contentType)
	})
}

func TestNewSchema(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()
	testData := filepath.Join("..", "..", "..", "testdata")

	t.Run("NewSchemaSuite", func(t *testing.T) {
		getSchema := func(t *testing.T, fileName, componentID string, parseConfig *config.ParseConfig) *openapi.Schema {
			t.Helper()
			kinDoc, err := NewDocumentFromFile(filepath.Join(testData, fileName))
			assert.Nil(err)
			doc := kinDoc.(*Document)
			kinSchema := doc.Components.Schemas[componentID].Value
			assert.NotNil(kinSchema)

			return NewSchemaFromKin(kinSchema, parseConfig)
		}
		tc := &providers.NewSchemaTestSuite{
			SchemaFactory: getSchema,
		}

		tc.Run(t)
	})

	t.Run("nested-all-of", func(t *testing.T) {
		target := openapi3.NewSchema()
		CreateSchemaFromYAMLFile(t, filepath.Join(testData, "schema-with-nested-all-of.yml"), target)

		res := NewSchemaFromKin(target, nil)
		assert.NotNil(res)

		expected := &openapi.Schema{
			Type: openapi.TypeObject,
			Properties: map[string]*openapi.Schema{
				"name":   {Type: openapi.TypeString},
				"age":    {Type: openapi.TypeInteger},
				"league": {Type: openapi.TypeString},
				"rating": {Type: openapi.TypeInteger},
				"tag":    {Type: openapi.TypeString},
			},
		}
		a, b := GetJSONPair(expected, res)
		if a != b {
			t.Errorf("expected / actual: \n%s\n%s", a, b)
		}
	})

	t.Run("with-parse-config-applied", func(t *testing.T) {
		target := openapi3.NewSchema()
		CreateSchemaFromYAMLFile(t, filepath.Join(testData, "document-petstore.yml"), target)

		res := newSchemaFromKin(target, &config.ParseConfig{MaxLevels: 1}, nil, []string{"user", "id"})
		assert.Nil(res)
	})

	t.Run("with-circular-detected", func(t *testing.T) {
		target := openapi3.NewSchema()
		CreateSchemaFromYAMLFile(t, filepath.Join(testData, "document-petstore.yml"), target)

		res := newSchemaFromKin(target, &config.ParseConfig{}, []string{"#/components/User", "#/components/User"}, []string{"user", "id"})
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

func TestMergeSubSchemas(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	t.Run("implied-string-type", func(t *testing.T) {
		target := &openapi3.Schema{
			OneOf: openapi3.SchemaRefs{
				{Value: &openapi3.Schema{Type: &openapi3.Types{openapi.TypeString}}},
			},
			Not: &openapi3.SchemaRef{
				Value: &openapi3.Schema{Enum: []any{"doggie"}},
			},
		}

		schema, ref := mergeKinSubSchemas(target)
		assert.Nil(schema.OneOf)
		assert.Equal("", ref)
		assert.Equal(openapi.TypeString, schema.Type)
	})

	t.Run("implied-array-type", func(t *testing.T) {
		target := &openapi3.Schema{
			OneOf: openapi3.SchemaRefs{
				{Value: &openapi3.Schema{
					Items: &openapi3.SchemaRef{
						Value: &openapi3.Schema{Type: &openapi3.Types{openapi.TypeString}},
					},
				}},
			},
		}

		schema, ref := mergeKinSubSchemas(target)
		assert.Nil(schema.OneOf)
		assert.Equal("", ref)
		assert.Equal(openapi.TypeArray, schema.Type)
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
		assert.Equal(openapi.TypeObject, schema.Type)
	})

	t.Run("inferred-from-enum", func(t *testing.T) {
		target := &openapi3.Schema{
			Enum: []any{1, 2, 3},
		}

		schema, _ := mergeKinSubSchemas(target)
		assert.Equal(openapi.TypeInteger, schema.Type)
	})
}

func TestPickSchemaProxy(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	t.Run("happy-path", func(t *testing.T) {
		items := []*openapi3.SchemaRef{
			nil,
			{Value: &openapi3.Schema{Type: &openapi3.Types{openapi.TypeString}}},
			{Value: &openapi3.Schema{Type: &openapi3.Types{openapi.TypeInteger}}},
		}
		res := pickKinSchemaProxy(items)
		assert.Equal(items[1], res)
	})

	t.Run("prefer-reference", func(t *testing.T) {
		items := []*openapi3.SchemaRef{
			nil,
			{Value: &openapi3.Schema{Type: &openapi3.Types{openapi.TypeString}}},
			{Ref: "#ref"},
		}
		res := pickKinSchemaProxy(items)
		assert.Equal(items[2], res)
	})
}

func TestGetAdditionalProperties(t *testing.T) {
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
			Type: &openapi3.Types{openapi.TypeString},
		}
		assert.Equal(expected, res)
	})

	t.Run("inlined-object", func(t *testing.T) {
		source := openapi3.AdditionalProperties{
			Schema: &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type: &openapi3.Types{openapi.TypeObject},
					Properties: map[string]*openapi3.SchemaRef{
						"name": {
							Value: &openapi3.Schema{
								Type: &openapi3.Types{openapi.TypeString},
							},
						},
						"age": {
							Value: &openapi3.Schema{
								Type: &openapi3.Types{openapi.TypeInteger},
							},
						},
					},
				},
			},
		}

		expected := &openapi3.Schema{
			Type: &openapi3.Types{openapi.TypeObject},
			Properties: map[string]*openapi3.SchemaRef{
				"name": {
					Value: &openapi3.Schema{Type: &openapi3.Types{openapi.TypeString}},
				},
				"age": {
					Value: &openapi3.Schema{Type: &openapi3.Types{openapi.TypeInteger}},
				},
			},
		}

		res := getKinAdditionalProperties(source)

		assert.Equal(expected, res)
	})
}
