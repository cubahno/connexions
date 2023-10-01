package connexions

import (
	"log"
	"net/http"
	"strings"
)

// registerFixedRoutes registers fixed routes for a service.
func registerFixedRoute(fileProps *FileProperties, router *Router) *RouteDescription {
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

	// register all routes
	for _, resource := range resources {
		router.Method(fileProps.Method, resource, createFixedResponseHandler(router, fileProps))
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
func createFixedResponseHandler(router *Router, fileProps *FileProperties) http.HandlerFunc {
	config := router.Config
	serviceCfg := config.GetServiceConfig(fileProps.ServiceName)

	serviceCtxs := serviceCfg.Contexts
	if len(serviceCtxs) == 0 {
		serviceCtxs = router.GetDefaultContexts()
	}
	contexts := CollectContexts(serviceCtxs, router.GetContexts(), nil)
	valueReplacer := CreateValueReplacer(config, contexts)

	return func(w http.ResponseWriter, r *http.Request) {
		if HandleErrorAndLatency(serviceCfg, w) {
			return
		}

		content := generateContentFromFileProperties(fileProps.FilePath, fileProps.ContentType, valueReplacer)
		NewAPIResponse(w).WithHeader("Content-Type", fileProps.ContentType).Send(content)
	}
}
