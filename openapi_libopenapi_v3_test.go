package connexions

import (
	assert2 "github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

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
			"/pets":      {"GET", "POST"},
			"/pets/{id}": {"GET", "DELETE"},
		}
		assert.ElementsMatch(expected["/pets"], res["/pets"])
		assert.ElementsMatch(expected["/pets/{id}"], res["/pets/{id}"])
	})

	t.Run("FindOperation", func(t *testing.T) {
		op := doc.FindOperation(&FindOperationOptions{"", "/pets", "GET", nil})
		assert.NotNil(op)
		libOp, ok := op.(*LibV3Operation)
		assert.True(ok)

		assert.Equal(2, len(op.GetParameters()))
		assert.Equal("findPets", libOp.OperationId)
	})

	t.Run("FindOperation-res-not-found", func(t *testing.T) {
		op := doc.FindOperation(&FindOperationOptions{"", "/pets2", "GET", nil})
		assert.Nil(op)
	})

	t.Run("FindOperation-method-not-found", func(t *testing.T) {
		op := doc.FindOperation(&FindOperationOptions{"", "/pets", "PATCH", nil})
		assert.Nil(op)
	})
}

func TestLibV3Operation(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()
	doc := CreateLibDocumentFromFile(t, filepath.Join("test_fixtures", "document-petstore.yml"))
	docWithFriends := CreateLibDocumentFromFile(t, filepath.Join("test_fixtures", "person-with-friends.yml"))

	t.Run("FindOperation-with-no-options", func(t *testing.T) {
		op := doc.FindOperation(nil)
		assert.Nil(op)
	})

	t.Run("GetParameters", func(t *testing.T) {
		op := doc.FindOperation(&FindOperationOptions{"", "/pets", "GET", nil})
		params := op.GetParameters()

		expected := OpenAPIParameters{
			{
				Name:     "limit",
				In:       ParameterInQuery,
				Required: false,
				Schema: &Schema{
					Type:   TypeInteger,
					Format: "int32",
				},
			},
			{
				Name:     "tags",
				In:       ParameterInQuery,
				Required: false,
				Schema: &Schema{
					Type: "array",
					Items: &Schema{
						Type: "string",
					},
				},
			},
		}

		AssertJSONEqual(t, expected, params)
	})

	t.Run("GetRequestBody", func(t *testing.T) {
		op := doc.FindOperation(&FindOperationOptions{"", "/pets", "POST", nil})
		body, contentType := op.GetRequestBody()

		expectedBody := &Schema{
			Type: "object",
			Properties: map[string]*Schema{
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
		AssertJSONEqual(t, expectedBody, body)
	})

	t.Run("GetRequestBody-empty", func(t *testing.T) {
		op := docWithFriends.FindOperation(&FindOperationOptions{"", "/person/{id}/find", "POST", nil})
		body, contentType := op.GetRequestBody()

		assert.Nil(body)
		assert.Equal("", contentType)
	})

	t.Run("GetRequestBody-empty-content", func(t *testing.T) {
		op := docWithFriends.FindOperation(&FindOperationOptions{"", "/person/{id}/find", "DELETE", nil})
		body, contentType := op.GetRequestBody()

		assert.Nil(body)
		assert.Equal("", contentType)
	})

	t.Run("GetRequestBody-with-xml-type", func(t *testing.T) {
		op := docWithFriends.FindOperation(&FindOperationOptions{"", "/person/{id}/find", "PATCH", nil})
		body, contentType := op.GetRequestBody()

		expectedBody := &Schema{
			Type: "object",
			Properties: map[string]*Schema{
				"id": {
					Type: TypeInteger,
				},
				"name": {
					Type: TypeString,
				},
			},
		}

		AssertJSONEqual(t, expectedBody, body)
		assert.Equal("application/xml", contentType)
	})

	t.Run("GetResponse", func(t *testing.T) {
		op := doc.FindOperation(&FindOperationOptions{"", "/pets", "GET", nil})
		res := op.GetResponse()
		content := res.Content
		contentType := res.ContentType

		var props []string
		for name, _ := range content.Items.Properties {
			props = append(props, name)
		}

		assert.Equal("application/json", contentType)
		assert.NotNil(content.Items)
		assert.Equal("array", content.Type)
		assert.Equal("object", content.Items.Type)
		assert.ElementsMatch([]string{"name", "tag", "id"}, props)
	})

	t.Run("GetResponse-first-defined-non-default", func(t *testing.T) {
		op := docWithFriends.FindOperation(&FindOperationOptions{"", "/person/{id}", "GET", nil})
		assert.NotNil(op)

		res := op.GetResponse()

		expected := &OpenAPIResponse{
			Content: &Schema{
				Type: "object",
				Properties: map[string]*Schema{
					"user": {
						Type: TypeObject,
						Properties: map[string]*Schema{
							"name": {
								Type: TypeString,
							},
						},
					},
				},
			},
			ContentType: "application/json",
			StatusCode:  404,
			Headers: OpenAPIHeaders{
				"x-header": {
					Name:     "x-header",
					In:       ParameterInHeader,
					Required: true,
					Schema: &Schema{
						Type: "string",
					},
				},
			},
		}

		AssertJSONEqual(t, expected, res)
	})

	t.Run("GetResponse-default-used", func(t *testing.T) {
		op := docWithFriends.FindOperation(&FindOperationOptions{"", "/person/{id}", "PUT", nil})
		assert.NotNil(op)

		res := op.GetResponse()

		expected := &OpenAPIResponse{
			Content: &Schema{
				Type: "object",
				Properties: map[string]*Schema{
					"code": {
						Type:   TypeInteger,
						Format: "int32",
					},
					"message": {
						Type: TypeString,
					},
				},
				Required: []string{"code", "message"},
			},
			ContentType: "application/json",
			StatusCode:  200,
		}

		AssertJSONEqual(t, expected, res)
	})

	t.Run("GetResponse-empty", func(t *testing.T) {
		op := docWithFriends.FindOperation(&FindOperationOptions{"", "/person/{id}", "PATCH", nil})
		assert.NotNil(op)

		res := op.GetResponse()

		expected := &OpenAPIResponse{}

		AssertJSONEqual(t, expected, res)
	})

	t.Run("GetResponse-non-predefined", func(t *testing.T) {
		op := docWithFriends.FindOperation(&FindOperationOptions{"", "/person/{id}/find", "GET", nil})
		assert.NotNil(op)

		res := op.GetResponse()

		expected := &OpenAPIResponse{
			Content: &Schema{
				Type: "object",
				Properties: map[string]*Schema{
					"id": {
						Type: TypeInteger,
					},
					"name": {
						Type: TypeString,
					},
				},
				Required: []string{"id"},
			},
			ContentType: "application/xml",
			StatusCode:  200,
		}

		AssertJSONEqual(t, expected, res)
	})

	t.Run("getContent-empty", func(t *testing.T) {
		op := docWithFriends.FindOperation(&FindOperationOptions{"", "/person/{id}/find", "GET", nil})
		assert.NotNil(op)
		opLib := op.(*LibV3Operation)

		res, contentType := opLib.getContent(nil)

		assert.Nil(res)
		assert.Equal("", contentType)
	})

	t.Run("WithParseConfig", func(t *testing.T) {
		op := docWithFriends.FindOperation(&FindOperationOptions{"", "/person/{id}/find", "GET", nil})
		assert.NotNil(op)
		op = op.WithParseConfig(&ParseConfig{OnlyRequired: true})

		res := op.GetResponse()
		expected := &OpenAPIResponse{
			Content: &Schema{
				Type: "object",
				Properties: map[string]*Schema{
					"id": {
						Type: TypeInteger,
					},
				},
				Required: []string{"id"},
			},
			ContentType: "application/xml",
			StatusCode:  200,
		}

		AssertJSONEqual(t, expected, res)
	})
}
