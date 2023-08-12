package api

import (
	"github.com/cubahno/xs"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestGetPropertiesFromFilePath(t *testing.T) {
	t.Run("openapi-root", func(t *testing.T) {
		filePath := xs.ServicePath + "/.openapi/index.yml"
		props := GetPropertiesFromFilePath(filePath)

		assert.Equal(t, &FileProperties{
			ServiceName: "index",
			Prefix:      "/index",
			IsOpenAPI:   true,
			FilePath:    filePath,
			FileName:    "index.yml",
			Extension:   ".yml",
			ContentType: "application/x-yaml",
		}, props)
	})

	t.Run("openapi-nested", func(t *testing.T) {
		filePath := xs.ServicePath + "/.openapi/nice/dice/rice/index.yml"
		props := GetPropertiesFromFilePath(filePath)

		assert.Equal(t, &FileProperties{
			ServiceName: "nice",
			IsOpenAPI:   true,
			Prefix:      "/nice/dice/rice",
			FilePath:    filePath,
			FileName:    "index.yml",
			Extension:   ".yml",
			ContentType: "application/x-yaml",
		}, props)
	})

	t.Run("service-index-with-method", func(t *testing.T) {
		filePath := xs.ServicePath + "/users/get/index.json"
		props := GetPropertiesFromFilePath(filePath)

		assert.Equal(t, &FileProperties{
			ServiceName: "users",
			Prefix:      "/users",
			Resource:    "/",
			Method:      http.MethodGet,
			FilePath:    filePath,
			FileName:    "index.json",
			Extension:   ".json",
			ContentType: "application/json",
		}, props)
	})

	t.Run("service-non-index-with-method", func(t *testing.T) {
		filePath := xs.ServicePath + "/users/get/index.html"
		props := GetPropertiesFromFilePath(filePath)

		assert.Equal(t, &FileProperties{
			ServiceName: "users",
			Method:      http.MethodGet,
			Prefix:      "/users",
			Resource:    "/index.html",
			FilePath:    filePath,
			FileName:    "index.html",
			Extension:   ".html",
			ContentType: "text/html; charset=utf-8",
		}, props)
	})

	t.Run("service-without-method", func(t *testing.T) {
		filePath := xs.ServicePath + "/users/all/index.xml"
		props := GetPropertiesFromFilePath(filePath)

		assert.Equal(t, &FileProperties{
			ServiceName: "users",
			Method:      http.MethodGet,
			Prefix:      "/users",
			Resource:    "/all/index.xml",
			FilePath:    filePath,
			FileName:    "index.xml",
			Extension:   ".xml",
			ContentType: "text/xml; charset=utf-8",
		}, props)
	})

	t.Run("root-service", func(t *testing.T) {
		filePath := xs.ServicePath + "/get/users.json"
		props := GetPropertiesFromFilePath(filePath)

		assert.Equal(t, &FileProperties{
			ServiceName: "",
			Method:      http.MethodGet,
			Prefix:      "",
			Resource:    "/users.json",
			FilePath:    filePath,
			FileName:    "users.json",
			Extension:   ".json",
			ContentType: "application/json",
		}, props)
	})
}

func TestComposeFileSavePath(t *testing.T) {
	type params struct {
		service   string
		method    string
		resource  string
		ext       string
		isOpenAPI bool
	}

	testCases := []struct {
		params   params
		expected string
	}{
		{
			params: params{
				service:   "test",
				method:    "get",
				resource:  "test-path",
				ext:       ".json",
				isOpenAPI: false,
			},
			expected: xs.ServicePath + "/test/get/test-path/index.json",
		},
		{
			params: params{
				resource: "/foo/bar",
			},
			expected: xs.ServicePath + "/get/foo/bar/index.json",
		},
		{
			params: params{
				service: "nice",
				method:  "patch",
			},
			expected: xs.ServicePath + "/nice/patch/index.json",
		},
		{
			params: params{
				service:   "nice",
				method:    "patch",
				resource:  "/dice/rice",
				ext:       ".yml",
				isOpenAPI: true,
			},
			expected: xs.ServicePath + "/.openapi/nice/dice/rice.yml",
		},
		{
			params: params{
				isOpenAPI: true,
			},
			expected: xs.ServicePath + "/.openapi/index",
		},
		{
			params: params{
				isOpenAPI: true,
				ext:       ".yml",
			},
			expected: xs.ServicePath + "/.openapi/index.yml",
		},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			actual := ComposeFileSavePath(
				tc.params.service, tc.params.method, tc.params.resource, tc.params.ext, tc.params.isOpenAPI)
			if actual != tc.expected {
				t.Errorf("ComposeFileSavePath(%v) - Expected: %v, Got: %v", tc.params, tc.expected, actual)
			}
		})
	}
}
