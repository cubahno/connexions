//go:build !integration

package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cubahno/connexions/internal/config"
	"github.com/cubahno/connexions/internal/files"
	"github.com/cubahno/connexions/internal/openapi"
	"github.com/cubahno/connexions/internal/testhelpers"
	assert2 "github.com/stretchr/testify/assert"
)

func TestCreateServiceRoutes_Disabled(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()
	router, _ := SetupApp(t.TempDir())
	router.Config.App.DisableUI = true

	_ = createServiceRoutes(router)
	assert.Equal(0, len(router.Mux.Routes()))
}

func TestServiceItem_AddOpenAPIFile(t *testing.T) {
	assert := assert2.New(t)

	svc := &ServiceItem{}
	fileProps := &openapi.FileProperties{
		FileName: "index.yaml",
	}
	svc.AddOpenAPIFile(fileProps)
	svc.AddOpenAPIFile(fileProps)
	assert.Equal([]*openapi.FileProperties{fileProps}, svc.OpenAPIFiles)
}

func TestServiceItem_AddRoutes(t *testing.T) {
	assert := assert2.New(t)

	svc := &ServiceItem{}
	route1 := &RouteDescription{Type: FixedRouteType}
	route2 := &RouteDescription{Type: OpenAPIRouteType}
	route3 := &RouteDescription{Type: FixedRouteType}

	svc.AddRoutes(RouteDescriptions{route1})
	svc.AddRoutes(RouteDescriptions{route2})
	svc.AddRoutes(RouteDescriptions{route3})

	assert.Equal(RouteDescriptions{route1, route2, route3}, svc.Routes)
	assert.False(route1.Overwrites)
	assert.False(route2.Overwrites)
	assert.True(route3.Overwrites)
}

func TestRouteDescriptions_Sort(t *testing.T) {
	assert := assert2.New(t)

	routes := RouteDescriptions{
		{Path: "/a", Method: http.MethodDelete},
		{Path: "/a", Method: http.MethodGet},
		{Path: "/a", Method: http.MethodPost},
		{Path: "/a", Method: http.MethodOptions},
		{Path: "/a", Method: http.MethodPatch},
		{Path: "/c", Method: http.MethodPost},
		{Path: "/c", Method: http.MethodGet},
		{Path: "/b", Method: http.MethodOptions},
	}
	routes.Sort()
	assert.Equal(RouteDescriptions{
		{Path: "/a", Method: http.MethodGet},
		{Path: "/a", Method: http.MethodPost},
		{Path: "/a", Method: http.MethodDelete},
		{Path: "/a", Method: http.MethodOptions},
		{Path: "/a", Method: http.MethodPatch},
		{Path: "/b", Method: http.MethodOptions},
		{Path: "/c", Method: http.MethodGet},
		{Path: "/c", Method: http.MethodPost},
	}, routes)
}

