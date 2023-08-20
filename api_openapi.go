package xs

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"log"
	"net/http"
	"strings"
)

type ResourceGeneratePayload struct {
	Resource     string         `json:"resource"`
	Method       string         `json:"method"`
	Replacements map[string]any `json:"replacements"`
}

// RegisterOpenAPIRoutes adds spec routes to the router and
// creates necessary closure to serve routes.
func RegisterOpenAPIRoutes(fileProps *FileProperties, router *Router) ([]*RouteDescription, error) {
	fmt.Printf("Registering OpenAPI service %s\n", fileProps.ServiceName)

	res := make([]*RouteDescription, 0)

	doc := fileProps.Spec
	if doc == nil {
		return nil, ErrOpenAPISpecIsEmpty
	}

	handler := &OpenAPIHandler{
		router:    router,
		spec:      doc,
		fileProps: fileProps,
	}

	for resName, pathItem := range doc.Paths {
		for method, _ := range pathItem.Operations() {
			// register route
			router.MethodFunc(method, fileProps.Prefix+resName, handler.serve)

			res = append(res, &RouteDescription{
				Method: method,
				Path:   resName,
				Type:   OpenAPIRouteType,
				File:   fileProps,
			})
		}
	}

	return res, nil
}

type OpenAPIHandler struct {
	router    *Router
	spec      *Document
	fileProps *FileProperties
}

func (h *OpenAPIHandler) serve(w http.ResponseWriter, r *http.Request) {
	prefix := h.fileProps.Prefix
	doc := h.spec
	config := h.router.Config
	serviceCfg := config.GetServiceConfig(h.fileProps.ServiceName)

	ctx := chi.RouteContext(r.Context())
	resourceName := strings.Replace(ctx.RoutePatterns[0], prefix, "", 1)
	paths := doc.Paths[resourceName]
	if paths == nil {
		NewJSONResponse(http.StatusNotFound, ErrResourceNotFound, w)
		return
	}

	currentMethod := r.Method
	var operation *Operation

	if currentMethod == http.MethodGet {
		operation = paths.Get
	} else if currentMethod == http.MethodPost {
		operation = paths.Post
	} else if currentMethod == http.MethodPut {
		operation = paths.Put
	} else if currentMethod == http.MethodDelete {
		operation = paths.Delete
	} else if currentMethod == http.MethodOptions {
		operation = paths.Options
	} else if currentMethod == http.MethodHead {
		operation = paths.Head
	} else if currentMethod == http.MethodPatch {
		operation = paths.Patch
	} else if currentMethod == http.MethodTrace {
		operation = paths.Trace
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if serviceCfg.Validate.Request && operation.RequestBody != nil {
		err := ValidateRequest(r, operation.RequestBody.Value)
		if err != nil {
			log.Printf("error validating request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
	}

	var valueReplacer ValueReplacer
	if h.fileProps.ValueReplacerFactory != nil {
		serviceCtxs := serviceCfg.Contexts
		if len(serviceCtxs) == 0 {
			serviceCtxs = h.router.ContextNames
		}

		contexts := CollectContexts(serviceCtxs, h.router.Contexts, nil)

		valueReplacer = h.fileProps.ValueReplacerFactory(&Resource{
			Service:     strings.Trim(prefix, "/"),
			Path:        resourceName,
			ContextData: contexts,
		})
	}

	response := NewResponseFromOperation(operation, valueReplacer)
	if serviceCfg.Validate.Response {
		err := ValidateResponse(r, response, operation)
		if err != nil {
			log.Printf("error validating response: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Invalid response: " + err.Error()))
			return
		}
	}

	if handled := handleErrorAndLatency(serviceCfg, w); handled {
		return
	}

	NewJSONResponse(response.StatusCode, response.Content, w)
}
