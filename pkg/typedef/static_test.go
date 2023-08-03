package typedef

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRegistryFromStaticRoutes(t *testing.T) {
	t.Run("Creates registry from JSON routes", func(t *testing.T) {
		routes := []StaticRoute{
			{
				Method:      "GET",
				Path:        "/users",
				ContentType: "application/json",
				Content:     `[{"id":"1","name":"Alice"},{"id":"2","name":"Bob"}]`,
			},
			{
				Method:      "GET",
				Path:        "/users/{id}",
				ContentType: "application/json",
				Content:     `{"id":"123","name":"John Doe","email":"john@example.com"}`,
			},
			{
				Method:      "POST",
				Path:        "/users",
				ContentType: "application/json",
				Content:     `{"id":"456","name":"Jane Smith","created":true}`,
			},
		}

		registry, err := NewRegistryFromStaticRoutes(routes)
		assert.NoError(t, err)
		assert.NotNil(t, registry)

		// Check operations
		ops := registry.Operations()
		assert.Len(t, ops, 3)

		// Check GET /users
		getUsersOp := registry.FindOperation("/users", "GET")
		assert.NotNil(t, getUsersOp)
		assert.Equal(t, "GET", getUsersOp.Method)
		assert.Equal(t, "/users", getUsersOp.Path)
		assert.Equal(t, "application/json", getUsersOp.ContentType)

		success := getUsersOp.Response.GetSuccess()
		assert.NotNil(t, success)
		assert.Equal(t, "array", success.Content.Type)
		assert.NotEmpty(t, success.Content.StaticContent)

		// Check GET /users/{id}
		getUserOp := registry.FindOperation("/users/{id}", "GET")
		assert.NotNil(t, getUserOp)
		assert.Equal(t, "object", getUserOp.Response.GetSuccess().Content.Type)
		assert.Contains(t, getUserOp.Response.GetSuccess().Content.Properties, "id")
		assert.Contains(t, getUserOp.Response.GetSuccess().Content.Properties, "name")
		assert.Contains(t, getUserOp.Response.GetSuccess().Content.Properties, "email")
	})

	t.Run("Creates registry from mixed content types", func(t *testing.T) {
		routes := []StaticRoute{
			{
				Method:      "GET",
				Path:        "/data.json",
				ContentType: "application/json",
				Content:     `{"status":"ok"}`,
			},
			{
				Method:      "GET",
				Path:        "/page.html",
				ContentType: "text/html",
				Content:     `<html><body>Hello</body></html>`,
			},
			{
				Method:      "GET",
				Path:        "/data.xml",
				ContentType: "application/xml",
				Content:     `<root><item>value</item></root>`,
			},
		}

		registry, err := NewRegistryFromStaticRoutes(routes)
		assert.NoError(t, err)
		assert.NotNil(t, registry)

		ops := registry.Operations()
		assert.Len(t, ops, 3)

		// JSON should have object schema
		jsonOp := registry.FindOperation("/data.json", "GET")
		assert.Equal(t, "object", jsonOp.Response.GetSuccess().Content.Type)

		// HTML should have string schema
		htmlOp := registry.FindOperation("/page.html", "GET")
		assert.Equal(t, "string", htmlOp.Response.GetSuccess().Content.Type)

		// XML should have string schema with xml format
		xmlOp := registry.FindOperation("/data.xml", "GET")
		assert.Equal(t, "string", xmlOp.Response.GetSuccess().Content.Type)
		assert.Equal(t, "xml", xmlOp.Response.GetSuccess().Content.Format)
	})

	t.Run("Returns error for invalid JSON", func(t *testing.T) {
		routes := []StaticRoute{
			{
				Method:      "GET",
				Path:        "/bad",
				ContentType: "application/json",
				Content:     `{invalid json}`,
			},
		}

		_, err := NewRegistryFromStaticRoutes(routes)
		assert.Error(t, err)
	})
}
