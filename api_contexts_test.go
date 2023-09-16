package connexions

import (
    assert2 "github.com/stretchr/testify/assert"
    "net/http"
    "net/http/httptest"
    "path/filepath"
    "testing"
)

func TestCreateContextRoutes_Disabled(t *testing.T) {
    assert := assert2.New(t)
    t.Parallel()
    router, _ := SetupApp(t.TempDir())
    router.Config.App.DisableUI = true

    _ = CreateContextRoutes(router)
    assert.Equal(0, len(router.Mux.Routes()))
}

func TestContextHandler(t *testing.T) {
    assert := assert2.New(t)

    router, err := SetupApp(t.TempDir())
    if err != nil {
        t.Errorf("Error setting up app: %v", err)
        t.FailNow()
    }

    err = CreateContextRoutes(router)
    assert.Nil(err)

    t.Run("list", func(t *testing.T) {
        router.Contexts["bob"] = map[string]any{
            "name": "Bob",
        }
        router.Contexts["alice"] = map[string]any{
            "name": "Alice",
        }

        req := httptest.NewRequest("GET", "/.contexts", nil)
        w := httptest.NewRecorder()
        router.ServeHTTP(w, req)

        assert.Equal(http.StatusOK, w.Code)
        assert.Equal("application/json", w.Header().Get("Content-Type"))
        resp := UnmarshallResponse[ContextListResponse](t, w.Body)
        assert.Equal(&ContextListResponse{
            Items: []string{"alice", "bob"},
        }, resp)

        delete(router.Contexts, "bob")
        delete(router.Contexts, "alice")
    })

    t.Run("details", func(t *testing.T) {
        router.Contexts["bob"] = map[string]any{}
        err = CopyFile(filepath.Join("test_fixtures", "context-petstore.yml"), filepath.Join(router.Config.App.Paths.Contexts, "bob.yml"))
        if err != nil {
            t.Errorf("Error copying file: %v", err)
            t.FailNow()
        }
        req := httptest.NewRequest("GET", "/.contexts/bob", nil)
        w := httptest.NewRecorder()
        router.ServeHTTP(w, req)

        assert.Equal(http.StatusOK, w.Code)
        assert.Equal("application/x-yaml", w.Header().Get("Content-Type"))
        assert.Greater(w.Body.Len(), 0)

        delete(router.Contexts, "bob")
    })

    t.Run("details-not-found", func(t *testing.T) {
        req := httptest.NewRequest("GET", "/.contexts/bob", nil)
        w := httptest.NewRecorder()
        router.ServeHTTP(w, req)

        assert.Equal(http.StatusNotFound, w.Code)
        assert.Equal("application/json", w.Header().Get("Content-Type"))
        resp := UnmarshallResponse[SimpleResponse](t, w.Body)
        assert.Equal(&SimpleResponse{
            Success: false,
            Message: "Context not found",
        }, resp)
    })
}
