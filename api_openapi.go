package xs

import (
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"
	"net/http"
	"strings"
)

type ResourceGeneratePayload struct {
	Resource     string         `json:"resource"`
	Method       string         `json:"method"`
	Replacements map[string]any `json:"replacements"`
	IsOpenAPI    bool           `json:"isOpenApi"`
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

	for resName, pathItem := range doc.Paths {
		for method, _ := range pathItem.Operations() {
			// register route
			router.Method(method,
				fileProps.Prefix+resName,
				createOpenAPIResponseHandler(fileProps.Prefix, doc, fileProps.ValueReplacer, router.Config))

			res = append(res, &RouteDescription{
				Method: method,
				Path:   resName,
				Type:   OpenAPIRoute,
				File:   fileProps,
			})
		}
	}

	return res, nil
}

// createOpenAPIResponseHandler creates a handler function for an OpenAPI route.
func createOpenAPIResponseHandler(
	prefix string, doc *openapi3.T, valueReplacer ValueReplacer, config *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := chi.RouteContext(r.Context())
		resourceName := strings.Replace(ctx.RoutePatterns[0], prefix, "", 1)
		paths := doc.Paths[resourceName]
		if paths == nil {
			NewJSONResponse(http.StatusNotFound, ErrResourceNotFound, w)
			return
		}

		currentMethod := r.Method
		var operation *openapi3.Operation

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

		response := NewResponseFromOperation(operation, valueReplacer)

		if handled := handleErrorAndLatency(strings.TrimPrefix(prefix, "/"), config, w); handled {
			return
		}

		NewJSONResponse(response.StatusCode, response.Content, w)
	}
}
