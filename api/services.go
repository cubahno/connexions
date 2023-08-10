package api

import (
	"github.com/cubahno/xs"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"
	"net/http"
	"os"
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
	response := r.FormValue("response")

	// Get the uploaded file
	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	println(handler.Filename, response, service, method, isOpenAPI, path)

	// // Create a destination file on the server
	// destinationPath := filepath.Join("uploads", handler.Filename)
	// dst, err := os.Create(destinationPath)
	// if err != nil {
	// 	http.Error(w, "Error creating the file on the server", http.StatusInternalServerError)
	// 	return
	// }
	// defer dst.Close()
	//
	// // Copy the uploaded file to the destination file
	// _, err = io.Copy(dst, file)
	// if err != nil {
	// 	http.Error(w, "Error copying the file", http.StatusInternalServerError)
	// 	return
	// }

	NewJSONResponse(http.StatusCreated, nil, w)
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