func TestServiceHandler_list(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	err = createServiceRoutes(router)
	assert.Nil(err)

	// add services
	router.services = map[string]*ServiceItem{
		"svc-a": {Name: "svc-a"},
		"svc-c": {Name: "svc-c"},
		"svc-b": {Name: "svc-b", OpenAPIFiles: []*openapi.FileProperties{
			{Prefix: "/svc-b-1"},
			{Prefix: "/svc-b-2"},
		}},
	}

	// serve
	req := httptest.NewRequest("GET", "/.services", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	res := UnmarshallResponse[ServiceListResponse](t, w.Body)
	expected := ServiceListResponse{
		Items: []*ServiceItemResponse{
			{Name: "svc-a"},
			{
				Name:             "svc-b",
				OpenAPIResources: []string{"/svc-b-1", "/svc-b-2"},
			},
			{Name: "svc-c"},
		},
	}

	assert.Equal(http.StatusOK, w.Code)
	assert.Equal("application/json", w.Header().Get("Content-Type"))
	AssertJSONEqual(t, expected, res)
}

func TestServiceHandler_save_errors(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	err = createServiceRoutes(router)
	assert.Nil(err)

	t.Run("form-error", func(t *testing.T) {
		// serve
		req := httptest.NewRequest("POST", "/.services", strings.NewReader("InvalidMultipartData"))
		req.Header.Set("Content-Type", "multipart/form-data; boundary=invalidboundary")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(400, w.Code)
		resp := UnmarshallResponse[SimpleResponse](t, w.Body)
		assert.Equal("application/json", w.Header().Get("Content-Type"))
		assert.Equal(false, resp.Success)
		assert.Equal("multipart: NextPart: EOF", resp.Message)
	})

	t.Run("save-error", func(t *testing.T) {
		// prepare payload
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)

		_ = writer.WriteField("path", "petstore")
		_ = writer.WriteField("method", "X")
		_ = writer.Close()

		// serve
		req := httptest.NewRequest("POST", "/.services", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		resp := UnmarshallResponse[SimpleResponse](t, w.Body)
		assert.Equal(400, w.Code)
		assert.Equal(false, resp.Success)
		assert.Equal("invalid HTTP verb", resp.Message)
	})
}

func TestServiceHandler_save_openAPI(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	err = createServiceRoutes(router)
	assert.Nil(err)

	// prepare payload
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	_ = writer.WriteField("path", "petstore")
	_ = writer.WriteField("isOpenApi", "true")
	err = AddTestFileToForm(writer, "file", filepath.Join(testDataPath, "document-petstore.yml"))
	assert.Nil(err)

	_ = writer.Close()

	// serve
	req := httptest.NewRequest("POST", "/.services", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := UnmarshallResponse[SimpleResponse](t, w.Body)
	assert.Equal(http.StatusOK, w.Code)
	assert.Equal("application/json", w.Header().Get("Content-Type"))
	assert.Equal(true, resp.Success)
	assert.Equal("Resource saved!", resp.Message)

	svc := router.services["petstore"]
	expectedFileProps := &openapi.FileProperties{
		ServiceName: "petstore",
		IsOpenAPI:   false,
		Method:      "",
		Prefix:      "/petstore",
		Resource:    "",
		FilePath:    filepath.Join(router.Config.App.Paths.ServicesOpenAPI, "petstore", "index.yaml"),
		FileName:    "index.yaml",
		Extension:   ".yaml",
		ContentType: "application/x-yaml",
		Spec:        nil,
	}

	expected := &ServiceItem{
		Name: "petstore",
		OpenAPIFiles: []*openapi.FileProperties{
			expectedFileProps,
		},
		Routes: []*RouteDescription{
			{
				Method: "GET",
				Path:   "/pets",
				Type:   OpenAPIRouteType,
			},
			{
				Method: "POST",
				Path:   "/pets",
				Type:   OpenAPIRouteType,
			},
			{
				Method: "GET",
				Path:   "/pets/{id}",
				Type:   OpenAPIRouteType,
			},
			{
				Method: "DELETE",
				Path:   "/pets/{id}",
				Type:   OpenAPIRouteType,
			},
		},
	}
	AssertJSONEqual(t, expected, svc)
}

func TestServiceHandler_save_fixed(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	err = createServiceRoutes(router)
	assert.Nil(err)

	// prepare payload
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	_ = writer.WriteField("path", "/petstore/pets/update/{tag}")
	_ = writer.WriteField("method", http.MethodPatch)
	_ = writer.WriteField("response", `{"hallo":"welt!"}`)
	_ = writer.WriteField("contentType", "json")

	assert.Nil(err)
	_ = writer.Close()

	// serve
	req := httptest.NewRequest("POST", "/.services", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := UnmarshallResponse[SavedResourceResponse](t, w.Body)
	assert.Equal(http.StatusOK, w.Code)
	assert.Equal("application/json", w.Header().Get("Content-Type"))
	assert.Equal(true, resp.Success)
	assert.Equal("Resource saved!", resp.Message)
	assert.Equal(0, resp.ID)

	svc := router.services["petstore"]
	targetPath := filepath.Join(router.Config.App.Paths.Services, "petstore", "patch", "pets", "update", "{tag}", "index.json")
	expectedFileProps := &openapi.FileProperties{
		ServiceName: "petstore",
		IsOpenAPI:   false,
		Method:      http.MethodPatch,
		Prefix:      "/petstore",
		Resource:    "/pets/update/{tag}",
		FilePath:    targetPath,
		FileName:    "index.json",
		Extension:   ".json",
		ContentType: "application/json",
	}
	expected := &ServiceItem{
		Name: "petstore",
		Routes: []*RouteDescription{
			{
				Method:      http.MethodPatch,
				Path:        "/pets/update/{tag}",
				Type:        FixedRouteType,
				File:        expectedFileProps,
				ContentType: "application/json",
			},
		},
	}
	AssertJSONEqual(t, expected, svc)

	content, err := os.ReadFile(targetPath)
	assert.Nil(err)
	assert.Equal(`{"hallo":"welt!"}`, string(content))
}

func TestServiceHandler_save_fixedMultiple(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	err = createServiceRoutes(router)
	assert.Nil(err)

	for i := 0; i < 3; i++ {
		// prepare payload
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)

		_ = writer.WriteField("path", "/petstore/pets/update/{tag}")
		_ = writer.WriteField("method", http.MethodPatch)
		_ = writer.WriteField("response", `{"hallo":"welt!"}`)
		_ = writer.WriteField("contentType", "json")

		assert.Nil(err)
		_ = writer.Close()

		// serve
		req := httptest.NewRequest("POST", "/.services", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		resp := UnmarshallResponse[SavedResourceResponse](t, w.Body)
		assert.Equal(http.StatusOK, w.Code)
		assert.Equal("application/json", w.Header().Get("Content-Type"))
		assert.Equal(true, resp.Success)
		assert.Equal("Resource saved!", resp.Message)
		assert.Equal(0, resp.ID)

		svc := router.services["petstore"]
		assert.Equal(1, len(svc.Routes))
	}
}

func TestServiceHandler_save_fixedWithOverwrite(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	err = createServiceRoutes(router)
	assert.Nil(err)

	router.services = map[string]*ServiceItem{
		"petstore": {
			Name: "petstore",
			Routes: []*RouteDescription{
				{
					Method: http.MethodPatch,
					Path:   "/pets/update/{tag}",
					Type:   OpenAPIRouteType,
				},
			},
		},
	}

	// prepare payload
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// overwrite existing openAPI route
	_ = writer.WriteField("path", "petstore/pets/update/{tag}")
	_ = writer.WriteField("method", http.MethodPatch)
	_ = writer.WriteField("response", `{"hallo":"welt!"}`)
	_ = writer.WriteField("contentType", "json")

	assert.Nil(err)
	_ = writer.Close()

	// serve
	req := httptest.NewRequest("POST", "/.services", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := UnmarshallResponse[SavedResourceResponse](t, w.Body)
	assert.Equal(http.StatusOK, w.Code)
	assert.Equal("application/json", w.Header().Get("Content-Type"))
	assert.Equal(1, resp.ID)
	assert.Equal(true, resp.Success)
	assert.Equal("Resource saved!", resp.Message)

	svc := router.services["petstore"]
	expected := &ServiceItem{
		Name: "petstore",
		Routes: []*RouteDescription{
			{
				Method: http.MethodPatch,
				Path:   "/pets/update/{tag}",
				Type:   OpenAPIRouteType,
			},
			{
				Method:      http.MethodPatch,
				Path:        "/pets/update/{tag}",
				Type:        FixedRouteType,
				ContentType: "application/json",
				Overwrites:  true,
			},
		},
	}
	AssertJSONEqual(t, expected, svc)
}

func TestServiceHandler_resources_errors(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	err = createServiceRoutes(router)
	assert.Nil(err)

	t.Run("unknown-service", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/.services/"+config.RootServiceName, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		resp := UnmarshallResponse[SimpleResponse](t, w.Body)
		assert.Equal(http.StatusNotFound, w.Code)
		assert.Equal("application/json", w.Header().Get("Content-Type"))
		assert.Equal(false, resp.Success)
		assert.Equal("Service not found", resp.Message)
	})
}

func TestServiceHandler_resources(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	err = createServiceRoutes(router)
	assert.Nil(err)

	routes := RouteDescriptions{
		{
			Method: http.MethodGet,
			Path:   "/pets",
			Type:   OpenAPIRouteType,
		},
		{
			Method: http.MethodPatch,
			Path:   "/pets",
			Type:   OpenAPIRouteType,
		},
		{
			Method:      http.MethodGet,
			Path:        "/pets",
			Type:        FixedRouteType,
			ContentType: "application/json",
			Overwrites:  true,
		},
	}
	router.services = map[string]*ServiceItem{
		"petstore": {
			Name:   "petstore",
			Routes: routes,
			OpenAPIFiles: []*openapi.FileProperties{
				{
					Prefix: "index-pets.yml",
				},
			},
		},
	}

	req := httptest.NewRequest("GET", "/.services/petstore", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	expected := &ServiceResourcesResponse{
		Endpoints: routes,
		Service: &ServiceEmbedded{
			Name: "petstore",
		},
		OpenAPISpecNames: []string{"index-pets.yml"},
	}

	resp := UnmarshallResponse[ServiceResourcesResponse](t, w.Body)
	assert.Equal(http.StatusOK, w.Code)
	assert.Equal("application/json", w.Header().Get("Content-Type"))
	assert.Equal(expected, resp)
}

func TestServiceHandler_deleteService_errors(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	filePath := filepath.Join(router.Config.App.Paths.Services, "petstore", "post", "pets", "index.json")
	err = files.CopyFile(filepath.Join(testDataPath, "fixed-petstore-post-pets.json"), filePath)
	assert.Nil(err)

	err = createServiceRoutes(router)
	assert.Nil(err)

	t.Run("not-found", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/.services/x", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		resp := UnmarshallResponse[SimpleResponse](t, w.Body)
		assert.Equal(http.StatusNotFound, w.Code)
		assert.Equal("application/json", w.Header().Get("Content-Type"))
		assert.Equal(false, resp.Success)
		assert.Equal(ErrServiceNotFound.Error(), resp.Message)
	})
}

func TestServiceHandler_deleteService(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	router.services = map[string]*ServiceItem{
		"": {
			Name: "",
			Routes: []*RouteDescription{
				{
					Method: http.MethodGet,
					Path:   "/pets",
					File: &openapi.FileProperties{
						FilePath: filepath.Join(router.Config.App.Paths.Services, "get", "pets", "index.json"),
					},
				},
			},
		},
	}

	err = createServiceRoutes(router)
	assert.Nil(err)

	req := httptest.NewRequest("DELETE", "/.services/"+config.RootServiceName, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := UnmarshallResponse[SimpleResponse](t, w.Body)
	assert.Equal(http.StatusOK, w.Code)
	assert.Equal("application/json", w.Header().Get("Content-Type"))
	assert.Equal(true, resp.Success)
	assert.Equal("Service deleted!", resp.Message)
}

func TestServiceHandler_spec_errors(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	err = createServiceRoutes(router)
	assert.Nil(err)

	t.Run("unknown-service", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/.services/"+config.RootServiceName+"/spec", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		resp := UnmarshallResponse[SimpleResponse](t, w.Body)
		assert.Equal(http.StatusNotFound, w.Code)
		assert.Equal("Service not found", resp.Message)
	})

	t.Run("no-spec-attached", func(t *testing.T) {
		router.services = map[string]*ServiceItem{
			"petstore": {
				Name: "petstore",
			},
		}

		req := httptest.NewRequest("GET", "/.services/petstore/spec", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		resp := UnmarshallResponse[SimpleResponse](t, w.Body)
		assert.Equal(http.StatusNotFound, w.Code)
		assert.Equal("No Spec files attached", resp.Message)
	})

	t.Run("error-reading-spec", func(t *testing.T) {
		router.services = map[string]*ServiceItem{
			"petstore": {
				Name: "petstore",
				OpenAPIFiles: []*openapi.FileProperties{
					{
						Prefix: "index-pets.yml",
					},
				},
			},
		}

		req := httptest.NewRequest("GET", "/.services/petstore/spec", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		resp := UnmarshallResponse[SimpleResponse](t, w.Body)
		assert.Equal(http.StatusInternalServerError, w.Code)
		assert.Equal("open : no such file or directory", resp.Message)
	})
}

func TestServiceHandler_spec_happyPath(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	filePath := filepath.Join(router.Config.App.Paths.ServicesOpenAPI, "petstore", "index-pets.yml")
	err = files.CopyFile(filepath.Join(testDataPath, "document-petstore.yml"), filePath)
	assert.Nil(err)

	router.services = map[string]*ServiceItem{
		"petstore": {
			Name: "petstore",
			OpenAPIFiles: []*openapi.FileProperties{
				{
					FilePath: filePath,
				},
			},
		},
	}

	err = createServiceRoutes(router)
	assert.Nil(err)

	req := httptest.NewRequest("GET", "/.services/petstore/spec", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := w.Body.String()
	assert.Equal(http.StatusOK, w.Code)
	assert.Equal("text/plain", w.Header().Get("Content-Type"))
	assert.True(strings.HasPrefix(resp, `openapi: "3.0.0"`))
}

func TestServiceHandler_generate_errors(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	err = createServiceRoutes(router)
	assert.Nil(err)

	t.Run("unknown-service", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/.services/"+config.RootServiceName+"/0", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		resp := UnmarshallResponse[SimpleResponse](t, w.Body)
		assert.Equal(http.StatusNotFound, w.Code)
		assert.Equal(ErrServiceNotFound.Error(), resp.Message)
		assert.Equal(false, resp.Success)
	})

	t.Run("invalid-ix", func(t *testing.T) {
		router.services = map[string]*ServiceItem{
			"petstore": {
				Name: "petstore",
			},
		}

		req := httptest.NewRequest("POST", "/.services/petstore/x", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		resp := UnmarshallResponse[SimpleResponse](t, w.Body)
		assert.Equal(http.StatusNotFound, w.Code)
		assert.Equal(ErrResourceNotFound.Error(), resp.Message)
		assert.Equal(false, resp.Success)
	})

	t.Run("invalid-payload", func(t *testing.T) {
		router.services = map[string]*ServiceItem{
			"petstore": {
				Name: "petstore",
				Routes: RouteDescriptions{
					{
						Method: http.MethodGet,
						Path:   "/pets",
						Type:   OpenAPIRouteType,
					},
				},
			},
		}

		payload := strings.NewReader(`{"replacements": 1}`)

		req := httptest.NewRequest("POST", "/.services/petstore/0", payload)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		resp := UnmarshallResponse[SimpleResponse](t, w.Body)
		assert.Equal(http.StatusBadRequest, w.Code)
		assert.True(strings.HasPrefix(resp.Message, "json: cannot unmarshal number into Go struct field"))
		assert.Equal(false, resp.Success)
	})

	t.Run("file-not-found", func(t *testing.T) {
		router.services = map[string]*ServiceItem{
			"petstore": {
				Name: "petstore",
				Routes: RouteDescriptions{
					{
						Method: http.MethodGet,
						Path:   "/pets",
						Type:   OpenAPIRouteType,
					},
				},
			},
		}

		req := httptest.NewRequest("POST", "/.services/petstore/0", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		resp := UnmarshallResponse[SimpleResponse](t, w.Body)
		assert.Equal(http.StatusNotFound, w.Code)
		assert.Equal(ErrResourceNotFound.Error(), resp.Message)
		assert.Equal(false, resp.Success)
	})

	t.Run("method-not-allowed", func(t *testing.T) {
		filePath := filepath.Join(router.Config.App.Paths.ServicesOpenAPI, "petstore", "index-pets.yml")
		err = files.CopyFile(filepath.Join(testDataPath, "document-petstore.yml"), filePath)
		assert.Nil(err)
		file, err := openapi.GetPropertiesFromFilePath(filePath, router.Config.App)
		assert.Nil(err)

		router.services = map[string]*ServiceItem{
			"petstore": {
				Name: "petstore",
				Routes: RouteDescriptions{
					{
						Method: http.MethodOptions,
						Path:   "/pets",
						Type:   OpenAPIRouteType,
						File:   file,
					},
				},
			},
		}

		req := httptest.NewRequest("POST", "/.services/petstore/0", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		resp := UnmarshallResponse[SimpleResponse](t, w.Body)
		assert.Equal(http.StatusNotFound, w.Code)
		assert.Equal(ErrResourceMethodNotFound.Error(), resp.Message)
		assert.Equal(false, resp.Success)
	})
}

func TestServiceHandler_generate_openAPI(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	filePath := filepath.Join(router.Config.App.Paths.ServicesOpenAPI, "petstore", "index-pets.yml")
	err = files.CopyFile(filepath.Join(testDataPath, "document-petstore.yml"), filePath)
	assert.Nil(err)
	file, err := openapi.GetPropertiesFromFilePath(filePath, router.Config.App)
	assert.Nil(err)

	router.Config.Services[file.ServiceName] = &config.ServiceConfig{}

	router.services = map[string]*ServiceItem{
		"petstore": {
			Name: "petstore",
			Routes: RouteDescriptions{
				{
					Method: http.MethodPost,
					Path:   "/pets",
					Type:   OpenAPIRouteType,
					File:   file,
				},
			},
		},
	}
	router.Config.Services["petstore"].Contexts = nil

	err = createServiceRoutes(router)
	assert.Nil(err)

	replacements := map[string]any{
		"limit":  100,
		"offset": 2,
		"tag":    "Hund",
		"name":   "Hans",
		"id":     10,
	}
	replJs, _ := json.Marshal(replacements)
	payload := strings.NewReader(fmt.Sprintf(`{"replacements": %s}`, replJs))

	req := httptest.NewRequest("POST", "/.services/petstore/0", payload)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	expected := &GenerateResponse{
		Request: &openapi.GeneratedRequest{
			Method:      http.MethodPost,
			Path:        "/petstore/pets",
			Body:        `{"tag":"Hund","name":"Hans"}`,
			ContentType: "application/json",
			Query:       "",
			Examples: &openapi.ContentExample{
				CURL: `--data-raw '{"name":"Hans","tag":"Hund"}'`,
			},
		},
		Response: &openapi.GeneratedResponse{
			Content:     []byte(`{"id":10,"name":"Hans","tag":"Hund"}`),
			ContentType: "application/json",
			StatusCode:  http.StatusOK,
			Headers: http.Header{
				"Content-Type": []string{"application/json"},
			},
		},
	}

	resp := UnmarshallResponse[GenerateResponse](t, w.Body)
	assert.Equal(http.StatusOK, w.Code)
	assert.Equal("application/json", w.Header().Get("Content-Type"))

	assert.Equal(expected.Request.Method, resp.Request.Method)
	assert.Equal(expected.Request.Path, resp.Request.Path)
	assert.Equal(expected.Request.ContentType, resp.Request.ContentType)
	assert.Equal(expected.Request.Query, resp.Request.Query)

	assert.Equal(string(expected.Response.Content), string(resp.Response.Content))
	assert.Equal(expected.Response.ContentType, resp.Response.ContentType)
	assert.Equal(expected.Response.StatusCode, resp.Response.StatusCode)
	assert.Equal(expected.Response.Headers, resp.Response.Headers)

	reqBody := make(map[string]any)
	_ = json.Unmarshal([]byte(resp.Request.Body), &reqBody)
	assert.Equal(map[string]any{
		"name": "Hans",
		"tag":  "Hund",
	}, reqBody)

	respBody := make(map[string]any)
	_ = json.Unmarshal(resp.Response.Content, &respBody)
	assert.Equal(map[string]any{
		"id":   float64(10),
		"name": "Hans",
		"tag":  "Hund",
	}, respBody)
}

func TestServiceHandler_generate_fixed(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	filePath := filepath.Join(router.Config.App.Paths.Services, "petstore", "post", "pets", "index.json")
	err = files.CopyFile(filepath.Join(testDataPath, "fixed-petstore-post-pets.json"), filePath)
	assert.Nil(err)
	file, err := openapi.GetPropertiesFromFilePath(filePath, router.Config.App)
	assert.Nil(err)
	if err != nil {
		t.FailNow()
	}

	router.services = map[string]*ServiceItem{
		"petstore": {
			Name: "petstore",
			Routes: RouteDescriptions{
				{
					Method: http.MethodPost,
					Path:   "/pets",
					Type:   FixedRouteType,
					File:   file,
				},
			},
		},
	}

	err = createServiceRoutes(router)
	assert.Nil(err)

	req := httptest.NewRequest("POST", "/.services/petstore/0", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	expected := &GenerateResponse{
		Request: &openapi.GeneratedRequest{
			Method:      http.MethodPost,
			Path:        "/petstore/pets",
			ContentType: "application/json",
		},
		Response: &openapi.GeneratedResponse{
			Content:     []byte(`{"id":1,"name":"Bulbasaur","tag":"beedrill"}`),
			ContentType: "application/json",
			StatusCode:  http.StatusOK,
			Headers: http.Header{
				"Content-Type": []string{"application/json"},
			},
		},
	}

	resp := UnmarshallResponse[GenerateResponse](t, w.Body)
	assert.Equal(http.StatusOK, w.Code)
	assert.Equal("application/json", w.Header().Get("Content-Type"))

	assert.Equal(expected.Request.Method, resp.Request.Method)
	assert.Equal(expected.Request.Path, resp.Request.Path)
	assert.Equal(expected.Request.ContentType, resp.Request.ContentType)
	assert.Equal(expected.Request.Query, resp.Request.Query)

	assert.Equal(expected.Response.ContentType, resp.Response.ContentType)
	assert.Equal(expected.Response.StatusCode, resp.Response.StatusCode)
	assert.Equal(expected.Response.Headers, resp.Response.Headers)

	respBody := make(map[string]any)
	_ = json.Unmarshal(resp.Response.Content, &respBody)
	assert.Equal(map[string]any{
		"id":   float64(1),
		"name": "Bulbasaur",
		"tag":  "beedrill",
	}, respBody)
}

func TestServiceHandler_getResource_errors(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	err = createServiceRoutes(router)
	assert.Nil(err)

	t.Run("unknown-service", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/.services/x/0", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		resp := UnmarshallResponse[SimpleResponse](t, w.Body)
		assert.Equal(http.StatusNotFound, w.Code)
		assert.Equal("application/json", w.Header().Get("Content-Type"))
		assert.Equal(false, resp.Success)
		assert.Equal(ErrServiceNotFound.Error(), resp.Message)
	})

	t.Run("invalid-ix", func(t *testing.T) {
		router.services = map[string]*ServiceItem{
			"petstore": {
				Name: "petstore",
				Routes: RouteDescriptions{
					{
						Method: http.MethodPost,
						Path:   "/pets",
						Type:   FixedRouteType,
					},
				},
			},
		}

		req := httptest.NewRequest("GET", "/.services/petstore/x", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		resp := UnmarshallResponse[SimpleResponse](t, w.Body)
		assert.Equal(http.StatusNotFound, w.Code)
		assert.Equal("application/json", w.Header().Get("Content-Type"))
		assert.Equal(false, resp.Success)
		assert.Equal(ErrResourceNotFound.Error(), resp.Message)
	})

	t.Run("error-reading-file", func(t *testing.T) {
		router.services = map[string]*ServiceItem{
			"petstore": {
				Name: "petstore",
				Routes: RouteDescriptions{
					{
						Method: http.MethodPost,
						Path:   "/pets",
						Type:   FixedRouteType,
						File: &openapi.FileProperties{
							FilePath: "unknown",
						},
					},
				},
			},
		}

		req := httptest.NewRequest("GET", "/.services/petstore/0", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		resp := UnmarshallResponse[SimpleResponse](t, w.Body)
		assert.Equal(http.StatusInternalServerError, w.Code)
		assert.Equal("application/json", w.Header().Get("Content-Type"))
		assert.Equal(false, resp.Success)
		assert.Equal("open unknown: no such file or directory", resp.Message)
	})

	t.Run("not-fixed-resource", func(t *testing.T) {
		router.services = map[string]*ServiceItem{
			"petstore": {
				Name: "petstore",
				Routes: RouteDescriptions{
					{
						Method: http.MethodPost,
						Path:   "/pets",
						Type:   OpenAPIRouteType,
						File:   &openapi.FileProperties{},
					},
				},
			},
		}

		req := httptest.NewRequest("GET", "/.services/petstore/0", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		resp := UnmarshallResponse[SimpleResponse](t, w.Body)
		assert.Equal(http.StatusBadRequest, w.Code)
		assert.Equal("application/json", w.Header().Get("Content-Type"))
		assert.Equal(false, resp.Success)
		assert.Equal(ErrOnlyFixedResourcesAllowedEditing.Error(), resp.Message)
	})
}

func TestServiceHandler_getResource(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	err = createServiceRoutes(router)
	assert.Nil(err)

	filePath := filepath.Join(router.Config.App.Paths.Services, "petstore", "post", "pets", "index.json")
	_ = files.CopyFile(filepath.Join(testDataPath, "fixed-petstore-post-pets.json"), filePath)
	fileContents, _ := os.ReadFile(filePath)
	fileProps, _ := openapi.GetPropertiesFromFilePath(filePath, router.Config.App)

	router.services = map[string]*ServiceItem{
		"petstore": {
			Name: "petstore",
			Routes: RouteDescriptions{
				{
					Method: http.MethodPost,
					Path:   "/pets",
					Type:   FixedRouteType,
					File:   fileProps,
				},
			},
		},
	}

	req := httptest.NewRequest("GET", "/.services/petstore/0", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	expected := &ResourceResponse{
		Method:      http.MethodPost,
		Path:        "/petstore/pets",
		Extension:   "json",
		ContentType: "application/json",
		Content:     string(fileContents),
	}

	resp := UnmarshallResponse[ResourceResponse](t, w.Body)
	assert.Equal(http.StatusOK, w.Code)
	assert.Equal("application/json", w.Header().Get("Content-Type"))
	assert.Equal(expected, resp)
}

func TestServiceHandler_deleteResource_errors(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	err = createServiceRoutes(router)
	assert.Nil(err)

	t.Run("unknown-service", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/.services/petstore/0", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		resp := UnmarshallResponse[SimpleResponse](t, w.Body)
		assert.Equal(http.StatusNotFound, w.Code)
		assert.Equal("application/json", w.Header().Get("Content-Type"))
		assert.Equal(false, resp.Success)
		assert.Equal(ErrServiceNotFound.Error(), resp.Message)
	})

	t.Run("invalid-ix", func(t *testing.T) {
		router.services = map[string]*ServiceItem{
			"petstore": {
				Name: "petstore",
				Routes: RouteDescriptions{
					{
						Method: http.MethodPost,
						Path:   "/pets",
						Type:   FixedRouteType,
					},
				},
			},
		}

		req := httptest.NewRequest("DELETE", "/.services/petstore/x", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		resp := UnmarshallResponse[SimpleResponse](t, w.Body)
		assert.Equal(http.StatusNotFound, w.Code)
		assert.Equal("application/json", w.Header().Get("Content-Type"))
		assert.Equal(false, resp.Success)
		assert.Equal(ErrResourceNotFound.Error(), resp.Message)
	})

	t.Run("not-fixed-resource", func(t *testing.T) {
		router.services = map[string]*ServiceItem{
			"petstore": {
				Name: "petstore",
				Routes: RouteDescriptions{
					{
						Method: http.MethodPost,
						Path:   "/pets",
						Type:   OpenAPIRouteType,
					},
				},
			},
		}

		req := httptest.NewRequest("DELETE", "/.services/petstore/0", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		resp := UnmarshallResponse[SimpleResponse](t, w.Body)
		assert.Equal(http.StatusBadRequest, w.Code)
		assert.Equal("application/json", w.Header().Get("Content-Type"))
		assert.Equal(false, resp.Success)
		assert.Equal(ErrOnlyFixedResourcesAllowedEditing.Error(), resp.Message)
	})

	t.Run("error reading file", func(t *testing.T) {
		router.services = map[string]*ServiceItem{
			"petstore": {
				Name: "petstore",
				Routes: RouteDescriptions{
					{
						Method: http.MethodPost,
						Path:   "/pets",
						Type:   FixedRouteType,
						File: &openapi.FileProperties{
							FilePath: "unknown",
						},
					},
				},
			},
		}

		req := httptest.NewRequest("DELETE", "/.services/petstore/0", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		resp := UnmarshallResponse[SimpleResponse](t, w.Body)
		assert.Equal(http.StatusInternalServerError, w.Code)
		assert.Equal("application/json", w.Header().Get("Content-Type"))
		assert.Equal(false, resp.Success)
		assert.Equal("remove unknown: no such file or directory", resp.Message)
	})

}

func TestServiceHandler_deleteResource(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	err = createServiceRoutes(router)
	assert.Nil(err)

	router.services = map[string]*ServiceItem{
		"petstore": {
			Name: "petstore",
			Routes: RouteDescriptions{
				{
					Method: http.MethodGet,
					Path:   "/pets",
					Type:   FixedRouteType,
				},
				{
					Method: http.MethodPost,
					Path:   "/pets",
					Type:   FixedRouteType,
				},
			},
		},
	}

	req := httptest.NewRequest("DELETE", "/.services/petstore/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := UnmarshallResponse[SimpleResponse](t, w.Body)
	assert.Equal(http.StatusOK, w.Code)
	assert.Equal("application/json", w.Header().Get("Content-Type"))
	assert.Equal(true, resp.Success)
	assert.Equal("Resource deleted!", resp.Message)

	assert.Equal(1, len(router.services["petstore"].Routes))
	assert.Equal(http.MethodGet, router.services["petstore"].Routes[0].Method)
	assert.Equal("/pets", router.services["petstore"].Routes[0].Path)
}

func TestSaveService_errors(t *testing.T) {
	assert := assert2.New(t)

	appDir := t.TempDir()
	appCfg := config.NewDefaultAppConfig(appDir)
	_, _ = SetupApp(appDir)

	t.Run("invalid-url-resource", func(t *testing.T) {
		payload := &ServicePayload{
			Method: http.MethodPatch,
			Path:   "/{}",
		}
		res, err := saveService(payload, appCfg)
		assert.Nil(res)
		assert.Equal(ErrInvalidURLResource, err)
	})

	t.Run("empty-openapi-content", func(t *testing.T) {
		payload := &ServicePayload{
			Method:    http.MethodPatch,
			Path:      "/x",
			IsOpenAPI: true,
		}
		res, err := saveService(payload, appCfg)
		assert.Nil(res)
		assert.Equal(ErrOpenAPISpecIsEmpty, err)
	})

	t.Run("savefile-error", func(t *testing.T) {
		_ = os.Chmod(appDir, 0400)
		payload := &ServicePayload{
			Method:   http.MethodPatch,
			Path:     "/x",
			Response: []byte("check-check"),
		}
		res, err := saveService(payload, appCfg)
		assert.Nil(res)
		assert.True(strings.HasSuffix(err.Error(), "resources: permission denied"))

		_ = os.Chmod(appDir, 0777)
	})

	t.Run("err-getting-properties", func(t *testing.T) {
		payload := &ServicePayload{
			Method:    http.MethodPatch,
			Path:      "/x",
			Response:  []byte("check-check"),
			IsOpenAPI: true,
		}
		res, err := saveService(payload, appCfg)
		assert.Nil(res)
		assert.NotNil(err)
	})

	t.Run("collides-with-sys-routes", func(t *testing.T) {
		payload := &ServicePayload{
			Method:   http.MethodPatch,
			Path:     "/.services",
			Response: []byte("check-check"),
		}
		res, err := saveService(payload, appCfg)
		assert.Nil(res)
		assert.Equal(ErrReservedPrefix, err)
	})

	t.Run("from-file-download", func(t *testing.T) {
		mockServer := testhelpers.CreateMockServer(t, "text/plain", "Hallo, Welt!", http.StatusNotFound)
		defer mockServer.Close()

		payload := &ServicePayload{
			Method: http.MethodPatch,
			Path:   "/x",
			URL:    mockServer.URL,
		}
		res, err := saveService(payload, appCfg)
		assert.Nil(res)
		assert.Equal(ErrGettingFileFromURL, err)
	})
}

func TestServiceHandler_getRouteIndex(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	err = createServiceRoutes(router)
	assert.Nil(err)

	handler := &ServiceHandler{
		router: router,
	}

	router.services = map[string]*ServiceItem{
		"petstore": {
			Name: "petstore",
		},
	}

	t.Run("service-not-found", func(t *testing.T) {
		res := handler.getRouteIndex(&openapi.FileProperties{
			ServiceName: "nice",
		})
		assert.Equal(-1, res)
	})
}

func TestComposeFileSavePath(t *testing.T) {
	t.Parallel()

	appCfg := config.NewDefaultAppConfig("/app")
	paths := appCfg.Paths

	testCases := []struct {
		service  *ServiceDescription
		expected string
	}{
		{
			service: &ServiceDescription{
				Path: "/foo.html",
			},
			expected: paths.Services + "/root/get/foo.html",
		},
		{
			service: &ServiceDescription{
				Method: "patch",
				Path:   "/foo.html",
			},
			expected: paths.Services + "/root/patch/foo.html",
		},
		{
			service: &ServiceDescription{
				Method: "get",
				Path:   "test/test-path",
				Ext:    ".json",
			},
			expected: paths.Services + "/test/get/test-path/index.json",
		},
		{
			service: &ServiceDescription{
				Path: "/foo/bar",
			},
			expected: paths.Services + "/foo/get/bar/index.txt",
		},
		{
			service: &ServiceDescription{
				Path:   "/nice",
				Method: "patch",
			},
			expected: paths.Services + "/nice/patch/index.txt",
		},
		{
			service: &ServiceDescription{
				Path: "/x",
				Ext:  ".json",
			},
			expected: paths.Services + "/x/get/index.json",
		},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			actual := ComposeFileSavePath(tc.service, paths)
			if actual != tc.expected {
				t.Errorf("ComposeFileSavePath(%v): - Expected: %v, Got: %v",
					tc.service, tc.expected, actual)
			}
		})
	}
}

func TestComposeOpenAPISavePath(t *testing.T) {
	t.Parallel()

	appCfg := config.NewDefaultAppConfig("/app")
	paths := appCfg.Paths

	testCases := []struct {
		service  *ServiceDescription
		expected string
	}{
		{
			service:  &ServiceDescription{},
			expected: paths.ServicesOpenAPI + "/index",
		},
		{
			service: &ServiceDescription{
				Ext: ".yml",
			},
			expected: paths.ServicesOpenAPI + "/index.yml",
		},
		{
			service: &ServiceDescription{
				Path: "petstore",
				Ext:  ".yml",
			},
			expected: paths.ServicesOpenAPI + "/petstore/index.yml",
		},
		{
			service: &ServiceDescription{
				Path: "/petstore/v1",
				Ext:  ".yml",
			},
			expected: paths.ServicesOpenAPI + "/petstore/v1/index.yml",
		},
		{
			service: &ServiceDescription{
				Path: "/nice/dice/rice",
				Ext:  ".yml",
			},
			expected: paths.ServicesOpenAPI + "/nice/dice/rice/index.yml",
		},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			actual := ComposeOpenAPISavePath(tc.service, paths.ServicesOpenAPI)
			if actual != tc.expected {
				t.Errorf("ComposeFileSavePath(%v): Expected: %v, Got: %v",
					tc.service, tc.expected, actual)
			}
		})
	}
}
