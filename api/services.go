package api

import (
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

const RootServiceName = "--"

func CreateServiceRoutes(router *Router) error {
	handler := &ServiceHandler{
		router: router,
	}

	router.Get("/services", handler.list)
	router.Post("/services", handler.create)
	router.Get("/services/{name}", handler.home)
	router.Get("/services/{name}/spec", handler.spec)
	router.Post("/services/{name}", handler.generate)

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
	Name      string        `json:"name"`
	IsOpenAPI bool          `json:"isOpenApi"`
	Method    string        `json:"method"`
	Path      string        `json:"path"`
	Response  []byte        `json:"response"`
	File      *UploadedFile `json:"file"`
}

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
	mu    sync.Mutex
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

func (h *ServiceHandler) create(w http.ResponseWriter, r *http.Request) {
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
		Name:      r.FormValue("name"),
		IsOpenAPI: r.FormValue("isOpenApi") == "true",
		Method:    r.FormValue("method"),
		Path:      r.FormValue("path"),
		Response:  []byte(r.FormValue("response")),
		File:      uploadedFile,
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
			File:   fileProps,
		}
		h.router.Services[fileProps.ServiceName] = service
	} else {
		h.router.Services[fileProps.ServiceName].Routes = append(h.router.Services[fileProps.ServiceName].Routes, routes...)
		if doc != nil {
			h.router.Services[fileProps.ServiceName].Spec = doc
		}
	}

	h.success("Service created", w)
}

func (h *ServiceHandler) home(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == RootServiceName {
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
	if name == RootServiceName {
		name = ""
	}

	if service == nil {
		NewJSONResponse(http.StatusNotFound, "Service not found", w)
		return
	}

	fileProps := service.File
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
	if name == RootServiceName {
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

	if !payload.IsOpenAPI {
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
	fileProps := service.File

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

func saveService(payload *ServicePayload) (*FileProperties, error) {
	uploadedFile := payload.File
	service := payload.Name
	content := payload.Response
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

	if service == "" && path == "" {
		return nil, ErrInvalidURLResource
	}

	ext := ""
	if uploadedFile != nil {
		ext = uploadedFile.Extension
		content = uploadedFile.Content
	}

	if ext == "" && len(content) > 0 {
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
