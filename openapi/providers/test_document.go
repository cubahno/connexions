package providers

import (
	"encoding/json"
	"github.com/cubahno/connexions/config"
	"github.com/cubahno/connexions/openapi"
	"github.com/stretchr/testify/require"
	"net/http"
	"path/filepath"
	"testing"
)

func jsonPair(expected, actual any) (string, string) {
	expectedJSON, _ := json.Marshal(expected)
	actualJSON, _ := json.Marshal(actual)

	return string(expectedJSON), string(actualJSON)
}

type DocumentTestCase struct {
	DocFactory func(filePath string) (openapi.Document, error)
}

func (tc *DocumentTestCase) Run(t *testing.T) {
	t.Helper()
	assert := require.New(t)
	testData := filepath.Join("..", "..", "..", "testdata")

	doc, err := tc.DocFactory(filepath.Join(testData, "document-petstore.yml"))
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

	t.Run("FindOperation", func(t *testing.T) {
		op := doc.FindOperation(&openapi.OperationDescription{Resource: "/pets", Method: "GET"})
		assert.NotNil(op)

		assert.Equal(2, len(op.GetParameters()))
	})

	t.Run("FindOperation-res-not-found", func(t *testing.T) {
		op := doc.FindOperation(&openapi.OperationDescription{Resource: "/pets2", Method: "GET"})
		assert.Nil(op)
	})

	t.Run("FindOperation-method-not-found", func(t *testing.T) {
		op := doc.FindOperation(&openapi.OperationDescription{Resource: "/pets", Method: "PATCH"})
		assert.Nil(op)
	})
}

type OperationTestCase struct {
	DocFactory func(filePath string) (openapi.Document, error)
}

