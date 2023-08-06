package xs

import (
	"encoding/json"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewResponse(t *testing.T) {

}

func TestExtractResponse(t *testing.T) {
	t.Run("extract-response", func(t *testing.T) {
		operation := CreateOperationFromString(t, `
			{
				"responses": {
                    "500": {
                        "description": "Internal Server Error"
                    },
					"200": {
						"description": "OK",
						"content": {
							"application/json": {
								"schema": {
									"type": "object",
									"properties": {
										"foo": {
											"type": "string"	
										}	
									}
								}
							}
						}
					},
                    "400": {
                        "description": "Bad request"
                    }
				}
			}
		`)
		response, code := ExtractResponse(operation)

		assert.Equal(t, 200, code)
		assert.Equal(t, "OK", *response.Description)
		assert.NotNil(t, response.Content["application/json"])
	})

	t.Run("get-first-defined", func(t *testing.T) {
		operation := CreateOperationFromString(t, `
			{
				"responses": {
                    "500": {
                        "description": "Internal Server Error"
                    },
                    "400": {
                        "description": "Bad request"
                    }
				}
			}
		`)
		response, code := ExtractResponse(operation)

		assert.Equal(t, 500, code)
		assert.Equal(t, "Internal Server Error", *response.Description)
	})

	t.Run("get-default-if-nothing-else", func(t *testing.T) {
		operation := CreateOperationFromString(t, `
			{
				"responses": {
                    "default": {
                        "description": "unexpected error"
                    }
				}
			}
		`)
		response, code := ExtractResponse(operation)

		assert.Equal(t, 200, code)
		assert.Equal(t, "unexpected error", *response.Description)
	})
}

func TestTransformHTTPCode(t *testing.T) {
	type tc struct {
		name     string
		expected int
	}
	testCases := []tc{
		{"200", 200},
		{"2xx", 200},
		{"2XX", 200},
		{"default", 200},
		{"20x", 200},
		{"201", 201},
		{"*", 200},
		{"unknown", 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, TransformHTTPCode(tc.name))
		})
	}
}

func TestGetContentType(t *testing.T) {
	t.Run("get-first-prioritized", func(t *testing.T) {
		content := openapi3.Content{
			"text/html": {
				Schema: &openapi3.SchemaRef{Value: &openapi3.Schema{}},
			},
			"application/json": {
				Schema: &openapi3.SchemaRef{Value: &openapi3.Schema{}},
			},
			"text/plain": {
				Schema: &openapi3.SchemaRef{Value: &openapi3.Schema{}},
			},
		}
		contentType, schema := GetContentType(content)

		assert.Equal(t, "application/json", contentType)
		assert.NotNil(t, schema)
	})

	t.Run("get-first-found", func(t *testing.T) {
		content := openapi3.Content{
			"multipart/form-data; boundary=something": {
				Schema: &openapi3.SchemaRef{},
			},
			"application/xml": {
				Schema: &openapi3.SchemaRef{},
			},
		}
		contentType, _ := GetContentType(content)

		assert.Contains(t, []string{"multipart/form-data; boundary=something", "application/xml"}, contentType)
	})

	t.Run("nothing-found", func(t *testing.T) {
		content := openapi3.Content{}
		contentType, schema := GetContentType(content)

		assert.Equal(t, "", contentType)
		assert.Nil(t, schema)
	})
}

func TestGenerateURL(t *testing.T) {
	t.Run("params correctly replaced in path", func(t *testing.T) {
		path := "/users/{id}/{file-id}"
		valueMaker := func(schema *openapi3.Schema, state *ResolveState) any {
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
		valueMaker := func(schema *openapi3.Schema, state *ResolveState) any {
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
		valueMaker := func(schema *openapi3.Schema, state *ResolveState) any {
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

		valueMaker := func(schema *openapi3.Schema, state *ResolveState) any {
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

		valueMaker := func(schema *openapi3.Schema, state *ResolveState) any {
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

		valueMaker := func(schema *openapi3.Schema, state *ResolveState) any {
			callNum++
			items := []string{"a", "b", "c", "d"}
			return items[callNum]
		}

		res := generateContentArray(schema, valueMaker, nil)
		assert.ElementsMatch(t, []string{"a", "b", "c", "d"}, res)
	})
}
