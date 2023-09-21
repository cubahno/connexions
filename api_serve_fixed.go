package connexions

import (
	"log"
	"net/http"
	"strings"
)

func registerFixedRoute(fileProps *FileProperties, router *Router) *RouteDescription {
	log.Printf("Registering fixed %s route for %s at %s\n", fileProps.Method, fileProps.ServiceName, fileProps.Resource)

	baseResource := fileProps.Prefix + fileProps.Resource
	resources := []string{baseResource}

	if fileProps.FileName == "index.json" {
		// add trailing slash and direct access to index.json
		br := strings.TrimSuffix(baseResource, "/")
		resources = append(resources, br+"/")
		resources = append(resources, br+"/index.json")
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

func createFixedResponseHandler(router *Router, fileProps *FileProperties) http.HandlerFunc {
	config := router.Config
	serviceCfg := config.GetServiceConfig(fileProps.ServiceName)

	serviceCtxs := serviceCfg.Contexts
	if len(serviceCtxs) == 0 {
		serviceCtxs = router.ContextNames
	}
	contexts := CollectContexts(serviceCtxs, router.Contexts, nil)
	valueReplacer := CreateValueReplacer(config, contexts)

	return func(w http.ResponseWriter, r *http.Request) {
		if HandleErrorAndLatency(serviceCfg, w) {
			return
		}

		if content := generateContentFromFileProperties(fileProps.FilePath, fileProps.ContentType, valueReplacer); content != nil {
			NewJSONResponse(w).WithHeader("Content-Type", fileProps.ContentType).Send(content)
			return
		}

		http.ServeFile(w, r, fileProps.FilePath)
	}
}
