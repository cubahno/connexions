package api

import (
	"bufio"
	"bytes"
	"github.com/cubahno/xs"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

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
	Name             string              `json:"name"`
	Type             string              `json:"type"`
	HasOpenAPISchema bool                `json:"hasOpenAPISchema"`
	Routes           []*RouteDescription `json:"routes"`
	Spec             *openapi3.T         `json:"-"`
	SpecFile         *xs.FileProperties  `json:"-"`
}

type RouteDescription struct {
	Method string             `json:"method"`
	Path   string             `json:"path"`
	Type   string             `json:"type"`
	File   *xs.FileProperties `json:"-"`
}

type ServiceListResponse struct {
	Items []*ServiceItem `json:"items"`
}

type ServiceHomeResponse struct {
	Endpoints []*RouteDescription `json:"endpoints"`
}

type ServiceHandler struct {
	router *Router
}

func (h *ServiceHandler) list(w http.ResponseWriter, r *http.Request) {
	services := h.router.Services
	var keys []string
	for key := range services {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	items := make([]*ServiceItem, 0)
	for _, key := range keys {
		items = append(items, services[key])
	}
	res := &ServiceListResponse{
		Items: items,
	}

	NewJSONResponse(http.StatusOK, res, w)
}

func (h *ServiceHandler) create(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(256 * 1024 * 1024) // Limit form size to 256 MB
	if err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	service := r.FormValue("name")
	method := r.FormValue("method")
	isOpenAPI := r.FormValue("isOpenApi") == "true"
	path := r.FormValue("path")
	content := []byte(r.FormValue("response"))

	// Get the uploaded file
	file, handler, _ := r.FormFile("file")
	if file != nil {
		defer file.Close()
	}

	if !xs.IsValidHTTPVerb(method) {
		http.Error(w, "Invalid HTTP verb", http.StatusBadRequest)
		return
	}
	method = strings.ToUpper(method)

	path = "/" + strings.Trim(path, "/")
	if !xs.IsValidURLResource(path) {
		http.Error(w, "Invalid URL resource", http.StatusBadRequest)
		return
	}

	// get uploaded file
	ext := ""
	if handler != nil {
		file, err = handler.Open()
		if err != nil {
			http.Error(w, "Error retrieving the file", http.StatusInternalServerError)
			return
		}
		defer file.Close()
		reader := bufio.NewReader(file)
		buff := bytes.NewBuffer(make([]byte, 0))
		part := make([]byte, 1024)
		count := 0

		for {
			if count, err = reader.Read(part); err != nil {
				break
			}
			buff.Write(part[:count])
		}
		if err != io.EOF {
			http.Error(w, "Error reading the file", http.StatusInternalServerError)
		} else {
			err = nil
		}
		content = buff.Bytes()
		ext = filepath.Ext(handler.Filename)
	}

	filePath := xs.ServicePath
	if service != "" {
		filePath += "/" + service
	}

	var doc *openapi3.T
	if isOpenAPI {
		if len(content) == 0 {
			http.Error(w, "OpenAPI spec is empty", http.StatusBadRequest)
			return
		}
		loader := openapi3.NewLoader()
		doc, err = loader.LoadFromData(content)
		if err != nil {
			http.Error(w, "Invalid OpenAPI spec: "+err.Error(), http.StatusBadRequest)
			return
		}

		filePath += ext
	} else {
		filePath += "/" + strings.ToLower(method)
		if path != "" {
			filePath += path
		}

		pathExt := filepath.Ext(path)
		if pathExt == "" {
			pathExt = ext
		}
		if pathExt == "" {
			filePath += "/index.json"
		}
	}

	dirPath := filepath.Dir(filePath)
	// Create directories recursively
	err = os.MkdirAll(dirPath, os.ModePerm)
	if err != nil {
		http.Error(w, "Error creating directories", http.StatusInternalServerError)
		return
	}

	dest, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "Error creating file", http.StatusInternalServerError)
		return
	}

	_, err = dest.Write(content)
	if err != nil {
		http.Error(w, "Error writing file", http.StatusInternalServerError)
		return
	}

	fileProps := xs.GetPropertiesFromFilePath(filePath)
	println(doc)
	println(fileProps.ContentType)
	NewJSONResponse(http.StatusCreated, map[string]any{"success": true, "message": "Service saved!"}, w)
}

func (h *ServiceHandler) home(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
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
	if service == nil {
		NewJSONResponse(http.StatusNotFound, "Service not found", w)
		return
	}

	fileProps := service.SpecFile
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
	service := h.router.Services[name]
	if service == nil {
		NewJSONResponse(http.StatusNotFound, "Service not found", w)
		return
	}

	prefix := "/" + name
	// TODO(igor): move valueResolver to router
	valueResolver := xs.CreateValueResolver()
	jsonResolver := xs.CreateJSONResolver()
	res := map[string]any{}

	if !payload.IsOpenAPI {
		var fileProps *xs.FileProperties
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

		res["request"] = xs.NewRequestFromFileProperties(fileProps, jsonResolver)
		res["response"] = xs.NewResponseFromFileProperties(fileProps, jsonResolver)
		NewJSONResponse(http.StatusOK, res, w)
		return
	}

	spec := service.Spec
	if spec == nil {
		NewJSONResponse(http.StatusNotFound, "Service spec not found", w)
		return
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

	res["request"] = xs.NewRequestFromOperation(prefix, payload.Resource, payload.Method, operation, valueResolver)
	res["response"] = xs.NewResponseFromOperation(operation, valueResolver)

	NewJSONResponse(http.StatusOK, res, w)
}
