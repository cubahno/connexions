package xs

import (
	"fmt"
	"net/http"
	"strings"
)

func RegisterOverwriteService(fileProps *FileProperties, router *Router) ([]*RouteDescription, error) {
	fmt.Printf("Registering overwrite %s route for %s at %s\n", fileProps.Method, fileProps.ServiceName,
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
		router.Method(fileProps.Method, resource, createOverwriteResponseHandler(fileProps, router.Config))
	}

	return []*RouteDescription{
		{
			Method: fileProps.Method,
			// add only resource to the RouteDescription, it's used only for UI purposes
			Path:        fileProps.Resource,
			Type:        OverwriteRouteType,
			ContentType: fileProps.ContentType,
			File:        fileProps,
		},
	}, nil
}

func createOverwriteResponseHandler(fileProps *FileProperties, config *Config) http.HandlerFunc {
	svcConfig := config.GetServiceConfig(fileProps.ServiceName)

	return func(w http.ResponseWriter, r *http.Request) {
		if handled := handleErrorAndLatency(svcConfig, w); handled {
			return
		}

		http.ServeFile(w, r, fileProps.FilePath)
	}
}
