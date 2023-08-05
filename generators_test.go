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

		// sorted order
		expected := "file-id=foo&id=123"
		assert.Equal(t, expected, res)
	})
}

func TestGenerateContentObject(t *testing.T) {
	t.Run("test case 1", func(t *testing.T) {
		schema := &openapi3.Schema{}
		src := `
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
        }`

		err := json.Unmarshal([]byte(src), schema)
		if err != nil {
			t.Fail()
		}

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
