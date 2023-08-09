package api

import (
	"fmt"
	"github.com/cubahno/xs"
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

// RegisterOpenAPIService loads an OpenAPI specification from a file and adds the routes to the router.
func RegisterOpenAPIService(fileProps *FileProperties, config *xs.Config, router *Router) (*openapi3.T, []*RouteDescription, error) {
	fmt.Printf("Registering OpenAPI service %s\n", fileProps.ServiceName)
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(fileProps.FilePath)
	if err != nil {
		return nil, nil, err
	}

	res := make([]*RouteDescription, 0)

	prefix := ""
	if fileProps.ServiceName != "" {
		prefix = "/" + fileProps.ServiceName
	}

	valueMaker := xs.CreateValueResolver()

	for resName, pathItem := range doc.Paths {
		for method, _ := range pathItem.Operations() {
			// register route
			router.Method(method, prefix+resName, createOpenAPIResponseHandler(prefix, doc, valueMaker, config))

			res = append(res, &RouteDescription{
				Method: method,
				Path:   resName,
				Type:   "openapi",
			})
		}
	}

	return doc, res, nil
}

// createOpenAPIResponseHandler creates a handler function for an OpenAPI route.
func createOpenAPIResponseHandler(prefix string, doc *openapi3.T, valueMaker xs.ValueResolver, config *xs.Config) http.HandlerFunc {
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

		response := xs.NewResponseFromOperation(operation, valueMaker)

		if handled := handleErrorAndLatency(strings.TrimPrefix(prefix, "/"), config, w); handled {
			return
		}

		NewJSONResponse(response.StatusCode, response.Content, w)
	}
}
