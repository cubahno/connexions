package api

import (
	"github.com/cubahno/xs"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"
	"net/http"
	"sort"
	"strings"
)

func CreateServiceRoutes(router *Router) error {
	handler := &ServiceHandler{
		router: router,
	}

	router.Get("/services", handler.list)
	router.Get("/services/{name}", handler.home)
	router.Post("/services/{name}", handler.generate)
	return nil
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
	// TODO(igor): move valueMaker to router
	valueMaker := xs.CreateValueResolver()
	res := map[string]any{}

	if !payload.IsOpenAPI {
		res["request"] = &xs.Request{
			Method: payload.Method,
			Path:   payload.Resource,
		}
		// ... find corresponding fileProps
		res["response"] = &xs.Response{
			StatusCode:  http.StatusOK,
			Content:     "",
			ContentType: "",
		}
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

	res["request"] = xs.NewRequestFromOperation(prefix, payload.Resource, payload.Method, operation, valueMaker)
	res["response"] = xs.NewResponseFromOperation(operation, valueMaker)

	NewJSONResponse(http.StatusOK, res, w)
}

type ServiceItem struct {
	Name             string              `json:"name"`
	Type             string              `json:"type"`
	HasOpenAPISchema bool                `json:"hasOpenAPISchema"`
	Routes           []*RouteDescription `json:"routes"`
	Spec             *openapi3.T         `json:"-"`
}

type RouteDescription struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Type   string `json:"type"`
}

type ServiceListResponse struct {
	Items []*ServiceItem `json:"items"`
}

type ServiceHomeResponse struct {
	Endpoints []*RouteDescription `json:"endpoints"`
}
