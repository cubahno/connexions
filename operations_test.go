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
		valueResolver := func(schema *openapi3.Schema, state *ResolveState) any {
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
		res := GenerateURL(path, valueResolver, params)
		assert.Equal(t, "/users/123/foo", res)
	})
}

func TestGenerateQuery(t *testing.T) {
	t.Run("params correctly replaced in query", func(t *testing.T) {
		valueResolver := func(schema *openapi3.Schema, state *ResolveState) any {
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
		res := GenerateQuery(valueResolver, params)

		expected := "id=123&file-id=foo"
		assert.Equal(t, expected, res)
	})

	t.Run("arrays in url", func(t *testing.T) {
		valueResolver := func(schema *openapi3.Schema, state *ResolveState) any {
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
		res := GenerateQuery(valueResolver, params)

		expected := "tags[]=foo+bar&tags[]=foo+bar"
		assert.Equal(t, expected, res)
	})
}

func TestGenerateContent(t *testing.T) {
	t.Run("base-case", func(t *testing.T) {
		valueResolver := func(schema *openapi3.Schema, state *ResolveState) any {
			switch state.NamePath[len(state.NamePath)-1] {
			case "id":
				return 21
			case "score":
				return 11.5
			case "limit":
				return 100
			case "tag1":
				return "#dice"
			case "tag2":
				return "#nice"
			case "offset":
				return -1
			case "query":
				return "games"
			case "first":
				return 10
			case "second":
				return 20
			case "last":
				return 30
			}
			return nil
		}
		schema := CreateSchemaFromString(t, `
        {
            "type": "object",
            "properties": {
                "user": {
                    "type": "object",
                    "properties": {
                        "id": {
                            "type": "integer"
                        },
                        "score": {
                            "type": "number"
                        }
                    },
					"required": ["id"]
                },
                "pages": {
					"type": "array",
					"items": {
						"type": "object",
						"allOf": [
							{
								"type": "object",
								"properties": {
									"limit": {"type": "integer"},
									"tag1": {"type": "string"}
								},
						        "required": ["limit"]
							},
							{
								"type": "object",
								"properties": {"tag2": {"type": "string"}}
							}
						],
						"anyOf": [
							{
								"type": "object",
								"properties": {
									"offset": {"type": "integer"}
								},
						        "required": ["offset"]
							},
							{
								"type": "object",
								"properties": {
									"query": {"type": "string"}
								},
						        "required": ["query"]
							}
						],
						"oneOf": [
							{
								"type": "object",
								"properties": {
									"first": {"type": "integer"},
									"second": {"type": "integer"}
								},
                                "required": ["first", "second"]
							},
							{
								"type": "object",
								"properties": {
									"last": {"type": "integer"}
								},
                                "required": ["last"]
							}
						],
						"not": {
							"type": "object",
							"properties": {
								"second": {"type": "integer"}
                            }
                        }
					}
                }
            }
        }`)
		res := GenerateContent(schema, valueResolver, nil)

		expected := map[string]any{
			"user": map[string]any{"id": 21, "score": 11.5},
			"pages": []any{
				map[string]any{
					"limit": 100, "tag1": "#dice", "tag2": "#nice", "offset": -1, "first": 10,
				},
				map[string]any{
					"limit": 100, "tag1": "#dice", "tag2": "#nice", "offset": -1, "first": 10,
				},
			},
		}
		assert.Equal(t, expected, res)
	})

	t.Run("with-nested-all-of", func(t *testing.T) {
		valueResolver := func(schema *openapi3.Schema, state *ResolveState) any {
			switch state.NamePath[len(state.NamePath)-1] {
			case "name":
				return "Jane Doe"
			case "age":
				return 30
			case "tag":
				return "#doe"
			case "league":
				return "premier"
			case "rating":
				return 345.6
			}
			return nil
		}

		schema := CreateSchemaFromString(t, `
        {
			"type": "object",
			"allOf": [
				{
					"type": "object",
					"properties": {
						"name": {"type": "string"}
					}
				},
				{
					"type": "object",
	                "allOf": [
						{
							"type": "object",
							"properties": {"age": {"type": "integer"}}
						},
						{
							"type": "object",
							"allOf": [
								{
									"type": "object",
									"properties": {"tag": {"type": "string"}}
								},
								{
									"type": "object",
									"allOf": [
										{
											"type": "object",
											"properties": {"league": {"type": "string"}}
										}
									]
								},
								{
									"type": "object",
									"properties": {"rating": {"type": "integer"}}
								}
							]
						}
					]
				}
			]
        }`)
		expected := map[string]any{"name": "Jane Doe", "age": 30, "tag": "#doe", "league": "premier", "rating": 345.6}

		res := GenerateContent(schema, valueResolver, nil)
		assert.Equal(t, expected, res)
	})

	t.Run("fast-track-used-with-object", func(t *testing.T) {
		dice := map[string]string{"nice": "very nice", "rice": "good rice"}

		valueResolver := func(schema *openapi3.Schema, state *ResolveState) any {
			switch state.NamePath[0] {
			case "nice":
				return "not so nice"
			case "rice":
				return "not a rice"
			case "dice":
				return dice
			}
			return nil
		}
		schema := CreateSchemaFromString(t, `
        {
            "type":"object",
            "properties": {
                "dice": {
                    "type": "object",
                    "properties": {
                        "nice": {
                            "type": "string"
                        },
                        "rice": {
                            "type": "string"
                        }
                    }
                }
            }
        }`)
		res := GenerateContent(schema, valueResolver, nil)

		expected := map[string]any{"dice": dice}
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

		valueResolver := func(schema *openapi3.Schema, state *ResolveState) any {
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
		res := generateContentObject(schema, valueResolver, nil)

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

		valueResolver := func(schema *openapi3.Schema, state *ResolveState) any {
			return "foo"
		}

		res := generateContentArray(schema, valueResolver, nil)
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

		valueResolver := func(schema *openapi3.Schema, state *ResolveState) any {
			callNum++
			items := []string{"a", "b", "c", "d"}
			return items[callNum]
		}

		res := generateContentArray(schema, valueResolver, nil)
		assert.ElementsMatch(t, []string{"a", "b", "c", "d"}, res)
	})
}
