package api

import (
	"log"
	"net/http"
	"strings"

	"github.com/cubahno/connexions/internal/context"
	"github.com/cubahno/connexions/internal/openapi"
	"github.com/cubahno/connexions/internal/replacer"
)

// registerFixedRoutes registers fixed routes for a service.
func registerFixedRoute(fileProps *openapi.FileProperties, router *Router) *RouteDescription {
	log.Printf("Registering fixed %s route for %s at %s\n", fileProps.Method, fileProps.ServiceName, fileProps.Resource)

	baseResource := strings.TrimSuffix(fileProps.Prefix+fileProps.Resource, "/")
	if baseResource == "" {
		baseResource = "/"
	}
	resources := []string{baseResource}

	if strings.HasPrefix(fileProps.FileName, "index.") {
		// add trailing slash and direct access to index.*
		resources = append(resources, baseResource+"/")
		resources = append(resources, baseResource+"/"+fileProps.FileName)
	}

	config := router.Config
	serviceCfg := config.GetServiceConfig(fileProps.ServiceName)

	// register all routes
	for _, resource := range resources {
		mwParams := &MiddlewareParams{
			ServiceConfig:  serviceCfg,
			Service:        fileProps.ServiceName,
			Resource:       resource,
			ResourcePrefix: fileProps.Prefix,
			Plugin:         router.callbacksPlugin,
			history:        router.history,
		}

		router.
			With(CreateCacheRequestMiddleware(mwParams)).
			With(CreateRequestTransformerMiddleware(mwParams)).
			With(CreateUpstreamRequestMiddleware(mwParams)).
			With(CreateResponseMiddleware(mwParams)).
			Method(fileProps.Method, resource, createFixedResponseHandler(router, fileProps))
	}

	return &RouteDescription{
		Method: fileProps.Method,
		// add only resource to the RouteDescription, it's used only for UI purposes
		Path:        fileProps.Resource,
		Type:        FixedRouteType,
		ContentType: fileProps.ContentType,
		File:        fileProps,
	}
}

// createFixedResponseHandler creates a http.HandlerFunc for fixed routes.
func createFixedResponseHandler(router *Router, fileProps *openapi.FileProperties) http.HandlerFunc {
	config := router.Config
	serviceCfg := config.GetServiceConfig(fileProps.ServiceName)

	serviceCtxs := serviceCfg.Contexts
	if len(serviceCtxs) == 0 {
		serviceCtxs = router.GetDefaultContexts()
	}
	cts := context.CollectContexts(serviceCtxs, router.GetContexts(), nil)
	valueReplacer := replacer.CreateValueReplacer(config, replacer.Replacers, cts)

	return func(w http.ResponseWriter, r *http.Request) {
		router.history.Set(fileProps.Resource, r, nil)

		if HandleErrorAndLatency(serviceCfg, w) {
			return
		}

		content := openapi.GenerateContentFromFileProperties(fileProps.FilePath, fileProps.ContentType, valueReplacer)
		NewAPIResponse(w).WithHeader("Content-Type", fileProps.ContentType).Send(content)
	}
}
