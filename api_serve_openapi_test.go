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

    t.Run("method-not-allowed", func(t *testing.T) {
        req := httptest.NewRequest(http.MethodOptions, "/petstore/pets", nil)
        w := httptest.NewRecorder()
        router.ServeHTTP(w, req)

        assert.Equal(http.StatusMethodNotAllowed, w.Code)
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
}

func TestOpenAPIHandler_serve(t *testing.T) {

}
