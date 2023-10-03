package api

import (
	"fmt"
	"github.com/cubahno/connexions"
	"github.com/cubahno/connexions/internal"
	"github.com/go-chi/chi/v5"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	OpenAPIRouteType = "openapi"
	FixedRouteType   = "fixed"
)

// RouteDescription describes a route for the UI Application.
// Path is relative to the service prefix.
type RouteDescription struct {
	Method      string                     `json:"method"`
	Path        string                     `json:"path"`
	Type        string                     `json:"type"`
	ContentType string                     `json:"contentType"`
	Overwrites  bool                       `json:"overwrites"`
	File        *connexions.FileProperties `json:"-"`
}

// RouteDescriptions is a slice of RouteDescription.
// Allows to add custom methods.
type RouteDescriptions []*RouteDescription

// ServiceHandler handles service routes.
type ServiceHandler struct {
	*BaseHandler
	router *Router
	mu     sync.Mutex
}

// createServiceRoutes adds service routes to the router.
// It also creates necessary closures to serve routes.
// Implements RouteRegister interface.
func createServiceRoutes(router *Router) error {
	if router.Config.App.DisableUI || router.Config.App.ServiceURL == "" {
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
		r.Delete("/{name}", handler.deleteService)
		if !router.Config.App.DisableSwaggerUI {
			r.Get("/{name}/spec", handler.spec)
		}
		r.Get("/{name}/{id}", handler.getResource)
		r.Post("/{name}/{id}", handler.generate)
		r.Delete("/{name}/{id}", handler.deleteResource)
	})

	return nil
}

// ServiceItem represents a service with the route collection.
// Service can hold multiple OpenAPI specs.
type ServiceItem struct {
	Name         string                       `json:"name"`
	Routes       RouteDescriptions            `json:"routes"`
	OpenAPIFiles []*connexions.FileProperties `json:"-"`
	mu           sync.Mutex
}

// AddOpenAPIFile adds OpenAPI file to the service.
func (i *ServiceItem) AddOpenAPIFile(file *connexions.FileProperties) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if len(i.OpenAPIFiles) == 0 {
		i.OpenAPIFiles = make([]*connexions.FileProperties, 0)
	}

	for _, f := range i.OpenAPIFiles {
		if file.IsEqual(f) {
			return
		}
	}

	i.OpenAPIFiles = append(i.OpenAPIFiles, file)
}

// AddRoutes adds routes to the service.
// There's no check for duplicates.
func (i *ServiceItem) AddRoutes(routes RouteDescriptions) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if len(i.Routes) == 0 {
		i.Routes = make([]*RouteDescription, 0)
	}

	for _, route := range routes {
		if route.Type != FixedRouteType {
			continue
		}
		// check if the route overwrites other routes
		for _, r := range i.Routes {
			if r.Path == route.Path && r.Method == route.Method {
				route.Overwrites = true
				break
			}
		}
	}

	i.Routes = append(i.Routes, routes...)
}

