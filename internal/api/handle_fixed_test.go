//go:build !integration

package api

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/cubahno/connexions/internal/config"
	"github.com/cubahno/connexions/internal/files"
	"github.com/cubahno/connexions/internal/openapi"
	assert2 "github.com/stretchr/testify/assert"
)

func TestRegisterFixedRoute(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	filePath := filepath.Join(router.Config.App.Paths.Services, "petstore", "post", "pets", "index.json")
	err = files.CopyFile(filepath.Join(testDataPath, "fixed-petstore-post-pets.json"), filePath)
	assert.Nil(err)
	file, err := openapi.GetPropertiesFromFilePath(filePath, router.Config.App)
	assert.Nil(err)

	t.Run("base-case", func(t *testing.T) {
		router.Config.Services[file.ServiceName] = config.NewServiceConfig()

		rs := registerFixedRoute(file, router)

		expected := &RouteDescription{
			Method:      http.MethodPost,
			Path:        "/pets",
			Type:        FixedRouteType,
			ContentType: "application/json",
			File:        file,
		}

		assert.Equal(expected, rs)

		req := httptest.NewRequest(http.MethodPost, "/petstore/pets", nil)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		expectedResp := map[string]any{
			"id":   float64(1),
			"name": "Bulbasaur",
			"tag":  "beedrill",
		}

		resp := *UnmarshallResponse[map[string]any](t, w.Body)
		assert.Equal(http.StatusOK, w.Code)
		assert.Equal("application/json", w.Header().Get("Content-Type"))
		assert.Equal(expectedResp, resp)
	})

	t.Run("empty-resource", func(t *testing.T) {
		filePath := filepath.Join(router.Config.App.Paths.Services, "index.json")
		err = files.CopyFile(filepath.Join(testDataPath, "fixed-petstore-post-pets.json"), filePath)
		assert.Nil(err)
		file, err := openapi.GetPropertiesFromFilePath(filePath, router.Config.App)
		assert.Nil(err)

		rs := registerFixedRoute(file, router)

		expected := &RouteDescription{
			Method:      http.MethodGet,
			Path:        "/",
			Type:        FixedRouteType,
			ContentType: "application/json",
			File:        file,
		}

		assert.Equal(expected, rs)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		expectedResp := map[string]any{
			"id":   float64(1),
			"name": "Bulbasaur",
			"tag":  "beedrill",
		}

		resp := *UnmarshallResponse[map[string]any](t, w.Body)
		assert.Equal(http.StatusOK, w.Code)
		assert.Equal("application/json", w.Header().Get("Content-Type"))
		assert.Equal(expectedResp, resp)
	})

	t.Run("with-cfg-error", func(t *testing.T) {
		router.Config.Services[file.ServiceName] = config.NewServiceConfig()
		router.Config.Services[file.ServiceName].Errors = map[string]int{
			"p100": 400,
		}
		_ = registerFixedRoute(file, router)

		req := httptest.NewRequest(http.MethodPost, "/petstore/pets", nil)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(http.StatusBadRequest, w.Code)
		assert.Equal("configured service error: 400", w.Body.String())
	})
}
