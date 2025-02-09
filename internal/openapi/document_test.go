//--go:build !integration

package openapi

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/cubahno/connexions/internal/config"
	"github.com/cubahno/connexions/internal/types"
	"github.com/getkin/kin-openapi/openapi3"
	assert2 "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func jsonPair(expected, actual any) (string, string) {
	expectedJSON, _ := json.Marshal(expected)
	actualJSON, _ := json.Marshal(actual)

	return string(expectedJSON), string(actualJSON)
}

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
	assert := require.New(t)

	doc, err := NewDocumentFromFile(filepath.Join(testDataPath, "document-petstore.yml"))
	assert.NoError(err)

	t.Run("GetVersion", func(t *testing.T) {
		assert.Equal("3.0.0", doc.GetVersion())
	})

	t.Run("GetResources", func(t *testing.T) {
		res := doc.GetResources()
		expected := map[string][]string{
			"/pets":      {"GET", "POST"},
			"/pets/{id}": {"GET", "DELETE"},
		}
		assert.ElementsMatch(expected["/pets"], res["/pets"])
		assert.ElementsMatch(expected["/pets/{id}"], res["/pets/{id}"])
	})

	t.Run("GetSecurity", func(t *testing.T) {
		res := doc.GetSecurity()
		expected := SecurityComponents{
			"HTTPBearer": &SecurityComponent{
				Type:   AuthTypeHTTP,
				Scheme: AuthSchemeBearer,
				In:     AuthLocationHeader,
			},
		}
		assert.Equal(expected, res)
	})

	t.Run("FindOperation", func(t *testing.T) {
		op := doc.FindOperation(&OperationDescription{Resource: "/pets", Method: "GET"})
		assert.NotNil(op)
	})

	t.Run("FindOperation-res-not-found", func(t *testing.T) {
		op := doc.FindOperation(&OperationDescription{Resource: "/pets2", Method: "GET"})
		assert.Nil(op)
	})

	t.Run("FindOperation-method-not-found", func(t *testing.T) {
		op := doc.FindOperation(&OperationDescription{Resource: "/pets", Method: "PATCH"})
		assert.Nil(op)
	})
}

