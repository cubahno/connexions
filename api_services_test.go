package connexions

import (
	"bytes"
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

func TestServiceHandler_save_formError(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	err = CreateServiceRoutes(router)
	assert.Nil(err)

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
}

func TestServiceHandler_save_saveError(t *testing.T) {
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
	_ = writer.WriteField("method", "X")
	_ = writer.Close()

	// serve
	req := httptest.NewRequest("POST", "/.services", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := *UnmarshallResponse[map[string]any](t, w.Body)
	assert.Equal(400, w.Code)
	assert.Equal(false, resp["success"].(bool))
	assert.Equal("invalid HTTP verb", resp["message"].(string))
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

func TestServiceHandler_resources_unknownService(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	err = CreateServiceRoutes(router)
	assert.Nil(err)

	req := httptest.NewRequest("GET", "/.services/"+RootServiceName, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := *UnmarshallResponse[map[string]any](t, w.Body)
	assert.Equal(http.StatusNotFound, w.Code)
	assert.Equal("application/json", w.Header().Get("Content-Type"))
	assert.Equal(false, resp["success"].(bool))
	assert.Equal("Service not found", resp["message"].(string))
}

func TestServiceHandler_resources(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

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

	err = CreateServiceRoutes(router)
	assert.Nil(err)

	req := httptest.NewRequest("GET", "/.services/petstore", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	expected := ServiceResourcesResponse{
		Endpoints: routes,
		Service: &ServiceEmbedded{
			Name: "petstore",
		},
		OpenAPISpecNames: []string{"index-pets.yml"},
	}

	resp := *UnmarshallResponse[ServiceResourcesResponse](t, w.Body)
	assert.Equal(http.StatusOK, w.Code)
	assert.Equal("application/json", w.Header().Get("Content-Type"))
	assert.Equal(expected, resp)
}

func TestServiceHandler_spec_unknownService(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	err = CreateServiceRoutes(router)
	assert.Nil(err)

	req := httptest.NewRequest("GET", "/.services/"+RootServiceName+"/spec", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := *UnmarshallResponse[map[string]any](t, w.Body)
	assert.Equal(http.StatusNotFound, w.Code)
	assert.Equal("Service not found", resp["message"].(string))
}

func TestServiceHandler_spec_noSpecFilesAttached(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	router.Services = map[string]*ServiceItem{
		"petstore": {
			Name: "petstore",
		},
	}

	err = CreateServiceRoutes(router)
	assert.Nil(err)

	req := httptest.NewRequest("GET", "/.services/petstore/spec", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := *UnmarshallResponse[map[string]any](t, w.Body)
	assert.Equal(http.StatusNotFound, w.Code)
	assert.Equal("No Spec files attached", resp["message"].(string))
}

func TestServiceHandler_spec_errorReadingSpec(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

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

	err = CreateServiceRoutes(router)
	assert.Nil(err)

	req := httptest.NewRequest("GET", "/.services/petstore/spec", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := *UnmarshallResponse[map[string]any](t, w.Body)
	assert.Equal(http.StatusInternalServerError, w.Code)
	assert.Equal("open : no such file or directory", resp["message"].(string))
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
