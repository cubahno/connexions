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
			ServiceName: "",
			Prefix:      "",
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

	t.Run("service-root-direct", func(t *testing.T) {
		filePath := xs.ServicePath + "/.root/users.html"
		props := GetPropertiesFromFilePath(filePath)

		assert.Equal(t, &FileProperties{
			ServiceName: "",
			Prefix:      "",
			Resource:    "/users.html",
			Method:      http.MethodGet,
			FilePath:    filePath,
			FileName:    "users.html",
			Extension:   ".html",
			ContentType: "text/html; charset=utf-8",
		}, props)
	})

	// result should as above, in the .root
	t.Run("service-direct", func(t *testing.T) {
		filePath := xs.ServicePath + "/users.html"
		props := GetPropertiesFromFilePath(filePath)

		assert.Equal(t, &FileProperties{
			ServiceName: "",
			Prefix:      "",
			Resource:    "/users.html",
			Method:      http.MethodGet,
			FilePath:    filePath,
			FileName:    "users.html",
			Extension:   ".html",
			ContentType: "text/html; charset=utf-8",
		}, props)
	})

	t.Run("service-root-with-method", func(t *testing.T) {
		filePath := xs.ServicePath + "/.root/patch/users.html"
		props := GetPropertiesFromFilePath(filePath)

		assert.Equal(t, &FileProperties{
			ServiceName: "",
			Method:      http.MethodPatch,
			Prefix:      "",
			Resource:    "/users.html",
			FilePath:    filePath,
			FileName:    "users.html",
			Extension:   ".html",
			ContentType: "text/html; charset=utf-8",
		}, props)
	})

	t.Run("service-non-root-will-have-method-as-service", func(t *testing.T) {
		filePath := xs.ServicePath + "/patch/users.html"
		props := GetPropertiesFromFilePath(filePath)

		assert.Equal(t, &FileProperties{
			ServiceName: "patch",
			Method:      http.MethodGet,
			Prefix:      "/patch",
			Resource:    "/users.html",
			FilePath:    filePath,
			FileName:    "users.html",
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
		description string
		params      params
		expected    string
	}{
		{
			description: "root file",
			params: params{
				resource: "/foo.html",
			},
			expected: xs.ServicePath + "/.root/get/foo.html",
		},
		{
			description: "root patch file",
			params: params{
				method:   "patch",
				resource: "/foo.html",
			},
			expected: xs.ServicePath + "/.root/patch/foo.html",
		},
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
			expected: xs.ServicePath + "/get/foo/bar/index.txt",
		},
		{
			params: params{
				service: "nice",
				method:  "patch",
			},
			expected: xs.ServicePath + "/nice/patch/index.txt",
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
				t.Errorf("ComposeFileSavePath(%v): %v - Expected: %v, Got: %v",
					tc.params, tc.description, tc.expected, actual)
			}
		})
	}
}
