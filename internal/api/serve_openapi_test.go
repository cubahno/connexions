//go:build !integration

package api

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cubahno/connexions/internal/config"
	"github.com/cubahno/connexions/internal/openapi"
	"github.com/cubahno/connexions/internal/replacer"
	"github.com/cubahno/connexions/internal/types"
	assert2 "github.com/stretchr/testify/assert"
)

func TestRegisterOpenAPIRoutes(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	filePath := filepath.Join(router.Config.App.Paths.ServicesOpenAPI, "petstore", "document-petstore.yml")
	err = types.CopyFile(filepath.Join(testDataPath, "document-petstore.yml"), filePath)
	assert.Nil(err)
	file, err := openapi.GetPropertiesFromFilePath(filePath, router.Config.App)
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
	filePath := filepath.Join(router.Config.App.Paths.ServicesOpenAPI, "petstore", "document-petstore.yml")
	err = types.CopyFile(filepath.Join(testDataPath, "document-petstore.yml"), filePath)
	assert.Nil(err)
	file, err := openapi.GetPropertiesFromFilePath(filePath, router.Config.App)
	assert.Nil(err)

	svc := &ServiceItem{
		Name: "petstore",
	}

	router.SetContexts(
		map[string]map[string]any{
			"petstore": {
				"id": 12,
				// NULL allows setting nil value explicitly and skip other replacers
				"name": replacer.NULL,
				"tag":  "#hund",
			},
		},
		[]map[string]string{
			{"petstore": ""},
		},
	)

	svc.AddOpenAPIFile(file)
	router.services["petstore"] = svc
	router.Config.Services[file.ServiceName] = &config.ServiceConfig{}

	svcCfg := router.Config.Services[file.ServiceName]
	svcCfg.Contexts = nil
	svcCfg.Validate = &config.ServiceValidateConfig{
		Request:  true,
		Response: true,
	}
	svcCfg.Cache = &config.ServiceCacheConfig{
		Schema: false,
	}

	rs := registerOpenAPIRoutes(file, router)
	svc.AddRoutes(rs)

	t.Run("method-not-allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/petstore/pets", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(http.StatusMethodNotAllowed, w.Code)
	})

	t.Run("operation-not-found", func(t *testing.T) {
		// substitute the file with a different one
		filePath = filepath.Join(router.Config.App.Paths.ServicesOpenAPI, "petstore", "alt.yml")
		err = types.CopyFile(filepath.Join(testDataPath, "document-ab.yml"), filePath)
		assert.Nil(err)
		fileAlt, _ := openapi.GetPropertiesFromFilePath(filePath, router.Config.App)

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
		assert.True(strings.Contains(resp.Message, `property "name" is missing`))
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
		errPrefix := `response body doesn't match schema: Error at "/0/name": property "name" is missing`
		assert.Contains(resp.Message, errPrefix)
	})
}

func TestOpenAPIHandler_serve(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}
	filePath := filepath.Join(router.Config.App.Paths.ServicesOpenAPI, "petstore", "index.yml")
	err = types.CopyFile(filepath.Join(testDataPath, "document-pet-single.yml"), filePath)
	assert.Nil(err)
	file, err := openapi.GetPropertiesFromFilePath(filePath, router.Config.App)
	assert.Nil(err)

	router.Config.Services[file.ServiceName] = config.NewServiceConfig()
	svcCfg := router.Config.Services[file.ServiceName]
	svcCfg.Contexts = nil
	svcCfg.Validate = &config.ServiceValidateConfig{
		Request:  true,
		Response: true,
	}

	router.SetContexts(
		map[string]map[string]any{
			"petstore": {
				"id":   12,
				"name": "Hans",
				"tag":  "#hund",
			},
		},
		[]map[string]string{
			{"petstore": ""},
		},
	)

	svc := &ServiceItem{
		Name: "petstore",
	}
	svc.AddOpenAPIFile(file)
	router.services["petstore"] = svc

	rs := registerOpenAPIRoutes(file, router)
	svc.AddRoutes(rs)

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
		router.Config.Services[file.ServiceName] = config.NewServiceConfig()
		router.Config.Services[file.ServiceName].Errors = map[string]int{
			"p100": 400,
		}
		rs := registerOpenAPIRoutes(file, router)
		svc.AddRoutes(rs)

		req := httptest.NewRequest(http.MethodGet, "/petstore/pets/12", nil)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(http.StatusBadRequest, w.Code)
		assert.Equal("configured service error: 400", w.Body.String())
	})
}
