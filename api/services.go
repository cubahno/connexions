package api

import (
	"github.com/go-chi/chi/v5"
	"net/http"
	"sort"
)

func CreateServiceRoutes(router *Router) error {
	handler := &ServiceHandler{
		router: router,
	}

	router.Get("/services", handler.list)
	router.Get("/services/{name}", handler.home)
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

type ServiceItem struct {
	Name             string              `json:"name"`
	Type             string              `json:"type"`
	HasOpenAPISchema bool                `json:"hasOpenAPISchema"`
	Routes           []*RouteDescription `json:"routes"`
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
