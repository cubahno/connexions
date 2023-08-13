package xs

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestGetPropertiesFromFilePath(t *testing.T) {
	t.Run("openapi-root", func(t *testing.T) {
		t.SkipNow()
		filePath := ServicePath + "/.openapi/index.yml"
		props, _ := GetPropertiesFromFilePath(filePath)

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
		t.SkipNow()
		filePath := ServicePath + "/.openapi/nice/dice/rice/index.yml"
		props, _ := GetPropertiesFromFilePath(filePath)

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
		filePath := ServicePath + "/.root/users.html"
		props, _ := GetPropertiesFromFilePath(filePath)

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
		filePath := ServicePath + "/users.html"
		props, _ := GetPropertiesFromFilePath(filePath)

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
		filePath := ServicePath + "/.root/patch/users.html"
		props, _ := GetPropertiesFromFilePath(filePath)

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
		filePath := ServicePath + "/patch/users.html"
		props, _ := GetPropertiesFromFilePath(filePath)

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
		filePath := ServicePath + "/users/all/index.xml"
		props, _ := GetPropertiesFromFilePath(filePath)

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

	t.Run("service-with-index-file", func(t *testing.T) {
		filePath := ServicePath + "/users/patch/id/{userId}/index.json"
		props, _ := GetPropertiesFromFilePath(filePath)

		assert.Equal(t, &FileProperties{
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
			expected: ServicePath + "/.root/get/foo.html",
		},
		{
			description: "root patch file",
			params: params{
				method:   "patch",
				resource: "/foo.html",
			},
			expected: ServicePath + "/.root/patch/foo.html",
		},
		{
			params: params{
				service:   "test",
				method:    "get",
				resource:  "test-path",
				ext:       ".json",
				isOpenAPI: false,
			},
			expected: ServicePath + "/test/get/test-path/index.json",
		},
		{
			params: params{
				resource: "/foo/bar",
			},
			expected: ServicePath + "/foo/get/bar/index.txt",
		},
		{
			params: params{
				service: "nice",
				method:  "patch",
			},
			expected: ServicePath + "/nice/patch/index.txt",
		},
		{
			params: params{
				service:   "nice",
				method:    "patch",
				resource:  "/dice/rice",
				ext:       ".yml",
				isOpenAPI: true,
			},
			expected: ServicePath + "/.openapi/nice/dice/rice.yml",
		},
		{
			params: params{
				isOpenAPI: true,
			},
			expected: ServicePath + "/.openapi/index",
		},
		{
			params: params{
				isOpenAPI: true,
				ext:       ".yml",
			},
			expected: ServicePath + "/.openapi/index.yml",
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

	t.Run("openapi-with-prefix", func(t *testing.T) {
		res := ComposeFileSavePath("", "", "petstore", ".yml", true)
		assert.Equal(t, ServicePath+"/.openapi/petstore/index.yml", res)
	})
}
