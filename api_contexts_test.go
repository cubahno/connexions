//go:build !integration

package connexions

import (
	assert2 "github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateContextRoutes_Disabled(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()
	router, _ := SetupApp(t.TempDir())
	router.Config.App.DisableUI = true

	_ = createContextRoutes(router)
	assert.Equal(0, len(router.Mux.Routes()))
}

func TestContextHandler_list(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	err = createContextRoutes(router)
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
}

func TestContextHandler_details(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	err = createContextRoutes(router)
	assert.Nil(err)

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

func TestContextHandler_delete(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	err = createContextRoutes(router)
	assert.Nil(err)

	t.Run("happy-path", func(t *testing.T) {
		router.Contexts["bob"] = map[string]any{}
		bobPath := filepath.Join(router.Config.App.Paths.Contexts, "bob.yml")
		err = CopyFile(filepath.Join("test_fixtures", "context-petstore.yml"), bobPath)
		if err != nil {
			t.Errorf("Error copying file: %v", err)
			t.FailNow()
		}
		req := httptest.NewRequest("DELETE", "/.contexts/bob", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(http.StatusOK, w.Code)
		assert.Equal("application/json", w.Header().Get("Content-Type"))

		resp := UnmarshallResponse[SimpleResponse](t, w.Body)
		assert.Equal(&SimpleResponse{
			Success: true,
			Message: "Context deleted!",
		}, resp)

		_, err = os.ReadFile(bobPath)
		assert.NotNil(err)

		_, ok := router.Contexts["bob"]
		assert.False(ok)
	})

	t.Run("ctx-not-found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/.contexts/bob", nil)
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

func TestContextHandler_save(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	err = createContextRoutes(router)
	assert.Nil(err)

	t.Run("empty-contents", func(t *testing.T) {
		writer, payload := CreateTestMapFormReader(map[string]string{
			"name": "bob",
		})
		req := httptest.NewRequest(http.MethodPut, "/.contexts", payload)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		resp := UnmarshallResponse[SimpleResponse](t, w.Body)
		assert.Equal(&SimpleResponse{
			Success: true,
			Message: "Context saved",
		}, resp)

		var contexts []string
		for name := range router.Contexts {
			contexts = append(contexts, name)
		}
		assert.Equal([]string{"bob"}, contexts)
		_, err = os.ReadFile(filepath.Join(router.Config.App.Paths.Contexts, "bob.yml"))
		assert.Nil(err)
	})

	t.Run("not-empty-contents", func(t *testing.T) {
		yamlCont := `name: Bob
age: 30
address: 123 Main St
`
		writer, payload := CreateTestMapFormReader(map[string]string{
			"name":    "bob",
			"content": yamlCont,
		})
		req := httptest.NewRequest(http.MethodPut, "/.contexts", payload)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		resp := UnmarshallResponse[SimpleResponse](t, w.Body)
		assert.Equal(&SimpleResponse{
			Success: true,
			Message: "Context saved",
		}, resp)

		var contexts []string
		for name := range router.Contexts {
			contexts = append(contexts, name)
		}
		assert.Equal([]string{"bob"}, contexts)
		c, err := os.ReadFile(filepath.Join(router.Config.App.Paths.Contexts, "bob.yml"))
		assert.Nil(err)
		assert.Equal(yamlCont, string(c))
	})
}

func TestContextHandler_save_errors(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	err = createContextRoutes(router)
	assert.Nil(err)

	t.Run("missing-name", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/.contexts", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		resp := UnmarshallResponse[SimpleResponse](t, w.Body)
		assert.Equal(&SimpleResponse{
			Success: false,
			Message: "Name is required",
		}, resp)
	})

	t.Run("invalid-name", func(t *testing.T) {
		writer, payload := CreateTestMapFormReader(map[string]string{
			"name": "bob.yml",
		})
		req := httptest.NewRequest(http.MethodPut, "/.contexts", payload)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		resp := UnmarshallResponse[SimpleResponse](t, w.Body)
		assert.Equal(http.StatusBadRequest, w.Code)
		assert.Equal(&SimpleResponse{
			Success: false,
			Message: "Invalid name: must be alpha-numeric, _, - and not exceed 20 chars",
		}, resp)
	})

	t.Run("invalid-yaml", func(t *testing.T) {
		writer, payload := CreateTestMapFormReader(map[string]string{
			"name":    "bob",
			"content": "1",
		})
		req := httptest.NewRequest(http.MethodPut, "/.contexts", payload)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		resp := UnmarshallResponse[SimpleResponse](t, w.Body)
		assert.Equal(http.StatusBadRequest, w.Code)
		assert.False(resp.Success)
		assert.Contains(resp.Message, "Invalid context: yaml: unmarshal errors:")
	})

	t.Run("folder-permissions-error", func(t *testing.T) {
		writer, payload := CreateTestMapFormReader(map[string]string{
			"name": "bob",
		})
		req := httptest.NewRequest(http.MethodPut, "/.contexts", payload)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		_ = os.Chmod(router.Config.App.Paths.Contexts, 0000)
		defer func() {
			_ = os.Chmod(router.Config.App.Paths.Contexts, 0755)
		}()

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		resp := UnmarshallResponse[SimpleResponse](t, w.Body)
		assert.Equal(http.StatusInternalServerError, w.Code)
		assert.False(resp.Success)
		assert.Contains(resp.Message, "error creating file")
	})
}
