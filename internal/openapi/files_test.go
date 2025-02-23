//go:build !integration

package openapi

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/cubahno/connexions/internal/config"
	"github.com/cubahno/connexions/internal/files"
	assert2 "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileProperties(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	t.Run("IsEqual", func(t *testing.T) {
		f1 := &FileProperties{}
		f2 := &FileProperties{}
		assert.True(f1.IsEqual(f2))

		f1.ServiceName = "test"
		assert.False(f1.IsEqual(f2))
	})
}

func TestGetRequestFile(t *testing.T) {
	assert := assert2.New(t)

	t.Run("happy-path", func(t *testing.T) {
		// Create a mock Request with a file field
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		fileContents := []byte("test file contents")
		fileName := "test.txt"
		part, _ := writer.CreateFormFile("file", fileName)
		_, _ = part.Write(fileContents)
		_ = writer.Close()

		req := httptest.NewRequest("POST", "/", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		// Initialize a response recorder and handler
		rr := httptest.NewRecorder()

		// Perform the Request
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			file, err := GetRequestFile(r, "file")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// Assuming you want to validate the result
			if file != nil {
				// Perform your assertions here
				if string(file.Content) != string(fileContents) ||
					file.Filename != fileName ||
					file.Extension != filepath.Ext(fileName) ||
					file.Size != int64(len(fileContents)) {
					t.Errorf("Unexpected file attributes")
				}
			}
		})

		handler.ServeHTTP(rr, req)

		// Check the response status code
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
	})

	t.Run("no-file", func(t *testing.T) {
		body := &bytes.Buffer{}
		req := httptest.NewRequest("POST", "/", body)
		res, err := GetRequestFile(req, "file")

		assert.Nil(res)
		assert.Nil(err)
	})
}

func TestGetPropertiesFromFilePath(t *testing.T) {
	appCfg := config.NewDefaultAppConfig("/app")
	paths := appCfg.Paths

	t.Parallel()

	t.Run("service-root-direct", func(t *testing.T) {
		filePath := paths.Services + "/root/users.html"
		props, _ := GetPropertiesFromFilePath(filePath, appCfg)

		AssertJSONEqual(t, &FileProperties{
			ServiceName: "",
			Prefix:      "",
			Resource:    "/users.html",
			Method:      http.MethodGet,
			FilePath:    filePath,
			FileName:    "users.html",
			Extension:   ".html",
			ContentType: "text/html",
		}, props)
	})

	// result should be as above, in the `root`
	t.Run("service-direct", func(t *testing.T) {
		filePath := paths.Services + "/users.html"
		props, _ := GetPropertiesFromFilePath(filePath, appCfg)

		AssertJSONEqual(t, &FileProperties{
			ServiceName: "",
			Prefix:      "",
			Resource:    "/users.html",
			Method:      http.MethodGet,
			FilePath:    filePath,
			FileName:    "users.html",
			Extension:   ".html",
			ContentType: "text/html",
		}, props)
	})

	// result should be as above, in the `root`
	t.Run("service-direct-index", func(t *testing.T) {
		filePath := paths.Services + "/index.json"
		props, _ := GetPropertiesFromFilePath(filePath, appCfg)

		AssertJSONEqual(t, &FileProperties{
			ServiceName: "",
			Prefix:      "",
			Resource:    "/",
			Method:      http.MethodGet,
			FilePath:    filePath,
			FileName:    "index.json",
			Extension:   ".json",
			ContentType: "application/json",
		}, props)
	})

	t.Run("service-root-with-method", func(t *testing.T) {
		filePath := paths.Services + "/root/patch/users.html"
		props, _ := GetPropertiesFromFilePath(filePath, appCfg)

		AssertJSONEqual(t, &FileProperties{
			ServiceName: "",
			Method:      http.MethodPatch,
			Prefix:      "",
			Resource:    "/users.html",
			FilePath:    filePath,
			FileName:    "users.html",
			Extension:   ".html",
			ContentType: "text/html",
		}, props)
	})

	t.Run("service-non-root-will-have-method-as-service", func(t *testing.T) {
		filePath := paths.Services + "/patch/users.html"
		props, _ := GetPropertiesFromFilePath(filePath, appCfg)

		AssertJSONEqual(t, &FileProperties{
			ServiceName: "patch",
			Method:      http.MethodGet,
			Prefix:      "/patch",
			Resource:    "/users.html",
			FilePath:    filePath,
			FileName:    "users.html",
			Extension:   ".html",
			ContentType: "text/html",
		}, props)
	})

	t.Run("service-without-method", func(t *testing.T) {
		filePath := paths.Services + "/users/all/index.xml"
		props, _ := GetPropertiesFromFilePath(filePath, appCfg)

		AssertJSONEqual(t, &FileProperties{
			ServiceName: "users",
			Method:      http.MethodGet,
			Prefix:      "/users",
			Resource:    "/all",
			FilePath:    filePath,
			FileName:    "index.xml",
			Extension:   ".xml",
			ContentType: "text/xml",
		}, props)
	})

	t.Run("service-with-index-file", func(t *testing.T) {
		filePath := paths.Services + "/users/patch/id/{userId}/index.json"
		props, _ := GetPropertiesFromFilePath(filePath, appCfg)

		AssertJSONEqual(t, &FileProperties{
			ServiceName: "users",
			Method:      http.MethodPatch,
			Prefix:      "/users",
			Resource:    "/id/{userId}",
			FilePath:    filePath,
			FileName:    "index.json",
			Extension:   ".json",
			ContentType: "application/json",
		}, props)
	})
}

