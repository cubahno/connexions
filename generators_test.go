package xs

import (
	"encoding/json"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGenerateURL(t *testing.T) {
	t.Run("params correctly replaced in path", func(t *testing.T) {
		path := "/users/{id}/{file-id}"
		valueMaker := func(schema *openapi3.Schema, state *GeneratorState) any {
			if state.NamePath[0] == "id" {
				return 123
			}
			if state.NamePath[0] == "file-id" {
				return "foo"
			}
			return "something-else"
		}
		params := openapi3.Parameters{
			{
				Value: &openapi3.Parameter{
					Name: "id",
					In:   "path",
					Schema: &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type: "integer",
						},
					},
				},
			},
			{
				Value: &openapi3.Parameter{
					Name: "file-id",
					In:   "path",
					Schema: &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type: "string",
						},
					},
				},
			},
			{
				Value: &openapi3.Parameter{
					Name: "file-id",
					In:   "query",
					Schema: &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type: "integer",
						},
					},
				},
			},
		}
		res := GenerateURL(path, valueMaker, params)
		assert.Equal(t, "/users/123/foo", res)
	})
}

func TestGenerateQuery(t *testing.T) {
	t.Run("params correctly replaced in query", func(t *testing.T) {
		valueMaker := func(schema *openapi3.Schema, state *GeneratorState) any {
			if state.NamePath[0] == "id" {
				return 123
			}
			if state.NamePath[0] == "file-id" {
				return "foo"
			}
			return "something-else"
		}
		params := openapi3.Parameters{
			{
				Value: &openapi3.Parameter{
					Name: "id",
					In:   "query",
					Schema: &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type: "integer",
						},
					},
				},
			},
			{
				Value: &openapi3.Parameter{
					Name: "file-id",
					In:   "query",
					Schema: &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type: "foo",
						},
					},
				},
			},
		}
		res := GenerateQuery(valueMaker, params)

		expected := "id=123&file-id=foo"
		assert.Equal(t, expected, res)
	})

	t.Run("arrays in url", func(t *testing.T) {
		valueMaker := func(schema *openapi3.Schema, state *GeneratorState) any {
			return "foo bar"
		}
		params := openapi3.Parameters{
			{
				Value: &openapi3.Parameter{
					Name: "tags",
					In:   "query",
					Schema: &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type: "array",
							Items: &openapi3.SchemaRef{
								Value: &openapi3.Schema{
									Type: "string",
								},
							},
						},
					},
				},
			},
		}
		res := GenerateQuery(valueMaker, params)

		expected := "tags[]=foo+bar&tags[]=foo+bar"
		assert.Equal(t, expected, res)
	})
}

func TestGenerateContentObject(t *testing.T) {
	t.Run("test case 1", func(t *testing.T) {
		schema := CreateSchemaFromString(t, `
        {
            "type":"object",
            "properties": {
                "name": {
                    "type": "object",
                    "properties": {
                        "first": {
                            "type": "string"
                        },
                        "last": {
                            "type": "string"
                        }
                    }
                },
                "age": {
                    "type": "integer"
                }
            }
        }`)

		valueMaker := func(schema *openapi3.Schema, state *GeneratorState) any {
			namePath := state.NamePath
			for _, name := range namePath {
				if name == "first" {
					return "Jane"
				} else if name == "last" {
					return "Doe"
				} else if name == "age" {
					return 21
				}
			}
			return nil
		}
		res := generateContentObject(schema, valueMaker, nil)

		expected := `{"age":21,"name":{"first":"Jane","last":"Doe"}}`
		resJs, _ := json.Marshal(res)
		assert.Equal(t, expected, string(resJs))
	})
}

func TestGenerateContentArray(t *testing.T) {
	t.Run("generate simple array without min/max items", func(t *testing.T) {
		schema := CreateSchemaFromString(t, `{
            "type": "array",
            "items": {
                "type": "string"
            }
        }`)

		valueMaker := func(schema *openapi3.Schema, state *GeneratorState) any {
			return "foo"
		}

		res := generateContentArray(schema, valueMaker, nil)
		assert.ElementsMatch(t, []string{"foo", "foo"}, res)
	})

	t.Run("generate simple array", func(t *testing.T) {
		schema := CreateSchemaFromString(t, `{
            "type": "array",
			"minItems": 3,
            "items": {
                "type": "string"
            }
        }`)

		callNum := -1

		valueMaker := func(schema *openapi3.Schema, state *GeneratorState) any {
			callNum++
			items := []string{"a", "b", "c", "d"}
			return items[callNum]
		}

		res := generateContentArray(schema, valueMaker, nil)
		assert.ElementsMatch(t, []string{"a", "b", "c", "d"}, res)
	})
}
