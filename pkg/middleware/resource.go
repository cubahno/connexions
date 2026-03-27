package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

const resourcePathKey ctxKey = "resourcePath"

// CreateResourceResolverMiddleware resolves the OpenAPI spec resource path
// (e.g. /pets/{id}) from the request and stores it in the context.
func CreateResourceResolverMiddleware(params *Params) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			endpointPath := getEndpointPath(req, params.serviceConfig.Name)

			resourcePath := endpointPath
			if r := params.router; r != nil {
				rctx := chi.NewRouteContext()
				if r.Match(rctx, req.Method, endpointPath) {
					if pattern := rctx.RoutePattern(); pattern != "" {
						resourcePath = strings.TrimSuffix(pattern, "/*")
					}
				}
			}

			ctx := context.WithValue(req.Context(), resourcePathKey, resourcePath)
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	}
}

// GetResourcePath returns the resolved OpenAPI resource path from the context.
// Falls back to the raw URL path if the context value is not set.
func GetResourcePath(req *http.Request) string {
	if v, ok := req.Context().Value(resourcePathKey).(string); ok {
		return v
	}
	return req.URL.Path
}
