package connexions

import (
	"bytes"
	"fmt"
	assert2 "github.com/stretchr/testify/assert"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCreateHomeRoutes(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	err = createHomeRoutes(router)
	assert.Nil(err)
	_ = CopyDirectory(filepath.Join("resources", "ui"), router.Config.App.Paths.UI)

	t.Run("home", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/.ui/", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(http.StatusOK, w.Code)
		assert.Equal("text/html; charset=utf-8", w.Header().Get("Content-Type"))
		// template parsed
		assert.Contains(w.Body.String(), `serviceUrl: "\/.services"`)
	})

	t.Run("static", func(t *testing.T) {
		_ = os.WriteFile(filepath.Join(router.Config.App.Paths.UI, "test.yml"), []byte("app:"), 0644)
		req := httptest.NewRequest("GET", "/.ui/test.yml", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(http.StatusOK, w.Code)
		assert.Equal("application/x-yaml", w.Header().Get("Content-Type"))
		assert.Equal("app:", w.Body.String())
	})

	t.Run("docs", func(t *testing.T) {
		// site is the source dir for the docs
		siteDir := filepath.Join(router.Config.App.Paths.Base, "site")
		_ = os.Mkdir(siteDir, 0755)
		_ = os.WriteFile(filepath.Join(siteDir, "hi.md"), []byte("Hallo!"), 0644)

		req := httptest.NewRequest("GET", "/.ui/docs/hi.md", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(http.StatusOK, w.Code)
		assert.Equal("text/markdown; charset=utf-8", w.Header().Get("Content-Type"))
		assert.Equal("Hallo!", w.Body.String())
	})

	t.Run("export", func(t *testing.T) {
		_ = CopyFile(filepath.Join("test_fixtures", "document-petstore.yml"), filepath.Join(router.Config.App.Paths.ServicesOpenAPI, "petstore", "index.yml"))
		_ = CopyFile(filepath.Join("test_fixtures", "context-petstore.yml"), filepath.Join(router.Config.App.Paths.Contexts, "petstore.yml"))

		// empty dirs ignored
		_ = os.MkdirAll(filepath.Join(router.Config.App.Paths.Services, "petstore", "get", "pets"), 0755)
		// unreadable files ignored
		unreadablePath := filepath.Join(router.Config.App.Paths.Services, "petstore.index.json")
		_ = os.WriteFile(unreadablePath, []byte(`{"k":"v"}`), 0000)

		req := httptest.NewRequest("GET", "/.ui/export", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(http.StatusOK, w.Code)
		assert.Equal("application/zip", w.Header().Get("Content-Type"))
		assert.Equal(
			fmt.Sprintf("attachment; filename=connexions-%s.zip", time.Now().Format("2006-01-02")),
			w.Header().Get("Content-Disposition"))

		_ = os.Chmod(unreadablePath, 0777)
	})
}

func TestCreateHomeRoutes_import(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	err = createHomeRoutes(router)
	assert.Nil(err)

	createReqBody := func(fieldName, fileName string, contents []byte) (*multipart.Writer, *bytes.Buffer) {
		var requestBody bytes.Buffer
		writer := multipart.NewWriter(&requestBody)

		file, err := writer.CreateFormFile(fieldName, fileName)
		if err != nil {
			fmt.Println("Error creating form file:", err)
			return nil, nil
		}

		file.Write(contents)
		writer.Close()

		return writer, &requestBody
	}

	t.Run("happy-path", func(t *testing.T) {
		payload := CreateTestZip(map[string]string{
			filepath.Join("services", "pets", "get", "index.txt"):  "Pets content.",
			filepath.Join("services", "bets", "post", "index.txt"): "Bets content.",
			filepath.Join("contexts", "pets.yml"):                  "",
			filepath.Join("contexts", "common.yml"):                "id: 123",
		})
		writer, reqBody := createReqBody("file", "connexions.zip", payload.Bytes())
		req := httptest.NewRequest(http.MethodPost, "/.ui/import", reqBody)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		resp := UnmarshallResponse[SimpleResponse](t, w.Body)
		assert.Equal(http.StatusOK, w.Code)
		assert.Equal("application/json", w.Header().Get("Content-Type"))
		assert.Equal("Imported successfully!", resp.Message)

		var services []string
		for _, service := range router.Services {
			services = append(services, service.Name)
		}

		var contexts []string
		for contextName, _ := range router.Contexts {
			contexts = append(contexts, contextName)
		}

		assert.ElementsMatch([]string{"pets", "bets"}, services)
		assert.ElementsMatch([]string{"pets", "common"}, contexts)
	})

	t.Run("missing-content-type", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/.ui/import", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		resp := UnmarshallResponse[SimpleResponse](t, w.Body)
		assert.Equal(http.StatusBadRequest, w.Code)
		assert.Equal("application/json", w.Header().Get("Content-Type"))
		assert.Equal("request Content-Type isn't multipart/form-data", resp.Message)
	})

	t.Run("missing-write-permissions", func(t *testing.T) {
		payload := CreateTestZip(map[string]string{
			filepath.Join("services", "pets.txt"): "Pets content.",
		})
		writer, reqBody := createReqBody("file", "test.zip", payload.Bytes())
		req := httptest.NewRequest(http.MethodPost, "/.ui/import", reqBody)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		_ = os.Chmod(router.Config.App.Paths.Resources, 0400)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		resp := UnmarshallResponse[SimpleResponse](t, w.Body)
		assert.Equal(http.StatusInternalServerError, w.Code)
		assert.Equal("application/json", w.Header().Get("Content-Type"))
		assert.True(strings.Contains(resp.Message, "permission denied"))

		_ = os.Chmod(router.Config.App.Paths.Resources, 0755)
	})

	t.Run("missing-write-permissions-untakeable", func(t *testing.T) {
		payload := CreateTestZip(nil)
		writer, reqBody := createReqBody("file", "test.zip", payload.Bytes())
		req := httptest.NewRequest(http.MethodPost, "/.ui/import", reqBody)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		_ = os.Chmod(router.Config.App.Paths.Resources, 0400)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		resp := UnmarshallResponse[SimpleResponse](t, w.Body)
		assert.Equal(http.StatusInternalServerError, w.Code)
		assert.Equal("application/json", w.Header().Get("Content-Type"))
		assert.True(strings.Contains(resp.Message, "permission denied"))

		_ = os.Chmod(router.Config.App.Paths.Resources, 0755)
	})
}

func TestCreateHomeRoutes_export_errors(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	err = createHomeRoutes(router)
	assert.Nil(err)

	os.Chmod(router.Config.App.Paths.Resources, 0000)

	req := httptest.NewRequest("GET", "/.ui/export", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(http.StatusInternalServerError, w.Code)

	os.Chmod(router.Config.App.Paths.Resources, 0777)
}

func TestCreateHomeRoutes_disabled(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	router.Config.App.DisableUI = true
	err = createHomeRoutes(router)
	assert.Nil(err)
	_ = CopyDirectory(filepath.Join("resources", "ui"), router.Config.App.Paths.UI)

	req := httptest.NewRequest("GET", "/.ui/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(http.StatusNotFound, w.Code)
}

func TestCreateHomeRoutes_errors(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	err = createHomeRoutes(router)
	assert.Nil(err)

	t.Run("missing-template", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/.ui/", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// no template
		assert.Equal(http.StatusInternalServerError, w.Code)
	})

	t.Run("template-not-found", func(t *testing.T) {
		_ = CopyDirectory(filepath.Join("resources", "ui"), router.Config.App.Paths.UI)
		_ = os.Remove(filepath.Join(router.Config.App.Paths.UI, "home.html"))

		req := httptest.NewRequest("GET", "/.ui/", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(http.StatusOK, w.Code)
		assert.Contains(w.Body.String(), `serviceUrl: "\/.services"`)
	})

	t.Run("invalid-template", func(t *testing.T) {
		_ = CopyDirectory(filepath.Join("resources", "ui"), router.Config.App.Paths.UI)
		indexPath := filepath.Join(router.Config.App.Paths.UI, "index.html")
		tpl, _ := os.ReadFile(indexPath)
		tplContents := strings.Replace(string(tpl), "{{.AppConfig", "{{.AppConfig2", 1)
		_ = os.WriteFile(indexPath, []byte(tplContents), 0644)

		req := httptest.NewRequest("GET", "/.ui/", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(http.StatusInternalServerError, w.Code)
	})
}
