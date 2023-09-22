package connexions

import (
	"github.com/go-chi/chi/v5"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
)

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

	h.JSONResponse(w).Send(res)
}

// Save service resource.
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

	// if routes exists, do not add them again
	service, serviceExists := h.router.Services[fileProps.ServiceName]
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
		if len(h.router.Services) == 0 {
			h.router.Services = make(map[string]*ServiceItem)
		}
		h.router.Services[fileProps.ServiceName] = service
	} else {
		var addRoutes []*RouteDescription
		for _, route := range routes {
			addRoutes = append(addRoutes, route)
		}

		h.router.Services[fileProps.ServiceName].AddRoutes(addRoutes)
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

func (h *ServiceHandler) resources(w http.ResponseWriter, r *http.Request) {
	service := h.getService(r)
	if service == nil {
		h.error(http.StatusNotFound, "Service not found", w)
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

	delete(h.router.Services, service.Name)
	h.JSONResponse(w).Send(&SimpleResponse{
		Message: "Service deleted!",
		Success: true,
	})
}

func (h *ServiceHandler) spec(w http.ResponseWriter, r *http.Request) {
	service := h.getService(r)
	if service == nil {
		h.error(http.StatusNotFound, "Service not found", w)
		return
	}

	// TODO(cubahno): handle multiple spec files in the UI
	openAPIFiles := service.OpenAPIFiles
	if len(openAPIFiles) == 0 || openAPIFiles[0] == nil {
		h.error(http.StatusNotFound, "No Spec files attached", w)
		return
	}

	fileProps := openAPIFiles[0]
	content, err := os.ReadFile(fileProps.FilePath)
	if err != nil {
		h.error(http.StatusInternalServerError, err.Error(), w)
		return
	}

	NewAPIResponse(w).WithHeader("content-type", "text/plain").Send(content)
}

func (h *ServiceHandler) generate(w http.ResponseWriter, r *http.Request) {
	service := h.getService(r)
	if service == nil {
		h.error(http.StatusNotFound, ErrServiceNotFound.Error(), w)
		return
	}

	ix, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || (ix < 0 || ix >= len(service.Routes)) {
		h.error(http.StatusNotFound, ErrResourceNotFound.Error(), w)
		return
	}

	payload, err := GetJSONPayload[ResourceGeneratePayload](r)
	if err != nil {
		h.error(http.StatusBadRequest, err.Error(), w)
		return
	}

	rd := service.Routes[ix]
	fileProps := rd.File

	config := h.router.Config
	serviceCfg := config.GetServiceConfig(service.Name)

	res := &GenerateResponse{}

	if fileProps == nil {
		h.error(http.StatusNotFound, ErrResourceNotFound.Error(), w)
		return
	}

	// create ValueReplacer
	serviceCtxs := serviceCfg.Contexts
	if len(serviceCtxs) == 0 {
		serviceCtxs = h.router.ContextNames
	}
	contexts := CollectContexts(serviceCtxs, h.router.Contexts, payload.Replacements)
	valueReplacer := CreateValueReplacer(config, contexts)

	if !fileProps.IsOpenAPI {
		res.Request = newRequestFromFixedResource(
			fileProps.Prefix+fileProps.Resource,
			fileProps.Method,
			fileProps.ContentType,
			valueReplacer)
		res.Response = newResponseFromFixedResource(fileProps.FilePath, fileProps.ContentType, valueReplacer)

		h.JSONResponse(w).Send(res)
		return
	}

	spec := fileProps.Spec
	operation := spec.FindOperation(&OperationDescription{
		Service:  service.Name,
		Resource: rd.Path,
		Method:   strings.ToUpper(rd.Method),
	})

	if operation == nil {
		h.error(http.StatusMethodNotAllowed, ErrResourceMethodNotFound.Error(), w)
		return
	}
	operation = operation.WithParseConfig(serviceCfg.ParseConfig)

	req := NewRequestFromOperation(fileProps.Prefix, rd.Path, rd.Method, operation, valueReplacer)

	res.Request = req
	res.Response = NewResponseFromOperation(operation, valueReplacer)

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

	service.Routes = SliceDeleteAtIndex[*RouteDescription](service.Routes, ix)

	h.JSONResponse(w).Send(&SimpleResponse{
		Message: "Resource deleted!",
		Success: true,
	})
}

func (h *ServiceHandler) getService(r *http.Request) *ServiceItem {
	h.mu.Lock()
	defer h.mu.Unlock()

	name := chi.URLParam(r, "name")
	if name == RootServiceName {
		name = ""
	}

	return h.router.Services[name]
}

func (h *ServiceHandler) getRouteIndex(fileProps *FileProperties) int {
	service, ok := h.router.Services[fileProps.ServiceName]
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

type GenerateResponse struct {
	Request  *Request  `json:"request"`
	Response *Response `json:"response"`
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