func TestGetPropertiesFromOpenAPIFile(t *testing.T) {
	assert := require.New(t)
	appCfg := config.NewDefaultAppConfig("/app")
	paths := appCfg.Paths

	t.Parallel()

	t.Run("root-missing-file", func(t *testing.T) {
		filePath := paths.ServicesOpenAPI + "/index.yml"
		props, err := GetPropertiesFromFilePath(filePath, appCfg)
		assert.Nil(props)
		assert.Error(err)
	})

	t.Run("nested-with-index-name", func(t *testing.T) {
		baseDir := t.TempDir()
		appConfig := config.NewDefaultAppConfig(baseDir)
		ps := appConfig.Paths

		dir := filepath.Join(ps.ServicesOpenAPI, "nice", "dice", "rice")
		_ = os.MkdirAll(dir, 0755)

		filePath := filepath.Join(dir, "index.yml")
		contents, _ := os.ReadFile(filepath.Join(testDataPath, "document-petstore.yml"))
		err := files.SaveFile(filePath, contents)
		assert.NoError(err)

		props, err := GetPropertiesFromFilePath(filePath, appConfig)

		assert.NoError(err)
		assert.NotNil(props.Spec)

		expectedProps := &FileProperties{
			ServiceName: "nice",
			IsOpenAPI:   true,
			Prefix:      "/nice/dice/rice",
			FilePath:    filePath,
			FileName:    "index.yml",
			Extension:   ".yml",
		}
		AssertJSONEqual(t, expectedProps, props)
	})

	t.Run("nested-with-any-name", func(t *testing.T) {
		baseDir := t.TempDir()
		appConfig := config.NewDefaultAppConfig(baseDir)
		ps := appConfig.Paths

		dir := filepath.Join(ps.ServicesOpenAPI, "nice", "dice")
		_ = os.MkdirAll(dir, 0755)

		filePath := filepath.Join(dir, "rice.yml")
		contents, _ := os.ReadFile(filepath.Join(testDataPath, "document-petstore.yml"))
		err := files.SaveFile(filePath, contents)
		assert.NoError(err)

		props, err := GetPropertiesFromFilePath(filePath, appConfig)

		assert.NoError(err)
		assert.NotNil(props.Spec)

		expectedProps := &FileProperties{
			ServiceName: "nice",
			IsOpenAPI:   true,
			Prefix:      "/nice/dice",
			FilePath:    filePath,
			FileName:    "rice.yml",
			Extension:   ".yml",
		}
		AssertJSONEqual(t, expectedProps, props)
	})
}