// Sort sorts the routes by path and method.
// The order is: GET, POST, other methods (alphabetically)
func (rs RouteDescriptions) Sort() {
	sort.SliceStable(rs, func(i, j int) bool {
		a := rs[i]
		b := rs[j]

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
}

func (h *ServiceHandler) list(w http.ResponseWriter, r *http.Request) {
	services := h.router.GetServices()
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

	h.JSONResponse(w).Send(res)
}

// Save service resource.
func (h *ServiceHandler) save(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	err := r.ParseMultipartForm(10 * 1024 * 1024) // Limit form size to 10 MB
	if err != nil {
		h.Error(http.StatusBadRequest, err.Error(), w)
		return
	}

	uploadedFile, err := connexions.GetRequestFile(r, "file")
	if err != nil {
		h.Error(http.StatusBadRequest, err.Error(), w)
		return
	}

	isOpenAPI := r.FormValue("isOpenApi") == "true"

	payload := &ServicePayload{
		IsOpenAPI:   isOpenAPI,
		Method:      r.FormValue("method"),
		Path:        r.FormValue("path"),
		Response:    []byte(r.FormValue("response")),
		ContentType: r.FormValue("contentType"),
		File:        uploadedFile,
		URL:         r.FormValue("url"),
	}

	fileProps, err := saveService(payload, h.router.Config.App)
	if err != nil {
		h.Error(http.StatusBadRequest, err.Error(), w)
		return
	}

	// if routes exists, do not add them again
	service, serviceExists := h.router.GetServices()[fileProps.ServiceName]
	if serviceExists {
		savedRouteId := h.getRouteIndex(fileProps)
		if savedRouteId >= 0 {
			h.JSONResponse(w).Send(&SavedResourceResponse{
				Message: "Resource saved!",
				Success: true,
				ID:      savedRouteId,
			})
			return
		}
	}

	var routes RouteDescriptions

	if isOpenAPI {
		routes = registerOpenAPIRoutes(fileProps, h.router)
	} else {
		route := registerFixedRoute(fileProps, h.router)
		routes = RouteDescriptions{route}
	}

	if !serviceExists {
		service = &ServiceItem{
			Name:   fileProps.ServiceName,
			Routes: routes,
		}
		h.router.AddService(service)
	} else {
		var newRoutes []*RouteDescription
		for _, route := range routes {
			newRoutes = append(newRoutes, route)
		}

		service.AddRoutes(newRoutes)
	}

	if isOpenAPI {
		service.AddOpenAPIFile(fileProps)
	}

	h.JSONResponse(w).Send(&SavedResourceResponse{
		Message: "Resource saved!",
		Success: true,
		ID:      h.getRouteIndex(fileProps),
	})
}

// resources returns the list of resources for the service.
func (h *ServiceHandler) resources(w http.ResponseWriter, r *http.Request) {
	service := h.getService(r)
	if service == nil {
		h.Error(http.StatusNotFound, "Service not found", w)
		return
	}

	routes := service.Routes

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

	h.JSONResponse(w).Send(res)
}

// deleteService deletes the service and all files.
func (h *ServiceHandler) deleteService(w http.ResponseWriter, r *http.Request) {
	service := h.getService(r)
	if service == nil {
		h.JSONResponse(w).WithStatusCode(http.StatusNotFound).Send(&SimpleResponse{
			Message: ErrServiceNotFound.Error(),
			Success: false,
		})
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	var targets []string
	paths := h.router.Config.App.Paths
	name := service.Name
	if name == "" {
		targets = append(targets, paths.ServicesOpenAPI)
		targets = append(targets, paths.ServicesFixedRoot)
	}

	for _, route := range service.Routes {
		targets = append(targets, route.File.FilePath)
	}

	for _, targetDir := range targets {
		// errors not important here and can be ignored
		_ = os.RemoveAll(targetDir)
	}

	h.router.RemoveService(service.Name)

	h.JSONResponse(w).Send(&SimpleResponse{
		Message: "Service deleted!",
		Success: true,
	})
}

// spec serves the OpenAPI spec file.
func (h *ServiceHandler) spec(w http.ResponseWriter, r *http.Request) {
	service := h.getService(r)
	if service == nil {
		h.Error(http.StatusNotFound, "Service not found", w)
		return
	}

	// TODO(cubahno): handle multiple spec files in the UI
	openAPIFiles := service.OpenAPIFiles
	if len(openAPIFiles) == 0 || openAPIFiles[0] == nil {
		h.Error(http.StatusNotFound, "No Spec files attached", w)
		return
	}

	fileProps := openAPIFiles[0]
	content, err := os.ReadFile(fileProps.FilePath)
	if err != nil {
		h.Error(http.StatusInternalServerError, err.Error(), w)
		return
	}

	NewAPIResponse(w).WithHeader("content-type", "text/plain").Send(content)
}

// generate generates a request and response from the OpenAPI spec.
func (h *ServiceHandler) generate(w http.ResponseWriter, r *http.Request) {
	service := h.getService(r)
	if service == nil {
		h.Error(http.StatusNotFound, ErrServiceNotFound.Error(), w)
		return
	}

	ix, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || (ix < 0 || ix >= len(service.Routes)) {
		h.Error(http.StatusNotFound, ErrResourceNotFound.Error(), w)
		return
	}

	payload, err := GetJSONPayload[ResourceGeneratePayload](r)
	if err != nil {
		h.Error(http.StatusBadRequest, err.Error(), w)
		return
	}

	rd := service.Routes[ix]
	fileProps := rd.File

	config := h.router.Config
	serviceCfg := config.GetServiceConfig(service.Name)

	res := &GenerateResponse{}

	if fileProps == nil {
		h.Error(http.StatusNotFound, ErrResourceNotFound.Error(), w)
		return
	}

	// create ValueReplacer
	serviceCtxs := serviceCfg.Contexts
	if len(serviceCtxs) == 0 {
		serviceCtxs = h.router.GetDefaultContexts()
	}
	contexts := connexions.CollectContexts(serviceCtxs, h.router.GetContexts(), payload.Replacements)
	valueReplacer := connexions.CreateValueReplacer(config, contexts)

	if !fileProps.IsOpenAPI {
		res.Request = connexions.NewRequestFromFixedResource(
			fileProps.Prefix+fileProps.Resource,
			fileProps.Method,
			fileProps.ContentType,
			valueReplacer)
		res.Response = connexions.NewResponseFromFixedResource(fileProps.FilePath, fileProps.ContentType, valueReplacer)

		h.JSONResponse(w).Send(res)
		return
	}

	spec := fileProps.Spec
	operation := spec.FindOperation(&connexions.OperationDescription{
		Service:  service.Name,
		Resource: rd.Path,
		Method:   strings.ToUpper(rd.Method),
	})

	if operation == nil {
		h.Error(http.StatusMethodNotAllowed, ErrResourceMethodNotFound.Error(), w)
		return
	}
	operation = operation.WithParseConfig(serviceCfg.ParseConfig)

	req := connexions.NewRequestFromOperation(fileProps.Prefix, rd.Path, rd.Method, operation, valueReplacer)

	res.Request = req
	res.Response = connexions.NewResponseFromOperation(r, operation, valueReplacer)

	h.JSONResponse(w).Send(res)
}

// getResource returns the resource contents for editing.
// Only fixed resources are allowed to be edited since they represent single resource in comparison to OpemAPI spec.
func (h *ServiceHandler) getResource(w http.ResponseWriter, r *http.Request) {
	svcErr := &SimpleResponse{
		Message: ErrServiceNotFound.Error(),
		Success: false,
	}
	resErr := &SimpleResponse{
		Message: ErrResourceNotFound.Error(),
		Success: false,
	}

	service := h.getService(r)
	if service == nil {
		h.JSONResponse(w).WithStatusCode(http.StatusNotFound).Send(svcErr)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	ix, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil ||
		(ix < 0 || ix >= len(service.Routes)) ||
		service.Routes[ix] == nil ||
		service.Routes[ix].File == nil {
		h.JSONResponse(w).WithStatusCode(http.StatusNotFound).Send(resErr)
		return
	}

	rd := service.Routes[ix]
	if rd.Type != FixedRouteType {
		h.JSONResponse(w).WithStatusCode(http.StatusBadRequest).Send(&SimpleResponse{
			Message: ErrOnlyFixedResourcesAllowedEditing.Error(),
			Success: false,
		})
		return
	}

	content, err := os.ReadFile(rd.File.FilePath)
	if err != nil {
		h.JSONResponse(w).WithStatusCode(http.StatusInternalServerError).Send(&SimpleResponse{
			Message: err.Error(),
			Success: false,
		})
		return
	}

	h.JSONResponse(w).Send(&ResourceResponse{
		Method:      rd.Method,
		Path:        rd.File.Prefix + rd.File.Resource,
		Extension:   strings.TrimPrefix(rd.File.Extension, "."),
		ContentType: rd.File.ContentType,
		Content:     string(content),
	})
}

// deleteResource deletes the resource file.
func (h *ServiceHandler) deleteResource(w http.ResponseWriter, r *http.Request) {
	service := h.getService(r)
	if service == nil {
		h.JSONResponse(w).WithStatusCode(http.StatusNotFound).Send(&SimpleResponse{
			Message: ErrServiceNotFound.Error(),
			Success: false,
		})
		return
	}

	ix, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || (ix < 0 || ix >= len(service.Routes)) {
		h.JSONResponse(w).WithStatusCode(http.StatusNotFound).Send(&SimpleResponse{
			Message: ErrResourceNotFound.Error(),
			Success: false,
		})
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	rd := service.Routes[ix]
	if rd.Type != FixedRouteType {
		h.JSONResponse(w).WithStatusCode(http.StatusBadRequest).Send(&SimpleResponse{
			Message: ErrOnlyFixedResourcesAllowedEditing.Error(),
			Success: false,
		})
		return
	}

	if rd.File != nil && rd.File.FilePath != "" {
		if err = os.Remove(rd.File.FilePath); err != nil {
			h.JSONResponse(w).WithStatusCode(http.StatusInternalServerError).Send(&SimpleResponse{
				Message: err.Error(),
				Success: false,
			})
			return
		}
	}

	service.Routes = internal.SliceDeleteAtIndex[*RouteDescription](service.Routes, ix)

	h.JSONResponse(w).Send(&SimpleResponse{
		Message: "Resource deleted!",
		Success: true,
	})
}

// getService returns the service by name from the path.
func (h *ServiceHandler) getService(r *http.Request) *ServiceItem {
	h.mu.Lock()
	defer h.mu.Unlock()

	name := chi.URLParam(r, "name")
	if name == connexions.RootServiceName {
		name = ""
	}

	return h.router.GetServices()[name]
}

func (h *ServiceHandler) getRouteIndex(fileProps *connexions.FileProperties) int {
	service, ok := h.router.GetServices()[fileProps.ServiceName]
	if !ok {
		return -1
	}

	routes := service.Routes
	routes.Sort()

	filePropsType := FixedRouteType
	if fileProps.IsOpenAPI {
		filePropsType = OpenAPIRouteType
	}

	for ix, route := range routes {
		if route.Type == filePropsType && route.Path == fileProps.Resource && route.Method == fileProps.Method {
			return ix
		}
	}
	return -1
}

// saveService saves the service resource.
func saveService(payload *ServicePayload, appCfg *connexions.AppConfig) (*connexions.FileProperties, error) {
	prefixValidator := appCfg.IsValidPrefix
	uploadedFile := payload.File
	content := payload.Response
	contentType := payload.ContentType
	method := strings.ToUpper(payload.Method)
	path := "/" + strings.Trim(payload.Path, "/")

	if method != "" && !internal.IsValidHTTPVerb(method) {
		return nil, ErrInvalidHTTPVerb
	}
	if method == "" {
		method = http.MethodGet
	}

	if !internal.IsValidURLResource(path) {
		return nil, ErrInvalidURLResource
	}

	ext := ""
	if contentType != "" {
		ext = "." + contentType
	}

	if uploadedFile != nil {
		ext = uploadedFile.Extension
		content = uploadedFile.Content
	} else if payload.URL != "" {
		var err error

		// TODO(cubahno): move client to the handler and use config timeout
		client := &http.Client{
			Timeout: 10 * time.Second,
		}

		content, _, err = connexions.GetFileContentsFromURL(client, payload.URL)
		if err != nil {
			return nil, err
		}
	}

	// ignore supplied extension and check whether its json / yaml type
	if len(content) > 0 {
		if connexions.IsJsonType(content) {
			ext = ".json"
		} else if connexions.IsYamlType(content) {
			ext = ".yaml"
		}
	}

	descr := &ServiceDescription{
		Method:    method,
		Path:      path,
		Ext:       ext,
		IsOpenAPI: payload.IsOpenAPI,
	}
	filePath := ComposeFileSavePath(descr, appCfg.Paths)

	if payload.IsOpenAPI && len(content) == 0 {
		return nil, ErrOpenAPISpecIsEmpty
	}

	if err := connexions.SaveFile(filePath, content); err != nil {
		return nil, err
	}

	fileProps, err := connexions.GetPropertiesFromFilePath(filePath, appCfg)
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

type ServiceItemResponse struct {
	Name             string   `json:"name"`
	OpenAPIResources []string `json:"openApiResources"`
}

// ServicePayload is a struct that represents a new service payload.
type ServicePayload struct {
	IsOpenAPI   bool                     `json:"isOpenApi"`
	Method      string                   `json:"method"`
	Path        string                   `json:"path"`
	Response    []byte                   `json:"response"`
	ContentType string                   `json:"contentType"`
	File        *connexions.UploadedFile `json:"file"`
	URL         string                   `json:"url"`
}

// ServiceDescription is a struct created from the service payload to facilitate file path composition.
type ServiceDescription struct {
	Method    string
	Path      string
	Ext       string
	IsOpenAPI bool
}

type GenerateResponse struct {
	Request  *connexions.Request  `json:"request"`
	Response *connexions.Response `json:"response"`
}

type ServiceListResponse struct {
	Items []*ServiceItemResponse `json:"items"`
}

type ServiceEmbedded struct {
	Name string `json:"name"`
}

type ServiceResourcesResponse struct {
	Service          *ServiceEmbedded  `json:"service"`
	Endpoints        RouteDescriptions `json:"endpoints"`
	OpenAPISpecNames []string          `json:"openapiSpecNames"`
}

type ResourceResponse struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	Extension   string `json:"extension"`
	ContentType string `json:"contentType"`
	Content     string `json:"content"`
}

type SavedResourceResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	ID      int    `json:"id"`
}

// ComposeFileSavePath composes a save path for a file.
func ComposeFileSavePath(descr *ServiceDescription, paths *connexions.Paths) string {
	if descr.IsOpenAPI {
		return ComposeOpenAPISavePath(descr, paths.ServicesOpenAPI)
	}

	resource := strings.Trim(descr.Path, "/")
	parts := strings.Split(resource, "/")

	method := descr.Method
	if method == "" {
		method = http.MethodGet
	}
	method = strings.ToLower(method)
	ext := descr.Ext
	pathExt := filepath.Ext(resource)

	// it shouldn't usually be the case, as we determine ext based on multiple factors
	if ext == "" {
		ext = pathExt
	}
	if ext == "" {
		ext = ".txt"
	}

	if resource == "" {
		return fmt.Sprintf("/%s/%s/index%s", paths.ServicesFixedRoot, method, ext)
	}

	service := ""
	if len(parts) == 1 {
		if pathExt != "" {
			service = connexions.RootServiceName
		} else {
			service = parts[0]
			parts = []string{}
		}

	} else {
		// first part is always service name
		service = parts[0]
		// remove service from it
		parts = parts[1:]
	}

	// we have a service and a path without it now

	res := paths.Services
	res += "/" + service
	res += "/" + method
	res += "/" + strings.Join(parts, "/")
	res = strings.TrimSuffix(res, "/")

	if pathExt == "" {
		res += "/index" + ext
	}

	return res
}

// ComposeOpenAPISavePath composes a save path for an OpenAPI specification.
// The resulting filename is always index.<spec extension>.
func ComposeOpenAPISavePath(descr *ServiceDescription, baseDir string) string {
	resource := strings.Trim(descr.Path, "/")
	resourceParts := strings.Split(resource, "/")
	ext := descr.Ext

	service := ""
	if len(resourceParts) > 0 {
		// take the first part as service name and exclude it from resource parts
		service = resourceParts[0]
		resourceParts = resourceParts[1:]
	}

	result := baseDir
	if service != "" {
		result += "/" + service
	}

	resPart := "/" + strings.Join(resourceParts, "/")
	resPart = strings.TrimSuffix(resPart, "/") + "/index"
	result += resPart + ext

	return result
}
