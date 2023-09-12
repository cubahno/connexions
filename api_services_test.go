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