func TestOperation(t *testing.T) {
	assert := assert2.New(t)
	doc, err := NewDocumentFromFile(filepath.Join(testDataPath, "document-petstore.yml"))
	assert.NoError(err)

	docWithFriendsPath := filepath.Join(testDataPath, "document-person-with-friends.yml")

	t.Run("FindOperation-with-no-options", func(t *testing.T) {
		op := doc.FindOperation(nil)
		assert.Nil(op)
	})

	t.Run("getParameters", func(t *testing.T) {
		op := doc.FindOperation(&OperationDescription{Resource: "/pets", Method: "GET"})
		req := op.GetRequest(nil)
		params := req.Parameters

		expected := Parameters{
			{
				Name:     "limit",
				In:       ParameterInQuery,
				Required: false,
				Schema: &types.Schema{
					Type:   types.TypeInteger,
					Format: "int32",
				},
			},
			{
				Name:     "tags",
				In:       ParameterInQuery,
				Required: false,
				Schema: &types.Schema{
					Type: "array",
					Items: &types.Schema{
						Type: "string",
					},
				},
			},
		}

		a, b := jsonPair(expected, params)
		if a != b {
			t.Errorf("expected / actual: \n%s\n%s", a, b)
		}
	})

	t.Run("getRequestBody", func(t *testing.T) {
		op := doc.FindOperation(&OperationDescription{Resource: "/pets", Method: "POST"})
		req := op.GetRequest(nil)
		payload := req.Body
		body := payload.Schema
		contentType := payload.Type

		expectedBody := &types.Schema{
			Type: "object",
			Properties: map[string]*types.Schema{
				"name": {
					Type: "string",
				},
				"tag": {
					Type: "string",
				},
			},
			Required: []string{"name"},
		}
		assert.NotNil(body)
		assert.Equal("application/json", contentType)

		a, b := jsonPair(expectedBody, body)
		if a != b {
			t.Errorf("expected / actual: \n%s\n%s", a, b)
		}
	})

	t.Run("GetResponse", func(t *testing.T) {
		op := doc.FindOperation(&OperationDescription{Resource: "/pets", Method: "GET"})
		res := op.GetResponse()
		content := res.Content
		contentType := res.ContentType

		var props []string
		for name := range content.Items.Properties {
			props = append(props, name)
		}

		assert.Equal("application/json", contentType)
		assert.NotNil(content.Items)
		assert.Equal("array", content.Type)
		assert.Equal("object", content.Items.Type)
		assert.ElementsMatch([]string{"name", "tag", "id"}, props)
	})

	t.Run("getRequestBody-empty", func(t *testing.T) {
		docWithFriends, err := NewDocumentFromFile(docWithFriendsPath)
		assert.NoError(err)

		op := docWithFriends.FindOperation(&OperationDescription{Resource: "/person/{id}/find", Method: "POST"})
		req := op.GetRequest(nil)
		payload := req.Body
		body := payload.Schema
		contentType := payload.Type

		assert.Nil(body)
		assert.Equal("", contentType)
	})

	t.Run("getRequestBody-empty-content", func(t *testing.T) {
		docWithFriends, err := NewDocumentFromFile(docWithFriendsPath)
		assert.NoError(err)

		op := docWithFriends.FindOperation(&OperationDescription{Resource: "/person/{id}/find", Method: "DELETE"})
		req := op.GetRequest(nil)
		payload := req.Body
		body := payload.Schema
		contentType := payload.Type

		assert.Nil(body)
		assert.Equal("", contentType)
	})

	t.Run("getRequestBody-with-xml-type", func(t *testing.T) {
		docWithFriends, err := NewDocumentFromFile(docWithFriendsPath)
		assert.NoError(err)

		op := docWithFriends.FindOperation(&OperationDescription{Resource: "/person/{id}/find", Method: "PATCH"})
		req := op.GetRequest(nil)
		payload := req.Body
		body := payload.Schema
		contentType := payload.Type

		expectedBody := &types.Schema{
			Type: "object",
			Properties: map[string]*types.Schema{
				"id": {
					Type: types.TypeInteger,
				},
				"name": {
					Type: types.TypeString,
				},
			},
		}

		assert.Equal("application/xml", contentType)
		a, b := jsonPair(expectedBody, body)
		if a != b {
			t.Errorf("expected / actual: \n%s\n%s", a, b)
		}
	})

	t.Run("GetResponse-first-defined-non-default", func(t *testing.T) {
		docWithFriends, err := NewDocumentFromFile(docWithFriendsPath)
		assert.NoError(err)

		op := docWithFriends.FindOperation(&OperationDescription{Resource: "/person/{id}", Method: "GET"})
		assert.NotNil(op)

		res := op.GetResponse()

		expected := &Response{
			Content: &types.Schema{
				Type: "object",
				Properties: map[string]*types.Schema{
					"user": {
						Type: types.TypeObject,
						Properties: map[string]*types.Schema{
							"name": {
								Type: types.TypeString,
							},
						},
					},
				},
			},
			ContentType: "application/json",
			StatusCode:  404,
			Headers: Headers{
				"x-header": {
					Name:     "x-header",
					In:       ParameterInHeader,
					Required: true,
					Schema: &types.Schema{
						Type: "string",
					},
				},
				"y-header": {
					Name: "y-header",
					In:   ParameterInHeader,
				},
			},
		}

		a, b := jsonPair(expected, res)
		if a != b {
			t.Errorf("expected / actual: \n%s\n%s", a, b)
		}
	})

	t.Run("GetResponse-default-used", func(t *testing.T) {
		docWithFriends, err := NewDocumentFromFile(docWithFriendsPath)
		assert.NoError(err)

		op := docWithFriends.FindOperation(&OperationDescription{Resource: "/person/{id}", Method: "PUT"})
		assert.NotNil(op)

		res := op.GetResponse()

		expected := &Response{
			Content: &types.Schema{
				Type: "object",
				Properties: map[string]*types.Schema{
					"code": {
						Type:   types.TypeInteger,
						Format: "int32",
					},
					"message": {
						Type: types.TypeString,
					},
				},
				Required: []string{"code", "message"},
			},
			ContentType: "application/json",
			StatusCode:  200,
		}

		a, b := jsonPair(expected, res)
		if a != b {
			t.Errorf("expected / actual: \n%s\n%s", a, b)
		}
	})

	t.Run("GetResponse-empty", func(t *testing.T) {
		docWithFriends, err := NewDocumentFromFile(docWithFriendsPath)
		assert.NoError(err)

		op := docWithFriends.FindOperation(&OperationDescription{Resource: "/person/{id}", Method: "PATCH"})
		assert.NotNil(op)

		res := op.GetResponse()
		expected := &Response{StatusCode: http.StatusOK}

		a, b := jsonPair(expected, res)
		if a != b {
			t.Errorf("expected / actual: \n%s\n%s", a, b)
		}
	})

	t.Run("GetResponse-non-predefined", func(t *testing.T) {
		docWithFriends, err := NewDocumentFromFile(docWithFriendsPath)
		assert.NoError(err)

		op := docWithFriends.FindOperation(&OperationDescription{Resource: "/person/{id}/find", Method: "GET"})
		assert.NotNil(op)

		res := op.GetResponse()

		expected := &Response{
			Content: &types.Schema{
				Type: "object",
				Properties: map[string]*types.Schema{
					"id": {
						Type: types.TypeInteger,
					},
					"name": {
						Type: types.TypeString,
					},
				},
				Required: []string{"id"},
			},
			ContentType: "application/xml",
			StatusCode:  200,
		}

		a, b := jsonPair(expected, res)
		if a != b {
			t.Errorf("expected / actual: \n%s\n%s", a, b)
		}
	})

	t.Run("WithParseConfig", func(t *testing.T) {
		docWithFriends, err := NewDocumentFromFile(docWithFriendsPath)
		assert.NoError(err)

		op := docWithFriends.FindOperation(&OperationDescription{Resource: "/person/{id}/find", Method: "GET"})
		assert.NotNil(op)
		op = op.WithParseConfig(&config.ParseConfig{OnlyRequired: true})

		res := op.GetResponse()
		expected := &Response{
			Content: &types.Schema{
				Type: "object",
				Properties: map[string]*types.Schema{
					"id": {
						Type: types.TypeInteger,
					},
				},
				Required: []string{"id"},
			},
			ContentType: "application/xml",
			StatusCode:  200,
		}

		a, b := jsonPair(expected, res)
		if a != b {
			t.Errorf("expected / actual: \n%s\n%s", a, b)
		}
	})

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
		expected := Parameters{
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
		securityComponents := SecurityComponents{
			"HTTPBearer": &SecurityComponent{
				Type:   AuthTypeHTTP,
				Scheme: AuthSchemeBearer,
			},
			"APIKey": &SecurityComponent{
				Type: AuthTypeApiKey,
				In:   AuthLocationQuery,
				Name: "x-api-key",
			},
		}

		expected := Parameters{
			{
				Name: "code",
				In:   "header",
			},
			{
				Name:     "authorization",
				In:       "header",
				Required: true,
				Schema: &types.Schema{
					Type:   "string",
					Format: "bearer",
				},
			},
			{
				Name:     "x-api-key",
				In:       "query",
				Required: true,
				Schema: &types.Schema{
					Type: "string",
				},
			},
		}

		res := operation.getParameters(securityComponents)

		assert.ElementsMatch(expected, res)
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
		assert.ElementsMatch(expected, res)
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
		expected := Response{
			StatusCode: http.StatusOK,
			Headers: Headers{
				"x-rate-limit-limit": {
					Name: "x-rate-limit-limit",
					In:   ParameterInHeader,
					Schema: &types.Schema{
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

	getSchema := func(t *testing.T, fileName, componentID string, parseConfig *config.ParseConfig) *types.Schema {
		t.Helper()
		kinDoc, err := NewDocumentFromFile(filepath.Join(testDataPath, fileName))
		assert.Nil(err)
		doc := kinDoc
		kinSchema := doc.Components.Schemas[componentID].Value
		assert.NotNil(kinSchema)

		return NewSchemaFromKin(kinSchema, parseConfig)
	}

	t.Run("WithParseConfig-max-recursive-levels", func(t *testing.T) {
		res := getSchema(t, "document-circular-ucr.yml", "OrgByIdResponseWrapperModel",
			&config.ParseConfig{MaxRecursionLevels: 1})

		typs := []any{
			"Department",
			"Division",
			"Organization",
		}

		example := []any{
			map[string]any{
				"type":        "string",
				"code":        "string",
				"description": "string",
				"isActive":    true,
			},
			map[string]any{
				"type":        "string",
				"code":        "string",
				"description": "string",
				"isActive":    true,
			},
		}

		assert.NotNil(res)
		assert.Equal(types.TypeObject, res.Type)

		success := res.Properties["success"]
		assert.Equal(&types.Schema{Type: types.TypeBoolean}, success)

		response := res.Properties["response"]
		assert.Equal(types.TypeObject, response.Type)

		typ := response.Properties["type"]
		assert.Equal(&types.Schema{
			Type: types.TypeString,
			Enum: typs,
		}, typ)

		parent := response.Properties["parent"]
		assert.Equal(&types.Schema{
			Type: types.TypeObject,
			Properties: map[string]*types.Schema{
				"parent": nil,
				"type": {
					Type: types.TypeString,
					Enum: typs,
				},
				"children": {
					Type:    types.TypeArray,
					Items:   &types.Schema{Type: types.TypeString},
					Example: example,
				},
			},
		}, parent)

		children := response.Properties["children"]
		assert.Equal(types.TypeArray, children.Type)
		childrenItems := children.Items
		assert.Equal(types.TypeObject, childrenItems.Type)

		childrenParent := childrenItems.Properties["parent"]
		assert.Nil(childrenParent)

		childrenChildren := childrenItems.Properties["children"]
		assert.Equal(&types.Schema{
			Type:    types.TypeArray,
			Items:   &types.Schema{Type: types.TypeString},
			Example: example,
		}, childrenChildren)

		childrenType := childrenItems.Properties["type"]
		assert.Equal(&types.Schema{
			Type: types.TypeString,
			Enum: typs,
		}, childrenType)

		childrenExample := children.Example
		assert.Equal(example, childrenExample)
	})

	t.Run("circular-with-additional-properties", func(t *testing.T) {
		res := getSchema(t, "document-connexions.yml", "Map",
			&config.ParseConfig{MaxRecursionLevels: 0})

		expected := &types.Schema{
			Type: types.TypeObject,
			Properties: map[string]*types.Schema{
				"extra-1": {
					Type: types.TypeObject,
				},
				"extra-2": {
					Type: types.TypeObject,
				},
				"extra-3": {
					Type: types.TypeObject,
				},
			},
		}
		assert.NotNil(res)
		assert.Equal(expected, res)
	})

	t.Run("Not-parsed", func(t *testing.T) {
		res := getSchema(t, "document-person-with-friends.yml", "StateWithoutAbbr", nil)
		expected := &types.Schema{
			Type: types.TypeObject,
			Properties: map[string]*types.Schema{
				"name": {Type: types.TypeString},
				"abbr": {Type: types.TypeString},
			},
			Not: &types.Schema{
				Type: types.TypeObject,
				Properties: map[string]*types.Schema{
					"abbr": {Type: types.TypeString},
				},
			},
		}
		assert.NotNil(res)
		assert.Equal(expected, res)
	})

	t.Run("nested-all-of", func(t *testing.T) {
		target := openapi3.NewSchema()
		CreateSchemaFromYAMLFile(t, filepath.Join(testDataPath, "schema-with-nested-all-of.yml"), target)

		res := NewSchemaFromKin(target, nil)
		assert.NotNil(res)

		expected := &types.Schema{
			Type: types.TypeObject,
			Properties: map[string]*types.Schema{
				"name":   {Type: types.TypeString},
				"age":    {Type: types.TypeInteger},
				"league": {Type: types.TypeString},
				"rating": {Type: types.TypeInteger},
				"tag":    {Type: types.TypeString},
			},
		}
		a, b := GetJSONPair(expected, res)
		if a != b {
			t.Errorf("expected / actual: \n%s\n%s", a, b)
		}
	})

	t.Run("with-parse-config-applied", func(t *testing.T) {
		target := openapi3.NewSchema()
		CreateSchemaFromYAMLFile(t, filepath.Join(testDataPath, "document-petstore.yml"), target)

		res := newSchemaFromKin(target, &config.ParseConfig{MaxLevels: 1}, nil, []string{"user", "id"})
		assert.Nil(res)
	})

	t.Run("with-circular-detected", func(t *testing.T) {
		target := openapi3.NewSchema()
		CreateSchemaFromYAMLFile(t, filepath.Join(testDataPath, "document-petstore.yml"), target)

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
				{Value: &openapi3.Schema{Type: &openapi3.Types{types.TypeString}}},
			},
			Not: &openapi3.SchemaRef{
				Value: &openapi3.Schema{Enum: []any{"doggie"}},
			},
		}

		schema, ref := mergeKinSubSchemas(target)
		assert.Nil(schema.OneOf)
		assert.Equal("", ref)
		assert.True(schema.Type.Is(types.TypeString))
	})

	t.Run("implied-array-type", func(t *testing.T) {
		target := &openapi3.Schema{
			OneOf: openapi3.SchemaRefs{
				{Value: &openapi3.Schema{
					Items: &openapi3.SchemaRef{
						Value: &openapi3.Schema{Type: &openapi3.Types{types.TypeString}},
					},
				}},
			},
		}

		schema, ref := mergeKinSubSchemas(target)
		assert.Nil(schema.OneOf)
		assert.Equal("", ref)
		assert.True(schema.Type.Is(types.TypeArray))
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
		assert.True(schema.Type.Is(types.TypeObject))
	})

	t.Run("inferred-from-enum", func(t *testing.T) {
		target := &openapi3.Schema{
			Enum: []any{1, 2, 3},
		}

		schema, _ := mergeKinSubSchemas(target)
		assert.True(schema.Type.Is(types.TypeInteger))
	})
}

func TestPickSchemaProxy(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	t.Run("happy-path", func(t *testing.T) {
		items := []*openapi3.SchemaRef{
			nil,
			{Value: &openapi3.Schema{Type: &openapi3.Types{types.TypeString}}},
			{Value: &openapi3.Schema{Type: &openapi3.Types{types.TypeInteger}}},
		}
		res := pickKinSchemaProxy(items)
		assert.Equal(items[1], res)
	})

	t.Run("prefer-reference", func(t *testing.T) {
		items := []*openapi3.SchemaRef{
			nil,
			{Value: &openapi3.Schema{Type: &openapi3.Types{types.TypeString}}},
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
			Type: &openapi3.Types{types.TypeString},
		}
		assert.Equal(expected, res)
	})

	t.Run("inlined-object", func(t *testing.T) {
		source := openapi3.AdditionalProperties{
			Schema: &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type: &openapi3.Types{types.TypeObject},
					Properties: map[string]*openapi3.SchemaRef{
						"name": {
							Value: &openapi3.Schema{
								Type: &openapi3.Types{types.TypeString},
							},
						},
						"age": {
							Value: &openapi3.Schema{
								Type: &openapi3.Types{types.TypeInteger},
							},
						},
					},
				},
			},
		}

		expected := &openapi3.Schema{
			Type: &openapi3.Types{types.TypeObject},
			Properties: map[string]*openapi3.SchemaRef{
				"name": {
					Value: &openapi3.Schema{Type: &openapi3.Types{types.TypeString}},
				},
				"age": {
					Value: &openapi3.Schema{Type: &openapi3.Types{types.TypeInteger}},
				},
			},
		}

		res := getKinAdditionalProperties(source)

		assert.Equal(expected, res)
	})
}
