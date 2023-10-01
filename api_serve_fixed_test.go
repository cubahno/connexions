//go:build !integration

package connexions

import (
	assert2 "github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestRegisterFixedRoute(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	filePath := filepath.Join(router.Config.App.Paths.Services, "petstore", "post", "pets", "index.json")
	err = CopyFile(filepath.Join("test_fixtures", "fixed-petstore-post-pets.json"), filePath)
	assert.Nil(err)
	file, err := GetPropertiesFromFilePath(filePath, router.Config.App)
	assert.Nil(err)

	t.Run("base-case", func(t *testing.T) {
		router.Config.Services[file.ServiceName] = &ServiceConfig{}

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
		err = CopyFile(filepath.Join("test_fixtures", "fixed-petstore-post-pets.json"), filePath)
		assert.Nil(err)
		file, err := GetPropertiesFromFilePath(filePath, router.Config.App)
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
		router.Config.Services[file.ServiceName].Errors = &ServiceError{
			Codes: map[int]int{
				400: 100,
			},
			Chance: 100,
		}

		req := httptest.NewRequest(http.MethodPost, "/petstore/pets", nil)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(http.StatusBadRequest, w.Code)
		assert.Equal("Random config error", w.Body.String())
	})
}
