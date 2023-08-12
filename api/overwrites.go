package api

import (
	"fmt"
	"github.com/cubahno/xs"
	"net/http"
	"strings"
)

func RegisterOverwriteService(fileProps *FileProperties, router *Router) ([]*RouteDescription, error) {
	fmt.Printf("Registering overwrite %s route for %s at %s\n", fileProps.Method, fileProps.ServiceName,
		fileProps.Resource)

	baseResource := fileProps.Resource
	if fileProps.ServiceName != "" {
		baseResource = "/" + fileProps.ServiceName + baseResource
	}

	resources := []string{baseResource}
	res := make([]*RouteDescription, 0)

	var indexName string
	if fileProps.FileName == "index.json" {
		indexName = strings.Replace(baseResource, "/"+fileProps.FileName, "", 1)
		resources = append(resources, indexName)
		resources = append(resources, indexName+"/")
	}

	if indexName == "" {
		indexName = baseResource
	}
	res = append(res, &RouteDescription{
		Method: fileProps.Method,
		Path:   indexName,
		Type:   "overwrite",
		File:   fileProps,
	})

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
