package connexions

import (
	"bytes"
	"encoding/json"
	"fmt"
	assert2 "github.com/stretchr/testify/assert"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateServiceRoutes_Disabled(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()
	router, _ := SetupApp(t.TempDir())
	router.Config.App.DisableUI = true

	_ = CreateServiceRoutes(router)
	assert.Equal(0, len(router.Mux.Routes()))
}

func TestServiceItem_AddOpenAPIFile(t *testing.T) {
	assert := assert2.New(t)

	svc := &ServiceItem{}
	fileProps := &FileProperties{
		FileName: "index.yaml",
	}
	svc.AddOpenAPIFile(fileProps)
	svc.AddOpenAPIFile(fileProps)
	assert.Equal([]*FileProperties{fileProps}, svc.OpenAPIFiles)
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

	err = CreateServiceRoutes(router)
	assert.Nil(err)

	// add services
	router.Services = map[string]*ServiceItem{
		"svc-a": {Name: "svc-a"},
		"svc-c": {Name: "svc-c"},
		"svc-b": {Name: "svc-b", OpenAPIFiles: []*FileProperties{
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

	err = CreateServiceRoutes(router)
	assert.Nil(err)

	t.Run("form-error", func(t *testing.T) {
		// serve
		req := httptest.NewRequest("POST", "/.services", strings.NewReader("InvalidMultipartData"))
		req.Header.Set("Content-Type", "multipart/form-data; boundary=invalidboundary")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(400, w.Code)
		resp := *UnmarshallResponse[map[string]any](t, w.Body)
		assert.Equal("application/json", w.Header().Get("Content-Type"))
		assert.Equal(false, resp["success"].(bool))
		assert.Equal("multipart: NextPart: EOF", resp["message"].(string))
	})

	t.Run("save-error", func(t *testing.T) {
		// prepare payload
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)

		_ = writer.WriteField("name", "petstore")
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

	err = CreateServiceRoutes(router)
	assert.Nil(err)

	// prepare payload
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	_ = writer.WriteField("name", "petstore")
	_ = writer.WriteField("isOpenApi", "true")
	err = AddTestFileToForm(writer, "file", filepath.Join("test_fixtures", "document-petstore.yml"))
	assert.Nil(err)

	_ = writer.Close()

	// serve
	req := httptest.NewRequest("POST", "/.services", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := *UnmarshallResponse[map[string]any](t, w.Body)
	assert.Equal(http.StatusOK, w.Code)
	assert.Equal("application/json", w.Header().Get("Content-Type"))
	assert.Equal(true, resp["success"].(bool))
	assert.Equal("Resource saved!", resp["message"].(string))

	svc := router.Services["petstore"]
	expectedFileProps := &FileProperties{
		ServiceName:          "petstore",
		IsOpenAPI:            false,
		Method:               "",
		Prefix:               "/petstore",
		Resource:             "",
		FilePath:             filepath.Join(router.Config.App.Paths.ServicesOpenAPI, "petstore", "index.yaml"),
		FileName:             "index.yaml",
		Extension:            ".yaml",
		ContentType:          "application/x-yaml",
		Spec:                 nil,
		ValueReplacerFactory: nil,
	}

	expected := &ServiceItem{
		Name: "petstore",
		OpenAPIFiles: []*FileProperties{
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

	err = CreateServiceRoutes(router)
	assert.Nil(err)

	// prepare payload
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	_ = writer.WriteField("name", "petstore")
	_ = writer.WriteField("path", "/pets/update/{tag}")
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

	resp := *UnmarshallResponse[map[string]any](t, w.Body)
	assert.Equal(http.StatusOK, w.Code)
	assert.Equal("application/json", w.Header().Get("Content-Type"))
	assert.Equal(true, resp["success"].(bool))
	assert.Equal("Resource saved!", resp["message"].(string))

	svc := router.Services["petstore"]
	targetPath := filepath.Join(router.Config.App.Paths.Services, "petstore", "patch", "pets", "update", "{tag}", "index.json")
	expectedFileProps := &FileProperties{
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

func TestServiceHandler_save_fixedWithOverwrite(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	err = CreateServiceRoutes(router)
	assert.Nil(err)

	router.Services = map[string]*ServiceItem{
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
	_ = writer.WriteField("name", "petstore")
	_ = writer.WriteField("path", "/pets/update/{tag}")
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

	resp := *UnmarshallResponse[map[string]any](t, w.Body)
	assert.Equal(http.StatusOK, w.Code)
	assert.Equal("application/json", w.Header().Get("Content-Type"))
	assert.Equal(true, resp["success"].(bool))
	assert.Equal("Resource saved!", resp["message"].(string))

	svc := router.Services["petstore"]

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

	err = CreateServiceRoutes(router)
	assert.Nil(err)

	t.Run("unknown-service", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/.services/"+RootServiceName, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		resp := *UnmarshallResponse[map[string]any](t, w.Body)
		assert.Equal(http.StatusNotFound, w.Code)
		assert.Equal("application/json", w.Header().Get("Content-Type"))
		assert.Equal(false, resp["success"].(bool))
		assert.Equal("Service not found", resp["message"].(string))
	})
}

func TestServiceHandler_resources(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	err = CreateServiceRoutes(router)
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
	router.Services = map[string]*ServiceItem{
		"petstore": {
			Name:   "petstore",
			Routes: routes,
			OpenAPIFiles: []*FileProperties{
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

	err = CreateServiceRoutes(router)
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

	t.Run("error-removing-dir", func(t *testing.T) {
		fileDir := filepath.Join(router.Config.App.Paths.Services, "petstore")
		_ = os.Chmod(fileDir, 0400)
		filePath := filepath.Join(fileDir, "get", "pets", "index.json")
		router.Services = map[string]*ServiceItem{
			"petstore": {
				Name: "petstore",
				Routes: []*RouteDescription{
					{
						File: &FileProperties{
							FilePath: filePath,
						},
					},
				},
			},
		}

		req := httptest.NewRequest("DELETE", "/.services/petstore", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		resp := UnmarshallResponse[SimpleResponse](t, w.Body)
		assert.Equal(http.StatusInternalServerError, w.Code)
		assert.Equal("application/json", w.Header().Get("Content-Type"))
		assert.Equal(false, resp.Success)
		assert.True(strings.HasSuffix(resp.Message, "permission denied"))

		// allow to clean up
		_ = os.Chmod(fileDir, 0777)
	})
}

func TestServiceHandler_deleteService(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	router.Services = map[string]*ServiceItem{
		"": {
			Name: "",
			Routes: []*RouteDescription{
				{
					Method: http.MethodGet,
					Path:   "/pets",
					File: &FileProperties{
						FilePath: filepath.Join(router.Config.App.Paths.Services, "get", "pets", "index.json"),
					},
				},
			},
		},
	}

	err = CreateServiceRoutes(router)
	assert.Nil(err)

	req := httptest.NewRequest("DELETE", "/.services/"+RootServiceName, nil)
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

	err = CreateServiceRoutes(router)
	assert.Nil(err)

	t.Run("unknown-service", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/.services/"+RootServiceName+"/spec", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		resp := UnmarshallResponse[SimpleResponse](t, w.Body)
		assert.Equal(http.StatusNotFound, w.Code)
		assert.Equal("Service not found", resp.Message)
	})

	t.Run("no-spec-attached", func(t *testing.T) {
		router.Services = map[string]*ServiceItem{
			"petstore": {
				Name: "petstore",
			},
		}

		req := httptest.NewRequest("GET", "/.services/petstore/spec", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		resp := *UnmarshallResponse[map[string]any](t, w.Body)
		assert.Equal(http.StatusNotFound, w.Code)
		assert.Equal("No Spec files attached", resp["message"].(string))
	})

	t.Run("error-reading-spec", func(t *testing.T) {
		router.Services = map[string]*ServiceItem{
			"petstore": {
				Name: "petstore",
				OpenAPIFiles: []*FileProperties{
					{
						Prefix: "index-pets.yml",
					},
				},
			},
		}

		req := httptest.NewRequest("GET", "/.services/petstore/spec", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		resp := *UnmarshallResponse[map[string]any](t, w.Body)
		assert.Equal(http.StatusInternalServerError, w.Code)
		assert.Equal("open : no such file or directory", resp["message"].(string))
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
	err = CopyFile(filepath.Join("test_fixtures", "document-petstore.yml"), filePath)
	assert.Nil(err)

	router.Services = map[string]*ServiceItem{
		"petstore": {
			Name: "petstore",
			OpenAPIFiles: []*FileProperties{
				{
					FilePath: filePath,
				},
			},
		},
	}

	err = CreateServiceRoutes(router)
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

	err = CreateServiceRoutes(router)
	assert.Nil(err)

	t.Run("unknown-service", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/.services/"+RootServiceName+"/0", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		resp := UnmarshallResponse[ErrorMessage](t, w.Body)
		assert.Equal(http.StatusNotFound, w.Code)
		assert.Equal(ErrServiceNotFound.Error(), resp.Message)
	})

	t.Run("invalid-ix", func(t *testing.T) {
		router.Services = map[string]*ServiceItem{
			"petstore": {
				Name: "petstore",
			},
		}

		req := httptest.NewRequest("POST", "/.services/petstore/x", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		resp := UnmarshallResponse[ErrorMessage](t, w.Body)
		assert.Equal(http.StatusNotFound, w.Code)
		assert.Equal(ErrResourceNotFound.Error(), resp.Message)
	})

	t.Run("invalid-payload", func(t *testing.T) {
		router.Services = map[string]*ServiceItem{
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

		resp := UnmarshallResponse[ErrorMessage](t, w.Body)
		assert.Equal(http.StatusBadRequest, w.Code)
		assert.True(strings.HasPrefix(resp.Message, "json: cannot unmarshal number into Go struct field"))
	})

	t.Run("file-not-found", func(t *testing.T) {
		router.Services = map[string]*ServiceItem{
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

		resp := UnmarshallResponse[ErrorMessage](t, w.Body)
		assert.Equal(http.StatusNotFound, w.Code)
		assert.Equal(ErrResourceNotFound.Error(), resp.Message)
	})

	t.Run("method-not-allowed", func(t *testing.T) {
		filePath := filepath.Join(router.Config.App.Paths.ServicesOpenAPI, "petstore", "index-pets.yml")
		err = CopyFile(filepath.Join("test_fixtures", "document-petstore.yml"), filePath)
		assert.Nil(err)
		file, err := GetPropertiesFromFilePath(filePath, router.Config.App)
		assert.Nil(err)

		router.Services = map[string]*ServiceItem{
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

		resp := UnmarshallResponse[ErrorMessage](t, w.Body)
		assert.Equal(http.StatusMethodNotAllowed, w.Code)
		assert.Equal(ErrResourceMethodNotFound.Error(), resp.Message)
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
	err = CopyFile(filepath.Join("test_fixtures", "document-petstore.yml"), filePath)
	assert.Nil(err)
	file, err := GetPropertiesFromFilePath(filePath, router.Config.App)
	assert.Nil(err)

	router.Services = map[string]*ServiceItem{
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

	err = CreateServiceRoutes(router)
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
		Request: &Request{
			Method:      http.MethodPost,
			Path:        "/petstore/pets",
			Body:        `{"tag":"Hund","name":"Hans"}`,
			ContentType: "application/json",
			Query:       "",
			Examples: &ContentExample{
				CURL: `--data-raw '{"name":"Hans","tag":"Hund"}'`,
			},
		},
		Response: &Response{
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

	assert.Equal(expected.Response.Content, resp.Response.Content)
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
	err = CopyFile(filepath.Join("test_fixtures", "fixed-petstore-post-pets.json"), filePath)
	assert.Nil(err)
	file, err := GetPropertiesFromFilePath(filePath, router.Config.App)
	assert.Nil(err)
	if err != nil {
		t.FailNow()
	}

	router.Services = map[string]*ServiceItem{
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

	err = CreateServiceRoutes(router)
	assert.Nil(err)

	req := httptest.NewRequest("POST", "/.services/petstore/0", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	expected := &GenerateResponse{
		Request: &Request{
			Method:      http.MethodPost,
			Path:        "/petstore/pets",
			ContentType: "application/json",
		},
		Response: &Response{
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

	err = CreateServiceRoutes(router)
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
		router.Services = map[string]*ServiceItem{
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
		router.Services = map[string]*ServiceItem{
			"petstore": {
				Name: "petstore",
				Routes: RouteDescriptions{
					{
						Method: http.MethodPost,
						Path:   "/pets",
						Type:   FixedRouteType,
						File: &FileProperties{
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
		router.Services = map[string]*ServiceItem{
			"petstore": {
				Name: "petstore",
				Routes: RouteDescriptions{
					{
						Method: http.MethodPost,
						Path:   "/pets",
						Type:   OpenAPIRouteType,
						File:   &FileProperties{},
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

	err = CreateServiceRoutes(router)
	assert.Nil(err)

	filePath := filepath.Join(router.Config.App.Paths.Services, "petstore", "post", "pets", "index.json")
	_ = CopyFile(filepath.Join("test_fixtures", "fixed-petstore-post-pets.json"), filePath)
	fileContents, _ := os.ReadFile(filePath)
	fileProps, _ := GetPropertiesFromFilePath(filePath, router.Config.App)

	router.Services = map[string]*ServiceItem{
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

	err = CreateServiceRoutes(router)
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
		router.Services = map[string]*ServiceItem{
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
		router.Services = map[string]*ServiceItem{
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
		router.Services = map[string]*ServiceItem{
			"petstore": {
				Name: "petstore",
				Routes: RouteDescriptions{
					{
						Method: http.MethodPost,
						Path:   "/pets",
						Type:   FixedRouteType,
						File: &FileProperties{
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

	err = CreateServiceRoutes(router)
	assert.Nil(err)

	router.Services = map[string]*ServiceItem{
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

	assert.Equal(1, len(router.Services["petstore"].Routes))
	assert.Equal(http.MethodGet, router.Services["petstore"].Routes[0].Method)
	assert.Equal("/pets", router.Services["petstore"].Routes[0].Path)
}

func TestSaveService_errors(t *testing.T) {
	assert := assert2.New(t)

	appDir := t.TempDir()
	appCfg := NewDefaultAppConfig(appDir)
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
			Name:      "",
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
			Name:     "",
			Response: []byte("check-check"),
		}
		res, err := saveService(payload, appCfg)
		assert.Nil(res)
		assert.Equal("error creating directories", err.Error())

		_ = os.Chmod(appDir, 0777)
	})

	t.Run("err-getting-properties", func(t *testing.T) {
		payload := &ServicePayload{
			Method:    http.MethodPatch,
			Path:      "/x",
			Name:      "",
			Response:  []byte("check-check"),
			IsOpenAPI: true,
		}
		res, err := saveService(payload, appCfg)
		assert.Nil(res)
		assert.Equal("spec type not supported by libopenapi, sorry", err.Error())
	})

	t.Run("collides-with-sys-routes", func(t *testing.T) {
		payload := &ServicePayload{
			Method:   http.MethodPatch,
			Path:     "/.services",
			Name:     "",
			Response: []byte("check-check"),
		}
		res, err := saveService(payload, appCfg)
		assert.Nil(res)
		assert.Equal(ErrReservedPrefix, err)
	})
}
