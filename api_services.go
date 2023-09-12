package connexions

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
)

func CreateServiceRoutes(router *Router) error {
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
		if !router.Config.App.DisableSwaggerUI {
			r.Get("/{name}/spec", handler.spec)
		}
		r.Post("/{name}/{id}", handler.generate)
		r.Delete("/{name}", handler.deleteService)
		r.Get("/{name}/resources/{id}", handler.getResource)
		r.Delete("/{name}/resources/{id}", handler.deleteResource)
	})

	return nil
}

type ServiceItem struct {
	Name         string            `json:"name"`
	Routes       RouteDescriptions `json:"routes"`
	OpenAPIFiles []*FileProperties `json:"-"`
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

func (i *ServiceItem) AddRoutes(routes RouteDescriptions) {
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
	OpenAPIRouteType = "openapi"
	FixedRouteType   = "fixed"
)

// RouteDescription describes a route for the UI Application.
// Path is relative to the service prefix.
type RouteDescription struct {
	Method      string          `json:"method"`
	Path        string          `json:"path"`
	Type        string          `json:"type"`
	ContentType string          `json:"contentType"`
	Overwrites  bool            `json:"overwrites"`
	File        *FileProperties `json:"-"`
}

type RouteDescriptions []*RouteDescription

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

	NewAPIJSONResponse(http.StatusOK, res, w)
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

	isOpenAPI := r.FormValue("isOpenApi") == "true"

	payload := &ServicePayload{
		Name:        r.FormValue("name"),
		IsOpenAPI:   isOpenAPI,
		Method:      r.FormValue("method"),
		Path:        r.FormValue("path"),
		Response:    []byte(r.FormValue("response")),
		ContentType: r.FormValue("contentType"),
		File:        uploadedFile,
	}

	fileProps, err := saveService(payload, h.router.Config.App)
	if err != nil {
		h.error(http.StatusBadRequest, err.Error(), w)
		return
	}

	var routes RouteDescriptions

	if isOpenAPI {
		routes, err = RegisterOpenAPIRoutes(fileProps, h.router)
	} else {
		routes, err = RegisterFixedService(fileProps, h.router)
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
		if len(h.router.Services) == 0 {
			h.router.Services = make(map[string]*ServiceItem)
		}
		h.router.Services[fileProps.ServiceName] = service
	} else {
		var addRoutes []*RouteDescription
		for _, route := range routes {
			for _, rd := range h.router.Services[fileProps.ServiceName].Routes {
				if rd.Path == route.Path && rd.Method == route.Method {
					route.Overwrites = true
					break
				}
			}
			addRoutes = append(addRoutes, route)
		}

		h.router.Services[fileProps.ServiceName].Routes = append(
			h.router.Services[fileProps.ServiceName].Routes, addRoutes...)
	}

	if isOpenAPI {
		service.AddOpenAPIFile(fileProps)
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
		NewAPIJSONResponse(http.StatusNotFound, "Service not found", w)
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

	NewAPIJSONResponse(http.StatusOK, res, w)
}

func (h *ServiceHandler) spec(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	service := h.router.Services[name]
	if name == RootServiceName {
		name = ""
	}

	if service == nil {
		NewAPIJSONResponse(http.StatusNotFound, "Service not found", w)
		return
	}

	openAPIFiles := service.OpenAPIFiles
	if len(openAPIFiles) == 0 {
		NewAPIJSONResponse(http.StatusNotFound, "No Spec files attached", w)
		return
	}

	// TODO(igor): handle multiple spec files in the UI
	fileProps := openAPIFiles[0]
	if fileProps == nil {
		NewAPIJSONResponse(http.StatusNotFound, "Service spec not found", w)
		return
	}

	content, err := os.ReadFile(fileProps.FilePath)
	if err != nil {
		NewAPIJSONResponse(http.StatusInternalServerError, NewErrorMessage(err), w)
		return
	}

	NewAPIResponse(http.StatusOK, content, w, SetAPIResponseContentType("text/plain"))
}

func (h *ServiceHandler) generate(w http.ResponseWriter, r *http.Request) {
	payload, err := GetJSONPayload[ResourceGeneratePayload](r)
	if err != nil {
		NewAPIJSONResponse(http.StatusBadRequest, NewErrorMessage(err), w)
		return
	}

	name := chi.URLParam(r, "name")
	if name == RootServiceName {
		name = ""
	}

	service := h.router.Services[name]
	if service == nil {
		NewAPIJSONResponse(http.StatusNotFound, "Service not found", w)
		return
	}

	ix, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		h.error(400, "Invalid resource id", w)
		return
	}

	isValidIx := ix >= 0 && ix < len(service.Routes)
	if !isValidIx {
		h.error(404, "Resource not found", w)
		return
	}

	rd := service.Routes[ix]
	fileProps := rd.File

	config := h.router.Config
	serviceCfg := config.GetServiceConfig(name)

	res := map[string]any{}

	if fileProps == nil {
		NewAPIJSONResponse(http.StatusNotFound, NewErrorMessage(ErrResourceNotFound), w)
		return
	}

	serviceCtxs := serviceCfg.Contexts
	if len(serviceCtxs) == 0 {
		serviceCtxs = h.router.ContextNames
	}

	contexts := CollectContexts(serviceCtxs, h.router.Contexts, payload.Replacements)

	replaceResource := &Resource{
		Service:           name,
		Path:              rd.Path,
		ContextData:       contexts,
		ContextAreaPrefix: config.App.ContextAreaPrefix,
	}
	var valueReplacer ValueReplacer
	if fileProps.ValueReplacerFactory != nil {
		valueReplacer = fileProps.ValueReplacerFactory(replaceResource)
	}

	if !fileProps.IsOpenAPI {
		res["request"] = newRequestFromFixedResource(
			fileProps.Prefix+fileProps.Resource,
			fileProps.Method,
			fileProps.ContentType,
			valueReplacer)
		res["response"] = newResponseFromFixedResource(fileProps.FilePath, fileProps.ContentType, valueReplacer)

		NewAPIJSONResponse(http.StatusOK, res, w)
		return
	}

	spec := fileProps.Spec
	operation := spec.FindOperation(&FindOperationOptions{
		Service:  name,
		Resource: rd.Path,
		Method:   strings.ToUpper(rd.Method),
	})
	if operation == nil {
		NewAPIJSONResponse(http.StatusMethodNotAllowed, NewErrorMessage(ErrResourceMethodNotFound), w)
	}
	operation = operation.WithParseConfig(serviceCfg.ParseConfig)

	req := NewRequestFromOperation(fileProps.Prefix, rd.Path, rd.Method, operation, valueReplacer)

	res["request"] = req
	res["response"] = NewResponseFromOperation(operation, valueReplacer)

	NewAPIJSONResponse(http.StatusOK, res, w)
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

	if err := deleteService(service, h.router.Config.App.Paths); err != nil {
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

	ix, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		h.error(400, "Invalid resource id", w)
		return
	}

	isValidIx := ix >= 0 && ix < len(service.Routes)
	if !isValidIx {
		h.error(404, "Resource not found", w)
		return
	}

	rd := service.Routes[ix]

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
	NewAPIJSONResponse(http.StatusOK, res, w)
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

	ix, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		h.error(400, "Invalid resource id", w)
		return
	}

	isValidIx := ix >= 0 && ix < len(service.Routes)
	if !isValidIx {
		h.error(404, "Resource not found", w)
		return
	}

	rd := service.Routes[ix]

	if err := os.Remove(rd.File.FilePath); err != nil {
		h.error(500, err.Error(), w)
		return
	}
	service.Routes = SliceDeleteAtIndex[*RouteDescription](service.Routes, ix)

	h.success(fmt.Sprintf("Resource %s deleted!", rd.Path), w)
}

type ServiceDescription struct {
	Name      string
	Method    string
	Path      string
	Ext       string
	IsOpenAPI bool
}

func saveService(payload *ServicePayload, appCfg *AppConfig) (*FileProperties, error) {
	prefixValidator := appCfg.IsValidPrefix
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

	descr := &ServiceDescription{
		Name:      service,
		Method:    method,
		Path:      path,
		Ext:       ext,
		IsOpenAPI: payload.IsOpenAPI,
	}
	filePath := ComposeFileSavePath(descr, appCfg.Paths)

	if payload.IsOpenAPI && len(content) == 0 {
		return nil, ErrOpenAPISpecIsEmpty
	}

	if err := SaveFile(filePath, content); err != nil {
		return nil, err
	}

	fileProps, err := GetPropertiesFromFilePath(filePath, appCfg)
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

func deleteService(service *ServiceItem, paths *Paths) error {
	var targets []string

	name := service.Name
	if name == "" {
		targets = append(targets, paths.ServicesOpenAPI)
		targets = append(targets, paths.ServicesFixedRoot)
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
