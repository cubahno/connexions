package api

import (
	"github.com/cubahno/xs"
	"testing"
)

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
			actual := composeFileSavePath(
				tc.params.service, tc.params.method, tc.params.resource, tc.params.ext, tc.params.isOpenAPI)
			if actual != tc.expected {
				t.Errorf("composeFileSavePath(%v) - Expected: %v, Got: %v", tc.params, tc.expected, actual)
			}
		})
	}
}
