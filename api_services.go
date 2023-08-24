package connexions

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
)

func CreateServiceRoutes(router *Router) error {
	if !router.Config.App.ServeUI || router.Config.App.ServiceURL == "" {
		return nil
	}

	handler := &ServiceHandler{
		router: router,
	}

	url := router.Config.App.ServiceURL
	url = "/" + strings.Trim(url, "/")
	log.Printf("Mounting service URLs at %s\n", url)

	router.Route(url, func(r chi.Router) {
		r.Get("/", handler.list)
		r.Post("/", handler.save)
		r.Get("/{name}", handler.resources)
		if router.Config.App.ServeSpec {
			r.Get("/{name}/spec", handler.spec)
		}
		r.Post("/{name}", handler.generate)
		r.Delete("/{name}", handler.deleteService)
		r.Get("/{name}/resources/{method}", handler.getResource)
		r.Delete("/{name}/resources/{method}", handler.deleteResource)
	})

	return nil
}

type ServiceItem struct {
	Name         string              `json:"name"`
	Routes       []*RouteDescription `json:"routes"`
	OpenAPIFiles []*FileProperties   `json:"-"`
	mu           sync.Mutex
}

func (i *ServiceItem) AddOpenAPIFile(file *FileProperties) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if len(i.OpenAPIFiles) == 0 {
		i.OpenAPIFiles = make([]*FileProperties, 0)
	}

	for _, f := range i.OpenAPIFiles {
		if file.IsEqual(f) {
			return
		}
	}

	i.OpenAPIFiles = append(i.OpenAPIFiles, file)
}

func (i *ServiceItem) AddRoutes(routes []*RouteDescription) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if len(i.Routes) == 0 {
		i.Routes = make([]*RouteDescription, 0)
	}
	i.Routes = append(i.Routes, routes...)
}

