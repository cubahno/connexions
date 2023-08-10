package xs

import (
	"encoding/json"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewRequest(t *testing.T) {
	t.Run("base-case", func(t *testing.T) {
		valueResolver := func(schema *openapi3.Schema, state *ResolveState) any {
			if state.NamePath[0] == "userId" {
				return "123"
			}
			if schema.Example != nil {
				return schema.Example
			}
			return schema.Default
		}

		operation := CreateOperationFromString(t, `
{
  "operationId": "createUser",
  "parameters": [
    {
      "name": "userId",
      "in": "path",
      "description": "The unique identifier of the user.",
      "required": true,
      "schema": {
        "type": "string"
      }
    },
    {
      "name": "limit",
      "in": "query",
      "required": false,
      "schema": {
        "type": "integer",
        "minimum": 1,
        "maximum": 100,
        "default": 10
      }
    },
    {
      "name": "lang",
      "in": "header",
      "description": "The language preference for the response.",
      "required": false,
      "schema": {
        "type": "string",
        "enum": [
          "en",
          "es",
          "de"
        ],
        "default": "de"
      }
    }
  ],
  "requestBody": {
    "description": "JSON payload containing user information.",
    "required": true,
    "content": {
      "application/json": {
        "schema": {
          "type": "object",
          "properties": {
            "username": {
              "type": "string",
              "description": "The username of the new user.",
              "example": "john_doe"
            },
            "email": {
              "type": "string",
              "format": "email",
              "description": "The email address of the new user.",
              "example": "john.doe@example.com"
            }
          },
          "required": [
            "username",
            "email"
          ]
        }
      }
    }
  },
  "responses": {
    "500": {
      "description": "Internal Server Error"
    },
    "200": {
      "description": "User account successfully created.",
      "headers": {
        "Location": {
          "description": "The URL of the newly created user account.",
          "schema": {
            "type": "string"
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
		req := NewRequestFromOperation("/foo", "/users/{userId}", "POST", operation, valueResolver)

		expectedBody := map[string]any{
			"username": "john_doe",
			"email":    "john.doe@example.com",
		}

		expectedHeaders := map[string]any{"lang": "de"}

		assert.Equal(t, "POST", req.Method)
		assert.Equal(t, "/foo/users/123", req.Path)
		assert.Equal(t, "limit=10", req.Query)
		assert.Equal(t, "application/json", req.ContentType)
		assert.Equal(t, expectedBody, req.Body)
		assert.Equal(t, expectedHeaders, req.Headers)
	})
}

func TestNewResponse(t *testing.T) {
	t.Run("base-case", func(t *testing.T) {
		valueResolver := func(schema *openapi3.Schema, state *ResolveState) any {
			if state.NamePath[0] == "userId" {
				return 123
			}
			if schema.Example != nil {
				return schema.Example
			}
			return schema.Default
		}

		operation := CreateOperationFromString(t, `
{
  "operationId": "createUser",
  "parameters": [
    {
      "name": "userId",
      "in": "path",
      "description": "The unique identifier of the user.",
      "required": true,
      "schema": {
        "type": "string"
      }
    },
    {
      "name": "limit",
      "in": "query",
      "required": false,
      "schema": {
        "type": "integer",
        "minimum": 1,
        "maximum": 100,
        "default": 10
      }
    },
    {
      "name": "lang",
      "in": "header",
      "description": "The language preference for the response.",
      "required": false,
      "schema": {
        "type": "string",
        "enum": [
          "en",
          "es",
          "de"
        ],
        "default": "de"
      }
    }
  ],
  "responses": {
    "500": {
      "description": "Internal Server Error"
    },
    "200": {
      "description": "User account successfully created.",
      "headers": {
        "Location": {
          "description": "The URL of the newly created user account.",
          "schema": {
            "type": "string",
            "example": "https://example.com/users/123"
          }
        }
      },
      "content": {
        "application/json": {
          "schema": {
            "type": "object",
            "properties": {
              "id": {
                "type": "integer",
                "example": 123
              },
              "email": {
                "type": "string",
                "example": "jane.doe@example.com"
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
		res := NewResponseFromOperation(operation, valueResolver)

		expectedHeaders := map[string]any{
			"location":     "https://example.com/users/123",
			"content-type": "application/json",
		}
		expectedContent := map[string]any{
			"id":    float64(123),
			"email": "jane.doe@example.com",
		}

		assert.Equal(t, "application/json", res.ContentType)
		assert.Equal(t, 200, res.StatusCode)
		assert.Equal(t, expectedHeaders, res.Headers)
		assert.Equal(t, expectedContent, res.Content)
	})

	t.Run("no-content-type", func(t *testing.T) {
		valueResolver := func(schema *openapi3.Schema, state *ResolveState) any {
			if state.NamePath[0] == "userId" {
				return 123
			}
			if schema.Example != nil {
				return schema.Example
			}
			return schema.Default
		}

		operation := CreateOperationFromString(t, `
{
  "operationId": "createUser",
  "parameters": [
    {
      "name": "userId",
      "in": "path",
      "description": "The unique identifier of the user.",
      "required": true,
      "schema": {
        "type": "string"
      }
    }
  ],
  "responses": {
    "500": {
      "description": "Internal Server Error"
    },
    "200": {
      "description": "User account successfully created.",
      "headers": {
        "Location": {
          "description": "The URL of the newly created user account.",
          "schema": {
            "type": "string",
            "example": "https://example.com/users/123"
          }
        }
      },
      "400": {
        "description": "Bad request"
      }
    }
  }
}
		`)
		res := NewResponseFromOperation(operation, valueResolver)

		expectedHeaders := map[string]any{
			"location": "https://example.com/users/123",
		}

		assert.Equal(t, 200, res.StatusCode)
		assert.Equal(t, expectedHeaders, res.Headers)

		assert.Equal(t, "", res.ContentType)
		assert.Equal(t, nil, res.Content)
	})
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

		assert.Contains(t, []int{500, 400}, code)
		assert.Contains(t, []string{"Internal Server Error", "Bad request"}, *response.Description)
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
		res := GenerateURLFromSchemaParameters(path, valueResolver, params)
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

		// TODO(igor): fix order of query params
		assert.Contains(t, []string{"id=123&file-id=foo", "file-id=foo&id=123"}, res)
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

	t.Run("no-resolved-values", func(t *testing.T) {
		valueResolver := func(schema *openapi3.Schema, state *ResolveState) any {
			return nil
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
		}
		res := GenerateQuery(valueResolver, params)

		expected := "id="
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
		res := GenerateContentFromSchema(schema, valueResolver, nil)

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

		res := GenerateContentFromSchema(schema, valueResolver, nil)
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
		res := GenerateContentFromSchema(schema, valueResolver, nil)

		expected := map[string]any{"dice": dice}
		assert.Equal(t, expected, res)
	})

	t.Run("with-circular-array-references", func(t *testing.T) {
		valueResolver := func(schema *openapi3.Schema, state *ResolveState) any {
			switch state.NamePath[len(state.NamePath)-1] {
			case "id":
				return 123
			case "name":
				return "noda-123"
			}
			return nil
		}
		doc := CreateDocumentFromString(t, `
{
  "openapi": "3.0.3",
  "info": {
    "title": "Recursive API",
    "version": "1.0.0"
  },
  "paths": {
    "/nodes/{id}": {
      "get": {
        "summary": "Get a node by ID",
        "parameters": [
          {
            "name": "id",
            "in": "path",
            "description": "The ID of the node",
            "required": true,
            "schema": {
              "type": "integer"
            }
          }
        ],
        "responses": {
          "200": {
            "description": "Successful response",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/Node"
                }
              }
            }
          },
          "404": {
            "description": "Node not found"
          }
        }
      }
    }
  },
  "components": {
    "schemas": {
      "Node": {
        "type": "object",
        "properties": {
          "id": {
            "type": "integer"
          },
          "name": {
            "type": "string"
          },
          "children": {
            "type": "array",
            "items": {
              "$ref": "#/components/schemas/Node"
            }
          }
        }
      }
    }
  }
}
`)
		schema := doc.Paths["/nodes/{id}"].Get.Responses.Get(200).Value.Content.Get("application/json").Schema.Value
		res := GenerateContentFromSchema(schema, valueResolver, nil)

		expected := map[string]any{
			"id":   123,
			"name": "noda-123",
			"children": []any{
				map[string]any{
					"id":   123,
					"name": "noda-123",
				},
				map[string]any{
					"id":   123,
					"name": "noda-123",
				},
			},
		}
		assert.Equal(t, expected, res)
	})

	t.Run("with-circular-object-references", func(t *testing.T) {
		valueResolver := func(schema *openapi3.Schema, state *ResolveState) any {
			switch state.NamePath[len(state.NamePath)-1] {
			case "id":
				return 123
			case "name":
				return "noda-123"
			}
			return nil
		}
		doc := CreateDocumentFromString(t, `
{
  "openapi": "3.0.3",
  "info": {
    "title": "Recursive API",
    "version": "1.0.0"
  },
  "paths": {
    "/nodes/{id}": {
      "get": {
        "summary": "Get a node by ID",
        "parameters": [
          {
            "name": "id",
            "in": "path",
            "description": "The ID of the node",
            "required": true,
            "schema": {
              "type": "integer"
            }
          }
        ],
        "responses": {
          "200": {
            "description": "Successful response",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/Node"
                }
              }
            }
          },
          "404": {
            "description": "Node not found"
          }
        }
      }
    }
  },
  "components": {
    "schemas": {
      "Node": {
        "type": "object",
        "properties": {
          "id": {
            "type": "integer"
          },
          "name": {
            "type": "string"
          },
          "parent": {
            "$ref": "#/components/schemas/Node"
          }
        }
      }
    }
  }
}
`)
		schema := doc.Paths["/nodes/{id}"].Get.Responses.Get(200).Value.Content.Get("application/json").Schema.Value
		res := GenerateContentFromSchema(schema, valueResolver, nil)

		expected := map[string]any{
			"id":   123,
			"name": "noda-123",
			"parent": map[string]any{
				"id":   123,
				"name": "noda-123",
			},
		}
		assert.Equal(t, expected, res)
	})

	t.Run("with-no-resolved-values", func(t *testing.T) {
		schema := CreateSchemaFromString(t, `
        {
            "type":"object",
            "properties": {
                "name": {
                    "type": "object",
                    "properties": {
                        "first": {"type": "string"}
                    }
                }
            }
        }`)
		res := GenerateContentFromSchema(schema, nil, nil)
		assert.Nil(t, res)
	})
}

func TestGenerateContentObject(t *testing.T) {
	t.Run("GenerateContentObject", func(t *testing.T) {
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
		res := GenerateContentObject(schema, valueResolver, nil)

		expected := `{"age":21,"name":{"first":"Jane","last":"Doe"}}`
		resJs, _ := json.Marshal(res)
		assert.Equal(t, expected, string(resJs))
	})

	t.Run("with-no-properties", func(t *testing.T) {
		schema := CreateSchemaFromString(t, `{"type": "object"}`)
		res := GenerateContentObject(schema, nil, nil)
		assert.Nil(t, res)
	})

	t.Run("with-no-resolved-values", func(t *testing.T) {
		schema := CreateSchemaFromString(t, `
        {
            "type":"object",
            "properties": {
                "name": {
                    "type": "object",
                    "properties": {
                        "first": {"type": "string"}
                    }
                }
            }
        }`)
		res := GenerateContentObject(schema, nil, nil)
		assert.Nil(t, res)
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

		res := GenerateContentArray(schema, valueResolver, nil)
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

		res := GenerateContentArray(schema, valueResolver, nil)
		assert.ElementsMatch(t, []string{"a", "b", "c", "d"}, res)
	})

	t.Run("with-no-resolved-values", func(t *testing.T) {
		schema := CreateSchemaFromString(t, `{
            "type": "array",
			"minItems": 3,
            "items": {"type": "string"}
        }`)
		res := GenerateContentArray(schema, nil, nil)
		assert.Nil(t, res)
	})
}

func TestGenerateRequestBody(t *testing.T) {
	t.Run("GenerateRequestBody", func(t *testing.T) {
		valueResolver := func(schema *openapi3.Schema, state *ResolveState) any {
			namePath := state.NamePath
			for _, name := range namePath {
				if name == "foo" {
					return "bar"
				}
			}
			return nil
		}
		schema := CreateSchemaFromString(t, `{
			"type": "object",
			"properties": {
				"foo": {
					"type": "string"
				}
			}
	    }`)
		reqBodyRef := &openapi3.RequestBodyRef{
			Value: &openapi3.RequestBody{
				Content: openapi3.NewContentWithJSONSchema(schema),
			},
		}
		payload, contentType := GenerateRequestBody(reqBodyRef, valueResolver, nil)

		assert.Equal(t, "application/json", contentType)
		assert.Equal(t, map[string]any{"foo": "bar"}, payload)
	})

	t.Run("GenerateRequestBody-first-from-encountered", func(t *testing.T) {
		valueResolver := func(schema *openapi3.Schema, state *ResolveState) any {
			namePath := state.NamePath
			for _, name := range namePath {
				if name == "foo" {
					return "bar"
				}
			}
			return nil
		}

		schema := CreateSchemaFromString(t, `{
			"type": "object",
			"properties": {
				"foo": {
					"type": "string"
				}
			}
	    }`)
		reqBodyRef := &openapi3.RequestBodyRef{
			Value: &openapi3.RequestBody{
				Content: map[string]*openapi3.MediaType{
					"application/xml": {
						Schema: &openapi3.SchemaRef{Value: schema},
					},
				},
			},
		}
		payload, contentType := GenerateRequestBody(reqBodyRef, valueResolver, nil)

		assert.Equal(t, "application/xml", contentType)
		assert.Equal(t, map[string]any{"foo": "bar"}, payload)
	})

	t.Run("case-empty-body-reference", func(t *testing.T) {
		payload, contentType := GenerateRequestBody(nil, nil, nil)

		assert.Equal(t, "", contentType)
		assert.Equal(t, nil, payload)
	})

	t.Run("case-empty-schema", func(t *testing.T) {
		reqBodyRef := &openapi3.RequestBodyRef{}
		payload, contentType := GenerateRequestBody(reqBodyRef, nil, nil)

		assert.Equal(t, "", contentType)
		assert.Equal(t, nil, payload)
	})

	t.Run("case-empty-content-types", func(t *testing.T) {
		reqBodyRef := &openapi3.RequestBodyRef{Value: &openapi3.RequestBody{Content: nil}}
		payload, contentType := GenerateRequestBody(reqBodyRef, nil, nil)

		assert.Equal(t, "", contentType)
		assert.Equal(t, nil, payload)
	})
}

func TestGenerateRequestHeaders(t *testing.T) {
	t.Run("GenerateRequestHeaders", func(t *testing.T) {
		valueResolver := func(schema *openapi3.Schema, state *ResolveState) any {
			switch state.NamePath[len(state.NamePath)-1] {
			case "mode":
				return "dark"
			case "lang":
				return "de"
			case "x-key":
				return "abcdef"
			case "version":
				return "1.0.0"
			}
			return nil
		}
		params := openapi3.Parameters{
			{
				Value: &openapi3.Parameter{
					Name:   "X-Key",
					In:     openapi3.ParameterInHeader,
					Schema: &openapi3.SchemaRef{Value: &openapi3.Schema{Type: "string"}},
				},
			},
			{
				Value: &openapi3.Parameter{
					Name:   "Version",
					In:     openapi3.ParameterInHeader,
					Schema: &openapi3.SchemaRef{Value: &openapi3.Schema{Type: "string"}},
				},
			},
			{
				Value: &openapi3.Parameter{
					Name: "Preferences",
					In:   openapi3.ParameterInHeader,
					Schema: &openapi3.SchemaRef{Value: &openapi3.Schema{
						Type: "object",
						Properties: map[string]*openapi3.SchemaRef{
							"mode": {Value: &openapi3.Schema{Type: "string"}},
							"lang": {Value: &openapi3.Schema{Type: "string"}},
						},
					}},
				},
			},
			{
				Value: &openapi3.Parameter{
					Name:   "id",
					In:     openapi3.ParameterInPath,
					Schema: &openapi3.SchemaRef{Value: &openapi3.Schema{Type: "string"}},
				},
			},
		}

		expected := map[string]any{
			"x-key":       "abcdef",
			"version":     "1.0.0",
			"preferences": map[string]any{"mode": "dark", "lang": "de"},
		}

		res := GenerateRequestHeaders(params, valueResolver)
		assert.Equal(t, expected, res)
	})

	t.Run("param-is-nil", func(t *testing.T) {
		params := openapi3.Parameters{{}}
		res := GenerateRequestHeaders(params, nil)
		assert.Nil(t, res)
	})

	t.Run("schema-ref-is-nil", func(t *testing.T) {
		params := openapi3.Parameters{{Value: &openapi3.Parameter{Schema: nil, In: openapi3.ParameterInHeader}}}
		res := GenerateRequestHeaders(params, nil)
		assert.Nil(t, res)
	})

	t.Run("schema-is-nil", func(t *testing.T) {
		params := openapi3.Parameters{
			{
				Value: &openapi3.Parameter{
					Schema: &openapi3.SchemaRef{Value: nil},
					In:     openapi3.ParameterInHeader,
				},
			},
		}
		res := GenerateRequestHeaders(params, nil)
		assert.Nil(t, res)
	})
}

func TestGenerateResponseHeaders(t *testing.T) {
	t.Run("GenerateResponseHeaders", func(t *testing.T) {
		valueResolver := func(schema *openapi3.Schema, state *ResolveState) any {
			switch state.NamePath[len(state.NamePath)-1] {
			case "x-rate-limit-limit":
				return 100
			case "x-rate-limit-remaining":
				return 80
			}
			return nil
		}
		headers := openapi3.Headers{
			"X-Rate-Limit-Limit": {
				Value: &openapi3.Header{
					Parameter: openapi3.Parameter{
						Name:   "X-Key",
						In:     openapi3.ParameterInHeader,
						Schema: &openapi3.SchemaRef{Value: &openapi3.Schema{Type: "integer"}},
					},
				},
			},
			"X-Rate-Limit-Remaining": {
				Value: &openapi3.Header{
					Parameter: openapi3.Parameter{
						Name:   "X-Key",
						In:     openapi3.ParameterInHeader,
						Schema: &openapi3.SchemaRef{Value: &openapi3.Schema{Type: "integer"}},
					},
				},
			},
		}

		expected := map[string]any{
			"x-rate-limit-limit":     100,
			"x-rate-limit-remaining": 80,
		}

		res := GenerateResponseHeaders(headers, valueResolver)
		assert.Equal(t, expected, res)
	})
}

func TestMergeSubSchemas(t *testing.T) {
	t.Run("MergeSubSchemas", func(t *testing.T) {
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
                }
            },
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
        }`)
		res := MergeSubSchemas(schema)
		expectedProperties := []string{"user", "limit", "tag1", "tag2", "offset", "first"}

		resProps := make([]string, 0)
		for name, _ := range res.Properties {
			resProps = append(resProps, name)
		}

		assert.ElementsMatch(t, expectedProperties, resProps)
	})

	t.Run("without-all-of-and-empty-one-of-schema", func(t *testing.T) {
		schema := CreateSchemaFromString(t, `
        {
            "type": "object",
			"anyOf": [
				{}
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
			"not": {}
        }`)
		res := MergeSubSchemas(schema)
		expectedProperties := []string{"first", "second"}

		resProps := make([]string, 0)
		for name, _ := range res.Properties {
			resProps = append(resProps, name)
		}

		assert.ElementsMatch(t, expectedProperties, resProps)
	})

	t.Run("with-allof-nil-schema", func(t *testing.T) {
		schema := &openapi3.Schema{
			AllOf: openapi3.SchemaRefs{
				{
					Value: nil,
				},
			},
		}
		res := MergeSubSchemas(schema)
		assert.Equal(t, "object", res.Type)
	})

	t.Run("with-anyof-nil-schema", func(t *testing.T) {
		schema := &openapi3.Schema{
			AnyOf: openapi3.SchemaRefs{
				{
					Value: nil,
				},
			},
		}
		res := MergeSubSchemas(schema)
		assert.Equal(t, "object", res.Type)
	})

	t.Run("empty-type-defaults-in-object", func(t *testing.T) {
		schema := CreateSchemaFromString(t, `{"type": ""}`)
		res := MergeSubSchemas(schema)
		assert.Equal(t, "object", res.Type)
	})
}
