package api

import (
	"fmt"
	"github.com/cubahno/xs"
	"net/http"
)

func RegisterOverwriteService(fileProps *FileProperties, router *Router) ([]*RouteDescription, error) {
	fmt.Printf("Registering overwrite %s route for %s at %s\n", fileProps.Method, fileProps.ServiceName,
		fileProps.Resource)

	res := []*RouteDescription{
		{
			Method: fileProps.Method,
			Path:   fileProps.Resource,
			Type:   "overwrite",
			File:   fileProps,
		},
	}

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

	return res, nil
}

func createOverwriteResponseHandler(fileProps *FileProperties, config *xs.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if handled := handleErrorAndLatency(fileProps.ServiceName, config, w); handled {
			return
		}

		http.ServeFile(w, r, fileProps.FilePath)
	}
}