type ServiceItemResponse struct {
	Name             string   `json:"name"`
	OpenAPIResources []string `json:"openApiResources"`
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

const (
	OpenAPIRouteType   = "openapi"
	OverwriteRouteType = "overwrite"
)

// RouteDescription describes a route for the UI Application.
// Path is relative to the service prefix.
type RouteDescription struct {
	Method      string          `json:"method"`
	Path        string          `json:"path"`
	Type        string          `json:"type"`
	ContentType string          `json:"contentType"`
	File        *FileProperties `json:"-"`
}

type ServiceListResponse struct {
	Items []*ServiceItemResponse `json:"items"`
}

type ServiceEmbedded struct {
	Name string `json:"name"`
}

type ServiceResourcesResponse struct {
	Service          *ServiceEmbedded    `json:"service"`
	Endpoints        []*RouteDescription `json:"endpoints"`
	OpenAPISpecNames []string            `json:"openapiSpecNames"`
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
		var openApiNames []string
		for _, file := range svcItem.OpenAPIFiles {
			openApiNames = append(openApiNames, file.Prefix)
		}

		items = append(items, &ServiceItemResponse{
			Name:             key,
			OpenAPIResources: openApiNames,
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

	err := r.ParseMultipartForm(10 * 1024 * 1024) // Limit form size to 10 MB
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

	fileProps, err := saveService(payload, h.router.Config.App.IsValidPrefix)
	if err != nil {
		h.error(http.StatusBadRequest, err.Error(), w)
		return
	}

	var routes []*RouteDescription

	if payload.IsOpenAPI {
		routes, err = RegisterOpenAPIRoutes(fileProps, h.router)
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
	}

	h.success("Resource saved!", w)
}

func (h *ServiceHandler) resources(w http.ResponseWriter, r *http.Request) {
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
		a := routes[i]
		b := routes[j]

		if a.Path != b.Path {
			return a.Path < b.Path
		}
		methodOrder := map[string]int{
			http.MethodGet:  1,
			http.MethodPost: 2,
			"default":       3,
		}
		m1 := methodOrder[a.Method]
		if m1 == 0 {
			m1 = methodOrder["default"]
		}

		m2 := methodOrder[b.Method]
		if m2 == 0 {
			m2 = methodOrder["default"]
		}

		return m1 < m2
	})

	names := make([]string, 0, len(service.OpenAPIFiles))
	for _, f := range service.OpenAPIFiles {
		names = append(names, f.Prefix)
	}

	res := &ServiceResourcesResponse{
		Endpoints:        routes,
		OpenAPISpecNames: names,
		Service: &ServiceEmbedded{
			Name: service.Name,
		},
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

	openAPIFiles := service.OpenAPIFiles
	if len(openAPIFiles) == 0 {
		NewJSONResponse(http.StatusNotFound, "No Spec files attached", w)
		return
	}

	// TODO(igor): handle multiple spec files in the UI
	fileProps := openAPIFiles[0]
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

	method := strings.ToUpper(payload.Method)
	if !IsValidHTTPVerb(method) {
		NewJSONResponse(http.StatusBadRequest, "Invalid method", w)
		return
	}

	service := h.router.Services[name]
	if service == nil {
		NewJSONResponse(http.StatusNotFound, "Service not found", w)
		return
	}

	config := h.router.Config
	serviceCfg := config.GetServiceConfig(name)

	res := map[string]any{}

	var fileProps *FileProperties
	for _, r := range service.Routes {
		if r.Method == method && r.Path == payload.Resource {
			fileProps = r.File
			break
		}
	}

	if fileProps == nil {
		NewJSONResponse(http.StatusNotFound, GetErrorResponse(ErrResourceNotFound), w)
		return
	}

	serviceCtxs := serviceCfg.Contexts
	if len(serviceCtxs) == 0 {
		serviceCtxs = h.router.ContextNames
	}

	contexts := CollectContexts(serviceCtxs, h.router.Contexts, payload.Replacements)

	replaceResource := &Resource{
		Service:           name,
		Path:              payload.Resource,
		ContextData:       contexts,
		ContextAreaPrefix: config.App.ContextAreaPrefix,
	}
	var valueReplacer ValueReplacer
	if fileProps.ValueReplacerFactory != nil {
		valueReplacer = fileProps.ValueReplacerFactory(replaceResource)
	}

	if !fileProps.IsOpenAPI {
		res["request"] = NewRequestFromFileProperties(
			fileProps.Prefix+fileProps.Resource,
			fileProps.Method,
			fileProps.ContentType,
			valueReplacer)
		res["response"] = NewResponseFromFileProperties(fileProps.FilePath, fileProps.ContentType, valueReplacer)

		NewJSONResponse(http.StatusOK, res, w)
		return
	}

	spec := fileProps.Spec
	operation := spec.FindOperation(payload.Resource, strings.ToUpper(payload.Method))
	if operation == nil {
		NewJSONResponse(http.StatusMethodNotAllowed, GetErrorResponse(ErrResourceMethodNotFound), w)
	}

	req := NewRequestFromOperation(fileProps.Prefix, payload.Resource, payload.Method, operation, valueReplacer)

	res["request"] = req
	res["response"] = NewResponseFromOperation(operation, valueReplacer)

	NewJSONResponse(http.StatusOK, res, w)
}

func (h *ServiceHandler) deleteService(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	name := chi.URLParam(r, "name")
	if name == RootServiceName {
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
	if name == RootServiceName {
		name = ""
	}

	service := h.router.Services[name]
	if service == nil {
		h.error(404, "Service not found", w)
		return
	}

	method := chi.URLParam(r, "method")
	if method == "" || !IsValidHTTPVerb(method) {
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
	res["extension"] = strings.TrimPrefix(rd.File.Extension, ".")
	res["contentType"] = rd.File.ContentType
	res["content"] = string(content)
	NewJSONResponse(http.StatusOK, res, w)
}

func (h *ServiceHandler) deleteResource(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	name := chi.URLParam(r, "name")
	if name == RootServiceName {
		name = ""
	}

	service := h.router.Services[name]
	if service == nil {
		h.error(404, "Service not found", w)
		return
	}

	method := chi.URLParam(r, "method")
	if method == "" || !IsValidHTTPVerb(method) {
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
			service.Routes = SliceDeleteAtIndex[*RouteDescription](service.Routes, i)
			break
		}
	}

	h.success(fmt.Sprintf("Resource %s deleted!", path), w)
}

func saveService(payload *ServicePayload, prefixValidator func(string) bool) (*FileProperties, error) {
	uploadedFile := payload.File
	service := payload.Name
	content := payload.Response
	contentType := payload.ContentType
	method := strings.ToUpper(payload.Method)
	path := "/" + strings.Trim(payload.Path, "/")

	if method != "" && !IsValidHTTPVerb(method) {
		return nil, ErrInvalidHTTPVerb
	}
	if method == "" {
		method = http.MethodGet
	}

	if !IsValidURLResource(path) {
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

	if err := SaveFile(filePath, content); err != nil {
		return nil, err
	}

	fileProps, err := GetPropertiesFromFilePath(filePath)
	if err != nil {
		_ = os.RemoveAll(filePath)
		return nil, err
	}
	if !prefixValidator(fileProps.Prefix) || !prefixValidator(path) {
		_ = os.RemoveAll(filePath)
		return nil, ErrReservedPrefix
	}

	return fileProps, nil
}

func deleteService(service *ServiceItem) error {
	var targets []string

	name := service.Name
	if name == "" {
		targets = append(targets, ServiceOpenAPIPath)
		targets = append(targets, ServiceRootPath)
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