func (tc *OperationTestCase) Run(t *testing.T) {
	t.Helper()
	assert := require.New(t)
	testData := filepath.Join("..", "..", "..", "testdata")

	doc, err := tc.DocFactory(filepath.Join(testData, "document-petstore.yml"))
	assert.NoError(err)

	docWithFriendsPath := filepath.Join(testData, "document-person-with-friends.yml")

	t.Run("FindOperation-with-no-options", func(t *testing.T) {
		op := doc.FindOperation(nil)
		assert.Nil(op)
	})

	t.Run("GetParameters", func(t *testing.T) {
		op := doc.FindOperation(&openapi.OperationDescription{Resource: "/pets", Method: "GET"})
		params := op.GetParameters()

		expected := openapi.Parameters{
			{
				Name:     "limit",
				In:       openapi.ParameterInQuery,
				Required: false,
				Schema: &openapi.Schema{
					Type:   openapi.TypeInteger,
					Format: "int32",
				},
			},
			{
				Name:     "tags",
				In:       openapi.ParameterInQuery,
				Required: false,
				Schema: &openapi.Schema{
					Type: "array",
					Items: &openapi.Schema{
						Type: "string",
					},
				},
			},
		}

		a, b := jsonPair(expected, params)
		if a != b {
			t.Errorf("doc %s: \nexpected / actual: \n%s\n%s", doc.Provider(), a, b)
		}
	})

	t.Run("GetRequestBody", func(t *testing.T) {
		op := doc.FindOperation(&openapi.OperationDescription{Resource: "/pets", Method: "POST"})
		body, contentType := op.GetRequestBody()

		expectedBody := &openapi.Schema{
			Type: "object",
			Properties: map[string]*openapi.Schema{
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
			t.Errorf("doc %s: \nexpected / actual: \n%s\n%s", doc.Provider(), a, b)
		}
	})

	t.Run("GetResponse", func(t *testing.T) {
		op := doc.FindOperation(&openapi.OperationDescription{Resource: "/pets", Method: "GET"})
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

	t.Run("GetRequestBody-empty", func(t *testing.T) {
		docWithFriends, err := tc.DocFactory(docWithFriendsPath)
		assert.NoError(err)

		op := docWithFriends.FindOperation(&openapi.OperationDescription{Resource: "/person/{id}/find", Method: "POST"})
		body, contentType := op.GetRequestBody()

		assert.Nil(body)
		assert.Equal("", contentType)
	})

	t.Run("GetRequestBody-empty-content", func(t *testing.T) {
		docWithFriends, err := tc.DocFactory(docWithFriendsPath)
		assert.NoError(err)

		op := docWithFriends.FindOperation(&openapi.OperationDescription{Resource: "/person/{id}/find", Method: "DELETE"})
		body, contentType := op.GetRequestBody()

		assert.Nil(body)
		assert.Equal("", contentType)
	})

	t.Run("GetRequestBody-with-xml-type", func(t *testing.T) {
		docWithFriends, err := tc.DocFactory(docWithFriendsPath)
		assert.NoError(err)

		op := docWithFriends.FindOperation(&openapi.OperationDescription{Resource: "/person/{id}/find", Method: "PATCH"})
		body, contentType := op.GetRequestBody()

		expectedBody := &openapi.Schema{
			Type: "object",
			Properties: map[string]*openapi.Schema{
				"id": {
					Type: openapi.TypeInteger,
				},
				"name": {
					Type: openapi.TypeString,
				},
			},
		}

		assert.Equal("application/xml", contentType)
		a, b := jsonPair(expectedBody, body)
		if a != b {
			t.Errorf("doc %s: \nexpected / actual: \n%s\n%s", docWithFriends.Provider(), a, b)
		}
	})

	t.Run("GetResponse-first-defined-non-default", func(t *testing.T) {
		docWithFriends, err := tc.DocFactory(docWithFriendsPath)
		assert.NoError(err)

		op := docWithFriends.FindOperation(&openapi.OperationDescription{Resource: "/person/{id}", Method: "GET"})
		assert.NotNil(op)

		res := op.GetResponse()

		expected := &openapi.Response{
			Content: &openapi.Schema{
				Type: "object",
				Properties: map[string]*openapi.Schema{
					"user": {
						Type: openapi.TypeObject,
						Properties: map[string]*openapi.Schema{
							"name": {
								Type: openapi.TypeString,
							},
						},
					},
				},
			},
			ContentType: "application/json",
			StatusCode:  404,
			Headers: openapi.Headers{
				"x-header": {
					Name:     "x-header",
					In:       openapi.ParameterInHeader,
					Required: true,
					Schema: &openapi.Schema{
						Type: "string",
					},
				},
				"y-header": {
					Name: "y-header",
					In:   openapi.ParameterInHeader,
				},
			},
		}

		a, b := jsonPair(expected, res)
		if a != b {
			t.Errorf("doc %s: \nexpected / actual: \n%s\n%s", docWithFriends.Provider(), a, b)
		}
	})

	t.Run("GetResponse-default-used", func(t *testing.T) {
		docWithFriends, err := tc.DocFactory(docWithFriendsPath)
		assert.NoError(err)

		op := docWithFriends.FindOperation(&openapi.OperationDescription{Resource: "/person/{id}", Method: "PUT"})
		assert.NotNil(op)

		res := op.GetResponse()

		expected := &openapi.Response{
			Content: &openapi.Schema{
				Type: "object",
				Properties: map[string]*openapi.Schema{
					"code": {
						Type:   openapi.TypeInteger,
						Format: "int32",
					},
					"message": {
						Type: openapi.TypeString,
					},
				},
				Required: []string{"code", "message"},
			},
			ContentType: "application/json",
			StatusCode:  200,
		}

		a, b := jsonPair(expected, res)
		if a != b {
			t.Errorf("doc %s: \nexpected / actual: \n%s\n%s", docWithFriends.Provider(), a, b)
		}
	})

	t.Run("GetResponse-empty", func(t *testing.T) {
		docWithFriends, err := tc.DocFactory(docWithFriendsPath)
		assert.NoError(err)

		op := docWithFriends.FindOperation(&openapi.OperationDescription{Resource: "/person/{id}", Method: "PATCH"})
		assert.NotNil(op)

		res := op.GetResponse()
		expected := &openapi.Response{StatusCode: http.StatusOK}

		a, b := jsonPair(expected, res)
		if a != b {
			t.Errorf("doc %s: \nexpected / actual: \n%s\n%s", docWithFriends.Provider(), a, b)
		}
	})

	t.Run("GetResponse-non-predefined", func(t *testing.T) {
		docWithFriends, err := tc.DocFactory(docWithFriendsPath)
		assert.NoError(err)

		op := docWithFriends.FindOperation(&openapi.OperationDescription{Resource: "/person/{id}/find", Method: "GET"})
		assert.NotNil(op)

		res := op.GetResponse()

		expected := &openapi.Response{
			Content: &openapi.Schema{
				Type: "object",
				Properties: map[string]*openapi.Schema{
					"id": {
						Type: openapi.TypeInteger,
					},
					"name": {
						Type: openapi.TypeString,
					},
				},
				Required: []string{"id"},
			},
			ContentType: "application/xml",
			StatusCode:  200,
		}

		a, b := jsonPair(expected, res)
		if a != b {
			t.Errorf("doc %s: \nexpected / actual: \n%s\n%s", docWithFriends.Provider(), a, b)
		}
	})

	t.Run("WithParseConfig", func(t *testing.T) {
		docWithFriends, err := tc.DocFactory(docWithFriendsPath)
		assert.NoError(err)

		op := docWithFriends.FindOperation(&openapi.OperationDescription{Resource: "/person/{id}/find", Method: "GET"})
		assert.NotNil(op)
		op = op.WithParseConfig(&config.ParseConfig{OnlyRequired: true})

		res := op.GetResponse()
		expected := &openapi.Response{
			Content: &openapi.Schema{
				Type: "object",
				Properties: map[string]*openapi.Schema{
					"id": {
						Type: openapi.TypeInteger,
					},
				},
				Required: []string{"id"},
			},
			ContentType: "application/xml",
			StatusCode:  200,
		}

		a, b := jsonPair(expected, res)
		if a != b {
			t.Errorf("doc %s: \nexpected / actual: \n%s\n%s", docWithFriends.Provider(), a, b)
		}
	})
}

// NewSchemaTestSuite tests the NewSchema functions from providers, so we don't have to repeat the same tests for each
type NewSchemaTestSuite struct {
	SchemaFactory func(t *testing.T, fileName, componentID string, parseConfig *config.ParseConfig) *openapi.Schema
}

func (tc *NewSchemaTestSuite) Run(t *testing.T) {
	t.Helper()
	assert := require.New(t)

	t.Run("WithParseConfig-max-recursive-levels", func(t *testing.T) {
		res := tc.SchemaFactory(t, "document-circular-ucr.yml", "OrgByIdResponseWrapperModel",
			&config.ParseConfig{MaxRecursionLevels: 1})

		types := []any{
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
		assert.Equal(openapi.TypeObject, res.Type)

		success := res.Properties["success"]
		assert.Equal(&openapi.Schema{Type: openapi.TypeBoolean}, success)

		response := res.Properties["response"]
		assert.Equal(openapi.TypeObject, response.Type)

		typ := response.Properties["type"]
		assert.Equal(&openapi.Schema{
			Type: openapi.TypeString,
			Enum: types,
		}, typ)

		parent := response.Properties["parent"]
		assert.Equal(&openapi.Schema{
			Type: openapi.TypeObject,
			Properties: map[string]*openapi.Schema{
				"parent": nil,
				"type": {
					Type: openapi.TypeString,
					Enum: types,
				},
				"children": {
					Type:    openapi.TypeArray,
					Items:   &openapi.Schema{Type: openapi.TypeString},
					Example: example,
				},
			},
		}, parent)

		children := response.Properties["children"]
		assert.Equal(openapi.TypeArray, children.Type)
		childrenItems := children.Items
		assert.Equal(openapi.TypeObject, childrenItems.Type)

		childrenParent := childrenItems.Properties["parent"]
		assert.Nil(childrenParent)

		childrenChildren := childrenItems.Properties["children"]
		assert.Equal(&openapi.Schema{
			Type:    openapi.TypeArray,
			Items:   &openapi.Schema{Type: openapi.TypeString},
			Example: example,
		}, childrenChildren)

		childrenType := childrenItems.Properties["type"]
		assert.Equal(&openapi.Schema{
			Type: openapi.TypeString,
			Enum: types,
		}, childrenType)

		childrenExample := children.Example
		assert.Equal(example, childrenExample)
	})

	t.Run("circular-with-additional-properties", func(t *testing.T) {
		res := tc.SchemaFactory(t, "document-connexions.yml", "Map",
			&config.ParseConfig{MaxRecursionLevels: 0})

		expected := &openapi.Schema{
			Type: openapi.TypeObject,
			Properties: map[string]*openapi.Schema{
				"extra-1": {
					Type: openapi.TypeObject,
				},
				"extra-2": {
					Type: openapi.TypeObject,
				},
				"extra-3": {
					Type: openapi.TypeObject,
				},
			},
		}
		assert.NotNil(res)
		assert.Equal(expected, res)
	})

	t.Run("Not-parsed", func(t *testing.T) {
		res := tc.SchemaFactory(t, "document-person-with-friends.yml", "StateWithoutAbbr", nil)
		expected := &openapi.Schema{
			Type: openapi.TypeObject,
			Properties: map[string]*openapi.Schema{
				"name": {Type: openapi.TypeString},
				"abbr": {Type: openapi.TypeString},
			},
			Not: &openapi.Schema{
				Type: openapi.TypeObject,
				Properties: map[string]*openapi.Schema{
					"abbr": {Type: openapi.TypeString},
				},
			},
		}
		assert.NotNil(res)
		assert.Equal(expected, res)
	})
}
