package xs

import (
	"fmt"
	"net/http"
)

func RegisterOverwriteService(fileProps *FileProperties, router *Router) ([]*RouteDescription, error) {
	fmt.Printf("Registering overwrite %s route for %s at %s\n", fileProps.Method, fileProps.ServiceName,
		fileProps.Resource)

	baseResource := fileProps.Prefix + fileProps.Resource
	resources := []string{baseResource}

	if fileProps.FileName == "index.json" {
		// add trailing slash and direct access to index.json
		resources = append(resources, baseResource+"/")
		resources = append(resources, baseResource+"/index.json")
	}

	// register all routes
	for _, resource := range resources {
		router.Method(fileProps.Method, resource, createOverwriteResponseHandler(fileProps, router.Config))
	}

	return []*RouteDescription{
		{
			Method: fileProps.Method,
			// add only resource to the RouteDescription, it's used only for UI purposes
			Path: fileProps.Resource,
			Type: OverwriteRoute,
			File: fileProps,
		},
	}, nil
}

func createOverwriteResponseHandler(fileProps *FileProperties, config *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if handled := handleErrorAndLatency(fileProps.ServiceName, config, w); handled {
			return
		}

		http.ServeFile(w, r, fileProps.FilePath)
	}
}
