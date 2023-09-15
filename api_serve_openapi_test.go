package connexions

import (
	assert2 "github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestRegisterOpenAPIRoutes(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	filePath := filepath.Join(router.Config.App.Paths.ServicesOpenAPI, "petstore", "document-petstore.yml")
	err = CopyFile(filepath.Join("test_fixtures", "document-petstore.yml"), filePath)
	assert.Nil(err)
	file, err := GetPropertiesFromFilePath(filePath, router.Config.App)
	assert.Nil(err)

	rs := registerOpenAPIRoutes(file, router)

	expected := RouteDescriptions{
		{
			Method: http.MethodGet,
			Path:   "/pets",
			Type:   OpenAPIRouteType,
			File:   file,
		},
		{
			Method: http.MethodPost,
			Path:   "/pets",
			Type:   OpenAPIRouteType,
			File:   file,
		},
		{
			Method: http.MethodGet,
			Path:   "/pets/{id}",
			Type:   OpenAPIRouteType,
			File:   file,
		},
		{
			Method: http.MethodDelete,
			Path:   "/pets/{id}",
			Type:   OpenAPIRouteType,
			File:   file,
		},
	}

	assert.Equal(expected, rs)
}

func TestOpenAPIHandler_serve_errors(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}
	// response validation only with Kim for now
	router.Config.App.SchemaProvider = KinOpenAPIProvider

	filePath := filepath.Join(router.Config.App.Paths.ServicesOpenAPI, "petstore", "document-petstore.yml")
	err = CopyFile(filepath.Join("test_fixtures", "document-petstore.yml"), filePath)
	assert.Nil(err)
	file, err := GetPropertiesFromFilePath(filePath, router.Config.App)
	assert.Nil(err)

	svc := &ServiceItem{
		Name: "petstore",
	}

	rs := registerOpenAPIRoutes(file, router)
	svc.AddOpenAPIFile(file)
	svc.AddRoutes(rs)
	router.Services["petstore"] = svc

	svcCfg := router.Config.Services[file.ServiceName]
	svcCfg.Contexts = nil
	svcCfg.Validate = &ServiceValidateConfig{
		Request:  true,
		Response: true,
	}

	router.Contexts = map[string]map[string]any{
		"petstore": {
			"id": 12,
			// NULL allows setting nil value explicitly and skip other replacers
			"name": NULL,
			"tag":  "#hund",
		},
	}
	router.ContextNames = []map[string]string{
		{"petstore": ""},
	}

	t.Run("method-not-allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/petstore/pets", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(http.StatusMethodNotAllowed, w.Code)
	})

	t.Run("operation-not-found", func(t *testing.T) {
		// substitute the file with a different one
		filePath = filepath.Join(router.Config.App.Paths.ServicesOpenAPI, "petstore", "alt.yml")
		err = CopyFile(filepath.Join("test_fixtures", "document-ab.yml"), filePath)
		assert.Nil(err)
		fileAlt, _ := GetPropertiesFromFilePath(filePath, router.Config.App)

		oldSpec := file.Spec
		file.Spec = fileAlt.Spec

		req := httptest.NewRequest(http.MethodGet, "/petstore/pets", nil)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(http.StatusNotFound, w.Code)

		// restore the original service
		file.Spec = oldSpec
	})

	t.Run("invalid-payload", func(t *testing.T) {
		payload := strings.NewReader(`{"name1": "test"}`)

		req := httptest.NewRequest(http.MethodPost, "/petstore/pets", payload)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		resp := UnmarshallResponse[SimpleResponse](t, w.Body)
		assert.Equal(http.StatusBadRequest, w.Code)
		assert.True(strings.Contains(resp.Message, "Invalid request: request body has an error: doesn't match schema"))
		assert.False(resp.Success)
	})

	t.Run("invalid-response", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/petstore/pets", nil)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		resp := UnmarshallResponse[SimpleResponse](t, w.Body)
		assert.Equal(http.StatusBadRequest, w.Code)
		assert.Equal("application/json", w.Header().Get("Content-Type"))
		errPrefix := `Invalid response: response body doesn't match schema: Error at "/0/name": Value is not nullable`
		assert.True(strings.HasPrefix(resp.Message, errPrefix))
	})
}

func TestOpenAPIHandler_serve(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}
	// response validation only with Kim for now
	router.Config.App.SchemaProvider = KinOpenAPIProvider

	filePath := filepath.Join(router.Config.App.Paths.ServicesOpenAPI, "petstore", "index.yml")
	err = CopyFile(filepath.Join("test_fixtures", "document-pet-single.yml"), filePath)
	assert.Nil(err)
	file, err := GetPropertiesFromFilePath(filePath, router.Config.App)
	assert.Nil(err)

	svcCfg := router.Config.Services[file.ServiceName]
	svcCfg.Contexts = nil
	svcCfg.Validate = &ServiceValidateConfig{
		Request:  true,
		Response: true,
	}

	router.Contexts = map[string]map[string]any{
		"petstore": {
			"id":   12,
			"name": "Hans",
			"tag":  "#hund",
		},
	}
	router.ContextNames = []map[string]string{
		{"petstore": ""},
	}

	svc := &ServiceItem{
		Name: "petstore",
	}

	rs := registerOpenAPIRoutes(file, router)
	svc.AddOpenAPIFile(file)
	svc.AddRoutes(rs)
	router.Services["petstore"] = svc

	t.Run("happy-path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/petstore/pets/12", nil)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		expected := map[string]any{
			"id":   float64(12),
			"name": "Hans",
			"tag":  "#hund",
		}

		resp := *UnmarshallResponse[map[string]any](t, w.Body)
		assert.Equal(http.StatusOK, w.Code)
		assert.Equal("application/json", w.Header().Get("Content-Type"))
		assert.Equal(expected, resp)
	})

	t.Run("with-cfg-error", func(t *testing.T) {
		router.Config.Services[file.ServiceName].Errors = &ServiceError{
			Codes: map[int]int{
				400: 100,
			},
			Chance: 100,
		}
		req := httptest.NewRequest(http.MethodGet, "/petstore/pets/12", nil)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(http.StatusBadRequest, w.Code)
		assert.Equal("Random config error", w.Body.String())
	})
}
