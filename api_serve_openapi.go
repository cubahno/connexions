package connexions

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"net/http"
	"strings"
)

type ResourceGeneratePayload struct {
	Replacements map[string]any `json:"replacements"`
}

// registerOpenAPIRoutes adds spec routes to the router and
// creates necessary closure to serve routes.
func registerOpenAPIRoutes(fileProps *FileProperties, router *Router) RouteDescriptions {
	fmt.Printf("Registering OpenAPI service %s\n", fileProps.ServiceName)

	res := make(RouteDescriptions, 0)

	doc := fileProps.Spec

	handler := &OpenAPIHandler{
		router:    router,
		spec:      doc,
		fileProps: fileProps,
	}

	for resName, resMethods := range doc.GetResources() {
		for _, method := range resMethods {
			// register route
			router.MethodFunc(method, fileProps.Prefix+resName, handler.serve)

			res = append(res, &RouteDescription{
				Method:      method,
				Path:        resName,
				Type:        OpenAPIRouteType,
				ContentType: fileProps.ContentType,
				File:        fileProps,
			})
		}
	}
	res.Sort()

	return res
}

type OpenAPIHandler struct {
	*BaseHandler
	router    *Router
	spec      Document
	fileProps *FileProperties
}

func (h *OpenAPIHandler) serve(w http.ResponseWriter, r *http.Request) {
	prefix := h.fileProps.Prefix
	doc := h.spec
	config := h.router.Config
	serviceCfg := config.GetServiceConfig(h.fileProps.ServiceName)

	ctx := chi.RouteContext(r.Context())
	resourceName := strings.Replace(ctx.RoutePatterns[0], prefix, "", 1)

	operation := doc.FindOperation(&FindOperationOptions{
		Service:  h.fileProps.ServiceName,
		Resource: resourceName,
		Method:   r.Method,
	})
	if operation == nil {
		h.JSONResponse(w).WithStatusCode(http.StatusNotFound).Send(&SimpleResponse{
			Message: ErrResourceNotFound.Error(),
			Success: false,
		})
		return
	}
	operation = operation.WithParseConfig(serviceCfg.ParseConfig)

	reqBody, contentType := operation.GetRequestBody()
	if serviceCfg.Validate.Request && reqBody != nil {
		err := ValidateRequest(r, reqBody, contentType)
		if err != nil {
			h.JSONResponse(w).WithStatusCode(http.StatusBadRequest).Send(&SimpleResponse{
				Message: "Invalid request: " + err.Error(),
				Success: false,
			})
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
			Service:           strings.Trim(prefix, "/"),
			Path:              resourceName,
			ContextData:       contexts,
			ContextAreaPrefix: config.App.ContextAreaPrefix,
		})
	}

	response := NewResponseFromOperation(operation, valueReplacer)
	if serviceCfg.Validate.Response {
		err := ValidateResponse(r, response, operation)
		if err != nil {
			h.JSONResponse(w).WithStatusCode(http.StatusBadRequest).Send(&SimpleResponse{
				Message: "Invalid response: " + err.Error(),
				Success: false,
			})
			return
		}
	}

	// return error if configured
	if handleErrorAndLatency(serviceCfg, w) {
		return
	}

	res := h.Response(w).WithStatusCode(response.StatusCode)

	// set headers
	if response.Headers.Get("Content-Type") == "" {
		response.Headers.Set("Content-Type", response.ContentType)
	}
	for name, values := range response.Headers {
		for _, value := range values {
			res = res.WithHeader(name, value)
		}
	}

	res.Send(response.Content)
}
