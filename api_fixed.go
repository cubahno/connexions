package connexions

import (
	"fmt"
	"net/http"
	"strings"
)

func RegisterFixedService(fileProps *FileProperties, router *Router) (RouteDescriptions, error) {
	fmt.Printf("Registering fixed %s route for %s at %s\n", fileProps.Method, fileProps.ServiceName,
		fileProps.Resource)

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
		router.Method(fileProps.Method, resource, createFixedResponseHandler(fileProps, router.Config))
	}

	rd := &RouteDescription{
		Method: fileProps.Method,
		// add only resource to the RouteDescription, it's used only for UI purposes
		Path:        fileProps.Resource,
		Type:        FixedRouteType,
		ContentType: fileProps.ContentType,
		File:        fileProps,
	}

	return RouteDescriptions{rd}, nil
}

func createFixedResponseHandler(fileProps *FileProperties, config *Config) http.HandlerFunc {
	svcConfig := config.GetServiceConfig(fileProps.ServiceName)

	return func(w http.ResponseWriter, r *http.Request) {
		if handled := handleErrorAndLatency(svcConfig, w); handled {
			return
		}

		http.ServeFile(w, r, fileProps.FilePath)
	}
}
