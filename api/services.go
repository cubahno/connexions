package api

import (
	"fmt"
	"github.com/cubahno/xs"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

func CreateServiceRoutes(router *Router) error {
	handler := &ServiceHandler{
		router: router,
	}

	router.Route("/services", func(r chi.Router) {
		r.Get("/", handler.list)
		r.Post("/", handler.save)
		r.Get("/{name}", handler.home)
		r.Get("/{name}/spec", handler.spec)
		r.Post("/{name}", handler.generate)
		r.Delete("/{name}", handler.deleteService)
		r.Get("/{name}/resources/{method}", handler.getResource)
		r.Delete("/{name}/resources/{method}", handler.deleteResource)
	})

	return nil
}

type ServiceItem struct {
	Name   string              `json:"name"`
	Routes []*RouteDescription `json:"routes"`
	Spec   *openapi3.T         `json:"-"`
	File   *FileProperties     `json:"-"`
}

type ServiceItemResponse struct {
	Name      string `json:"name"`
	IsOpenAPI bool   `json:"isOpenApi"`
}

type ServicePayload struct {
	Name        string        `json:"name"`
	IsOpenAPI   bool          `json:"isOpenApi"`
	Method      string        `json:"method"`
	Path        string        `json:"path"`
	Response    []byte        `json:"response"`
	ContentType string        `json:"contentType"`
	File        *UploadedFile `json:"file"`
}

// RouteDescription describes a route for the UI Application.
// Path is relative to the service prefix.
type RouteDescription struct {
	Method string          `json:"method"`
	Path   string          `json:"path"`
	Type   string          `json:"type"`
	File   *FileProperties `json:"-"`
}

type ServiceListResponse struct {
	Items []*ServiceItemResponse `json:"items"`
}

type ServiceHomeResponse struct {
	Endpoints []*RouteDescription `json:"endpoints"`
}

type ServiceHandler struct {
	*BaseHandler
	router *Router
	mu     sync.Mutex
}

func (h *ServiceHandler) list(w http.ResponseWriter, r *http.Request) {
	services := h.router.Services
	var keys []string
	for key := range services {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	items := make([]*ServiceItemResponse, 0)
	for _, key := range keys {
		svcItem := services[key]
		items = append(items, &ServiceItemResponse{
			Name:      key,
			IsOpenAPI: svcItem.Spec != nil,
		})
	}
	res := &ServiceListResponse{
		Items: items,
	}

	NewJSONResponse(http.StatusOK, res, w)
}

func (h *ServiceHandler) save(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	err := r.ParseMultipartForm(256 * 1024 * 1024) // Limit form size to 256 MB
	if err != nil {
		h.error(http.StatusBadRequest, err.Error(), w)
		return
	}

	uploadedFile, err := GetRequestFile(r, "file")
	if err != nil {
		h.error(http.StatusBadRequest, err.Error(), w)
		return
	}

	payload := &ServicePayload{
		Name:        r.FormValue("name"),
		IsOpenAPI:   r.FormValue("isOpenApi") == "true",
		Method:      r.FormValue("method"),
		Path:        r.FormValue("path"),
		Response:    []byte(r.FormValue("response")),
		ContentType: r.FormValue("contentType"),
		File:        uploadedFile,
	}

	fileProps, err := saveService(payload)
	if err != nil {
		h.error(http.StatusBadRequest, err.Error(), w)
		return
	}

	var doc *openapi3.T
	var routes []*RouteDescription

	if payload.IsOpenAPI {
		doc, routes, err = RegisterOpenAPIService(fileProps, h.router)
	} else {
		routes, err = RegisterOverwriteService(fileProps, h.router)
	}

	if err != nil {
		h.error(http.StatusBadRequest, err.Error(), w)
		return
	}

	service, ok := h.router.Services[fileProps.ServiceName]
	if !ok {
		service = &ServiceItem{
			Name:   fileProps.ServiceName,
			Routes: routes,
			Spec:   doc,
		}
		h.router.Services[fileProps.ServiceName] = service
	} else {
		var addRoutes []*RouteDescription
		for _, route := range routes {
			exists := false
			for _, rd := range h.router.Services[fileProps.ServiceName].Routes {
				if rd.Path == route.Path && rd.Method == route.Method {
					exists = true
					break
				}
			}
			if !exists {
				addRoutes = append(addRoutes, route)
			}
		}

		h.router.Services[fileProps.ServiceName].Routes = append(
			h.router.Services[fileProps.ServiceName].Routes, addRoutes...)

		if doc != nil {
			h.router.Services[fileProps.ServiceName].Spec = doc
		}
	}

	h.success("Resource saved!", w)
}

func (h *ServiceHandler) home(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == xs.RootServiceName {
		name = ""
	}
	service := h.router.Services[name]
	if service == nil {
		NewJSONResponse(http.StatusNotFound, "Service not found", w)
		return
	}

	routes := service.Routes

	sort.SliceStable(routes, func(i, j int) bool {
		if routes[i].Path != routes[j].Path {
			return routes[i].Path < routes[j].Path
		}
		methodOrder := map[string]int{
			"GET":     1,
			"POST":    2,
			"default": 3,
		}
		m1 := methodOrder[routes[i].Method]
		if m1 == 0 {
			m1 = methodOrder["default"]
		}

		m2 := methodOrder[routes[j].Method]
		if m2 == 0 {
			m2 = methodOrder["default"]
		}

		return m1 < m2
	})
	res := &ServiceHomeResponse{
		Endpoints: routes,
	}

	NewJSONResponse(http.StatusOK, res, w)
}

func (h *ServiceHandler) spec(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	service := h.router.Services[name]
	if name == xs.RootServiceName {
		name = ""
	}

	if service == nil {
		NewJSONResponse(http.StatusNotFound, "Service not found", w)
		return
	}

	// find first route with spec
	var fileProps *FileProperties
	for _, r := range service.Routes {
		if r.File != nil && r.File.IsOpenAPI {
			fileProps = r.File
			break
		}
	}

	if fileProps == nil {
		NewJSONResponse(http.StatusNotFound, "Service spec not found", w)
		return
	}

	content, err := os.ReadFile(fileProps.FilePath)
	if err != nil {
		NewJSONResponse(http.StatusInternalServerError, GetErrorResponse(err), w)
		return
	}

	NewResponse(http.StatusOK, content, w, WithContentType("text/plain"))
}

func (h *ServiceHandler) generate(w http.ResponseWriter, r *http.Request) {
	payload, err := GetPayload[ResourceGeneratePayload](r)
	if err != nil {
		NewJSONResponse(http.StatusBadRequest, GetErrorResponse(err), w)
		return
	}

	name := chi.URLParam(r, "name")
	if name == xs.RootServiceName {
		name = ""
	}

	service := h.router.Services[name]
	if service == nil {
		NewJSONResponse(http.StatusNotFound, "Service not found", w)
		return
	}

	// TODO(igor): move valueResolver to router
	valueResolver := xs.CreateValueResolver()
	jsonResolver := xs.CreateJSONResolver()
	res := map[string]any{}

	var fileProps *FileProperties
	for _, r := range service.Routes {
		if r.Method == payload.Method && r.Path == payload.Resource {
			fileProps = r.File
			break
		}
	}

	if fileProps == nil {
		NewJSONResponse(http.StatusNotFound, GetErrorResponse(ErrResourceNotFound), w)
		return
	}

	if !payload.IsOpenAPI {
		res["request"] = xs.NewRequestFromFileProperties(
			fileProps.Prefix+fileProps.Resource, fileProps.Method, fileProps.ContentType, jsonResolver)
		res["response"] = xs.NewResponseFromFileProperties(fileProps.FilePath, fileProps.ContentType, jsonResolver)
		NewJSONResponse(http.StatusOK, res, w)
		return
	}

	spec := service.Spec
	if spec == nil {
		NewJSONResponse(http.StatusNotFound, "Service spec not found", w)
		return
	}

	if !fileProps.IsOpenAPI {
		h.error(500, "OpenAPI spec not found", w)
	}

	// handle openapi resource
	pathItem := spec.Paths[payload.Resource]
	if pathItem == nil {
		NewJSONResponse(http.StatusNotFound, GetErrorResponse(ErrResourceNotFound), w)
		return
	}

	operation := pathItem.GetOperation(strings.ToUpper(payload.Method))
	if operation == nil {
		NewJSONResponse(http.StatusMethodNotAllowed, GetErrorResponse(ErrResourceMethodNotFound), w)
	}

	res["request"] = xs.NewRequestFromOperation(
		fileProps.Prefix, payload.Resource, payload.Method, operation, valueResolver)
	res["response"] = xs.NewResponseFromOperation(operation, valueResolver)

	NewJSONResponse(http.StatusOK, res, w)
}

func (h *ServiceHandler) deleteService(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	name := chi.URLParam(r, "name")
	if name == xs.RootServiceName {
		name = ""
	}

	service := h.router.Services[name]
	if service == nil {
		h.error(404, "Service not found", w)
		return
	}

	if err := deleteService(service); err != nil {
		h.error(500, err.Error(), w)
		return
	}

	delete(h.router.Services, name)

	h.success(fmt.Sprintf("Service %s deleted!", name), w)
}

func (h *ServiceHandler) getResource(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	name := chi.URLParam(r, "name")
	if name == xs.RootServiceName {
		name = ""
	}

	service := h.router.Services[name]
	if service == nil {
		h.error(404, "Service not found", w)
		return
	}

	method := chi.URLParam(r, "method")
	if method == "" || !xs.IsValidHTTPVerb(method) {
		h.error(400, "Invalid method", w)
		return
	}
	method = strings.ToUpper(method)

	path := r.URL.Query().Get("path")
	if path == "" {
		h.error(400, "Invalid path", w)
		return
	}

	var rd *RouteDescription
	for _, route := range service.Routes {
		if route.Method == method && route.Path == path {
			rd = route
			break
		}
	}

	if rd == nil {
		h.error(404, "Resource not found", w)
		return
	}

	if rd.File == nil {
		h.error(404, "Resource file not found", w)
		return
	}

	content, err := os.ReadFile(rd.File.FilePath)
	if err != nil {
		h.error(500, err.Error(), w)
		return
	}

	res := make(map[string]any)
	res["method"] = rd.Method
	res["path"] = rd.File.Prefix + rd.File.Resource
	res["contentType"] = strings.TrimPrefix(rd.File.Extension, ".")
	res["content"] = string(content)
	NewJSONResponse(http.StatusOK, res, w)
}

func (h *ServiceHandler) deleteResource(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	name := chi.URLParam(r, "name")
	if name == xs.RootServiceName {
		name = ""
	}

	service := h.router.Services[name]
	if service == nil {
		h.error(404, "Service not found", w)
		return
	}

	method := chi.URLParam(r, "method")
	if method == "" || !xs.IsValidHTTPVerb(method) {
		h.error(400, "Invalid method", w)
		return
	}
	method = strings.ToUpper(method)

	path := r.URL.Query().Get("path")
	if path == "" {
		h.error(400, "Invalid path", w)
		return
	}

	for i, route := range service.Routes {
		if route.Method == method && route.Path == path {
			if err := os.Remove(route.File.FilePath); err != nil {
				h.error(500, err.Error(), w)
				return
			}
			service.Routes = xs.SliceDeleteAtIndex[*RouteDescription](service.Routes, i)
			break
		}
	}

	h.success(fmt.Sprintf("Resource %s deleted!", path), w)
}

func saveService(payload *ServicePayload) (*FileProperties, error) {
	uploadedFile := payload.File
	service := payload.Name
	content := payload.Response
	contentType := payload.ContentType
	method := strings.ToUpper(payload.Method)
	path := payload.Path

	if method != "" && !xs.IsValidHTTPVerb(method) {
		return nil, ErrInvalidHTTPVerb
	}
	if method == "" {
		method = http.MethodGet
	}

	if !xs.IsValidURLResource(path) {
		return nil, ErrInvalidURLResource
	}

	// TODO(igor): check if doesn't match ui route
	if service == "" && path == "" {
		return nil, ErrInvalidURLResource
	}

	ext := ""
	if len(contentType) > 0 {
		ext = "." + contentType
	}

	if uploadedFile != nil {
		ext = uploadedFile.Extension
		content = uploadedFile.Content
	}

	// ignore supplied extension and check whether its json / yaml type
	if len(content) > 0 {
		if IsJsonType(content) {
			ext = ".json"
		} else if IsYamlType(content) {
			ext = ".yaml"
		}
	}

	filePath := ComposeFileSavePath(service, method, path, ext, payload.IsOpenAPI)

	if payload.IsOpenAPI && len(content) == 0 {
		return nil, ErrOpenAPISpecIsEmpty
	}

	dirPath := filepath.Dir(filePath)
	// Create directories recursively
	err := os.MkdirAll(dirPath, os.ModePerm)
	if err != nil {
		return nil, ErrCreatingDirectories
	}

	dest, err := os.Create(filePath)
	if err != nil {
		return nil, ErrCreatingFile
	}

	_, err = dest.Write(content)
	if err != nil {
		return nil, err
	}

	fileProps := GetPropertiesFromFilePath(filePath)
	return fileProps, nil
}

func deleteService(service *ServiceItem) error {
	var targets []string

	name := service.Name
	if name == "" {
		targets = append(targets, xs.ServiceOpenAPIPath)
		targets = append(targets, xs.ServiceRootPath)
	}

	for _, route := range service.Routes {
		targets = append(targets, route.File.FilePath)
	}

	for _, targetDir := range targets {
		err := os.RemoveAll(targetDir)
		if err != nil {
			return err
		}
	}

	return nil
}
