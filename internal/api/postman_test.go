package api

import (
	"path/filepath"
	"testing"

	"github.com/cubahno/connexions/internal/files"
	"github.com/cubahno/connexions/internal/openapi"
	"github.com/cubahno/connexions/internal/replacer"
	assert2 "github.com/stretchr/testify/assert"
)

func createPostmanRouter(t *testing.T) *Router {
	t.Helper()
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	filePath := filepath.Join(router.Config.App.Paths.ServicesOpenAPI, "petstore", "index.yml")
	err = files.CopyFile(filepath.Join(testDataPath, "document-pet-single.yml"), filePath)
	assert.Nil(err)
	file, err := openapi.GetPropertiesFromFilePath(filePath, router.Config.App)
	assert.Nil(err)

	svc := &ServiceItem{Name: "petstore"}
	svc.AddOpenAPIFile(file)

	router.services["petstore"] = svc
	rs := registerOpenAPIRoutes(file, router)
	svc.AddRoutes(rs)

	return router
}

func Test_createPostman(t *testing.T) {
	assert := assert2.New(t)
	router := createPostmanRouter(t)
	if router == nil {
		t.FailNow()
	}

	options := &postmanOptions{
		config: router.Config,
	}

	services := router.GetServices()
	res := createPostman(services, options)

	assert.Equal("Connexions", res.Info.Name)
	assert.Equal("postman-id", res.Info.Postman)
	assert.Len(res.Item, 2)

	assert.Equal("petstore", res.Item[0].Name)
	assert.Nil(res.Item[0].Request)
	assert.Nil(res.Item[0].Response)
	assert.Len(res.Item[0].Item, 1)
	assert.Equal("/pets/{id}", res.Item[0].Item[0].Name)

	assert.Equal("Generator", res.Item[1].Name)
}

func Test_createPostmanCollection(t *testing.T) {
	assert := assert2.New(t)
	router := createPostmanRouter(t)
	if router == nil {
		t.FailNow()
	}

	services := router.GetServices()
	svc := services["petstore"]
	opts := &generateResourceOptions{
		config:      router.Config.GetServiceConfig("petstore"),
		withRequest: true,
		valueReplacer: func(_ any, _ *replacer.ReplaceState) any {
			return "foo"
		},
	}
	res := createPostmanCollection(svc, opts)
	assert.NotNil(res)
	assert.Equal("petstore", res.Name)
	assert.Nil(res.Request)
	assert.Nil(res.Response)
	assert.Len(res.Item, 1)
	assert.Equal("/pets/{id}", res.Item[0].Name)
}

func Test_createPostmanGeneratorEndpoint(t *testing.T) {
	assert := assert2.New(t)
	res := createPostmanGeneratorEndpoint()

	assert.Equal("Generator", res.Name)
	assert.Nil(res.Response)
	assert.Equal("POST", res.Request.Method)
	assert.Equal(&PostmanURL{
		Raw:  "{{url}}/.services/generate",
		Host: []string{"{{url}}"},
		Path: []string{".services", "generate"},
	}, res.Request.URL)
}

func Test_getPostmanPath(t *testing.T) {
	assert := assert2.New(t)

	t.Run("single prefix with variable", func(t *testing.T) {
		path, vars := getPostmanPath("/pets-service", "/pets/{id}", "/pets/123")
		assert.Equal([]string{"pets-service", "pets", ":id"}, path)
		assert.Equal([]*PostmanKeyValue{{"id", "123"}}, vars)
	})

	t.Run("longer prefix with variables", func(t *testing.T) {
		path, vars := getPostmanPath("/pets-service/v1", "/pets/{locationName}/{id}", "/pets/home/123")
		assert.Equal([]string{"pets-service", "v1", "pets", ":locationName", ":id"}, path)
		assert.Equal([]*PostmanKeyValue{{"locationName", "home"}, {"id", "123"}}, vars)
	})

	t.Run("single prefix no variables", func(t *testing.T) {
		path, vars := getPostmanPath("/pets-service", "/pets", "/pets")
		assert.Equal([]string{"pets-service", "pets"}, path)
		assert.Len(vars, 0)
	})

	t.Run("no prefix with variable", func(t *testing.T) {
		path, vars := getPostmanPath("", "/pets/{id}", "/pets/123")
		assert.Equal([]string{"pets", ":id"}, path)
		assert.Equal([]*PostmanKeyValue{{"id", "123"}}, vars)
	})

	t.Run("root prefix with variable", func(t *testing.T) {
		path, vars := getPostmanPath("/", "/pets/{id}", "/pets/123")
		assert.Equal([]string{"pets", ":id"}, path)
		assert.Equal([]*PostmanKeyValue{{"id", "123"}}, vars)
	})
}

func Test_getPostmanBody(t *testing.T) {
	assert := assert2.New(t)

	t.Run("application/json", func(t *testing.T) {
		body := getPostmanBody("application/json", `{"foo": "bar"}`)
		assert.JSONEq(`{"foo": "bar"}`, body.Raw)
		assert.Equal("raw", body.Mode)
		assert.Equal(&PostmanBodyOptions{
			Raw: &PostmanRawBody{
				Language: "json",
			},
		}, body.Options)
	})

	t.Run("application/x-www-form-urlencoded", func(t *testing.T) {
		body := getPostmanBody("application/x-www-form-urlencoded", `{ "foo": "bar" }`)
		assert.Equal([]*PostmanKeyValue{
			{"foo", "bar"},
		}, body.Urlencoded)
		assert.Equal("urlencoded", body.Mode)
	})

	t.Run("multipart/form-data", func(t *testing.T) {
		body := getPostmanBody("multipart/form-data", `{ "foo": "bar" }`)
		assert.Equal([]*PostmanKeyValue{
			{"foo", "bar"},
		}, body.FormData)
		assert.Equal("formdata", body.Mode)
	})
}

func Test_prettyPrintPostmanJSON(t *testing.T) {
	assert := assert2.New(t)

	t.Run("empty", func(t *testing.T) {
		assert.Equal("", prettyPrintPostmanJSON(""))
	})

	t.Run("pretty", func(t *testing.T) {
		assert.JSONEq(`{
  "foo": "bar"
}`, prettyPrintPostmanJSON(`{"foo": "bar"}`))
	})
}

func Test_createPostmanEnvironment(t *testing.T) {
	assert := assert2.New(t)

	vals := []*PostmanKeyValue{
		{"url", "http://localhost:8080"},
		{"token", "123"},
	}
	res := createPostmanEnvironment("cxs[dev]", vals)

	assert.Equal("connexions-environment-cxs[dev]", res.ID)
	assert.Equal("cxs[dev]", res.Name)
	assert.Equal(vals, res.Values)
	assert.Equal("environment", res.PostmanVariableScope)
	assert.Equal("connexions", res.PostmanExportedUsing)
}
