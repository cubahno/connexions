package api

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/cubahno/connexions/internal/context"
	"github.com/cubahno/connexions/internal/openapi"
	"github.com/cubahno/connexions/internal/replacer"
	"github.com/go-chi/chi/v5"
)

// OpenAPIHandler handles OpenAPI routes serve.
type OpenAPIHandler struct {
	*BaseHandler
	router        *Router
	fileProps     *openapi.FileProperties
	cache         openapi.CacheStorage
	valueReplacer replacer.ValueReplacer
}

// registerOpenAPIRoutes adds spec routes to the router and
// creates necessary closure to serve routes.
func registerOpenAPIRoutes(fileProps *openapi.FileProperties, router *Router) RouteDescriptions {
	log.Printf("Registering OpenAPI service %s\n", fileProps.ServiceName)

	res := make(RouteDescriptions, 0)

	config := router.Config
	serviceCfg := config.GetServiceConfig(fileProps.ServiceName)

	serviceCtxs := serviceCfg.Contexts
	if len(serviceCtxs) == 0 {
		serviceCtxs = router.GetDefaultContexts()
	}
	cts := context.CollectContexts(serviceCtxs, router.GetContexts(), nil)
	valueReplacer := replacer.CreateValueReplacer(config, replacer.Replacers, cts)

	handler := &OpenAPIHandler{
		router:        router,
		fileProps:     fileProps,
		cache:         openapi.NewMemoryStorage(),
		valueReplacer: valueReplacer,
	}

	for resName, resMethods := range fileProps.Spec.GetResources() {
		mwParams := &MiddlewareParams{
			ServiceConfig:  serviceCfg,
			Service:        fileProps.ServiceName,
			Resource:       resName,
			ResourcePrefix: fileProps.Prefix,
			Plugin:         router.middlewarePlugin,
			history:        router.history,
		}

		for _, method := range resMethods {
			// register route
			router.
				With(CreateCacheRequestMiddleware(mwParams)).
				With(CreateRequestTransformerMiddleware(mwParams)).
				With(CreateUpstreamRequestMiddleware(mwParams)).
				With(CreateResponseMiddleware(mwParams)).
				MethodFunc(method, fileProps.Prefix+resName, handler.serve)

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

// serve serves the OpenAPI spec resources.
func (h *OpenAPIHandler) serve(w http.ResponseWriter, r *http.Request) {
	prefix := h.fileProps.Prefix
	doc := h.fileProps.Spec
	config := h.router.Config
	serviceCfg := config.GetServiceConfig(h.fileProps.ServiceName)

	ctx := chi.RouteContext(r.Context())

	resourcePath := strings.Replace(ctx.RoutePatterns[0], prefix, "", 1)
	h.router.history.Set(resourcePath, r, nil)

	findOptions := &openapi.OperationDescription{
		Service:  h.fileProps.ServiceName,
		Resource: resourcePath,
		Method:   r.Method,
	}
	operation := doc.FindOperation(findOptions)
	if operation == nil {
		// edge case: we get here only if the file gets removed while router is running.
		// not json response because if path doesn't exist, it's just plain 404.
		contents := []byte(ErrResourceNotFound.Error())
		h.Response(w).WithStatusCode(http.StatusNotFound).Send(contents)
		return
	}

	if serviceCfg.Cache.Schema {
		operation = openapi.NewCacheOperationAdapter(h.fileProps.ServiceName, operation, h.cache)
	}
	operation = operation.WithParseConfig(serviceCfg.ParseConfig)

	validator := openapi.NewValidator(doc)
	req := replaceRequestResource(r, resourcePath)

	if serviceCfg.Validate.Request && validator != nil {
		hdrs := make(map[string]any)
		for name, values := range r.Header {
			hdrs[name] = values
		}

		body := ""
		if req.Body != nil {
			bodyBytes, err := io.ReadAll(req.Body)
			if err == nil {
				body = string(bodyBytes)
			}
		}

		errs := validator.ValidateRequest(&openapi.GeneratedRequest{
			Headers:     hdrs,
			Method:      r.Method,
			Path:        resourcePath,
			ContentType: req.Header.Get("Content-Type"),
			Request:     req,
			Body:        body,
		})
		if len(errs) > 0 {
			h.JSONResponse(w).WithStatusCode(http.StatusBadRequest).Send(&SimpleResponse{
				Message: fmt.Sprintf("Invalid request: %d errors: %v", len(errs), errs),
				Success: false,
			})
			return
		}
	}

	response := openapi.NewResponseFromOperation(req, operation, h.valueReplacer)
	if serviceCfg.Validate.Response && validator != nil {
		errs := validator.ValidateResponse(response)
		if len(errs) > 0 {
			h.JSONResponse(w).WithStatusCode(http.StatusBadRequest).Send(&SimpleResponse{
				Message: fmt.Sprintf("Invalid response: %d errors: %v", len(errs), errs),
				Success: false,
			})
			return
		}
	}

	// return error if configured
	if HandleErrorAndLatency(serviceCfg, w) {
		return
	}

	res := h.Response(w).WithStatusCode(response.StatusCode)

	// set headers
	for name, values := range response.Headers {
		for _, value := range values {
			res = res.WithHeader(name, value)
		}
	}

	res.Send(response.Content)
}

// replaceRequestResource replaces the resource in the request with the given one.
// Our services might get the extra prefix from the service name but the OpenAPI spec doesn't have it:
// so validation will fail.
func replaceRequestResource(req *http.Request, resource string) *http.Request {
	newReq := new(http.Request)
	*newReq = *req
	newReq.URL = newReq.URL.ResolveReference(&url.URL{Path: resource})
	return newReq
}

// ResourceGeneratePayload is a payload for generating resources.
// It contains a map of replacements.
// It is used only with generating resources endpoint.
// It's merged together with contexts but has higher priority.
type ResourceGeneratePayload struct {
	Replacements map[string]any `json:"replacements"`
}
