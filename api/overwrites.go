package api

import (
	"fmt"
	"github.com/cubahno/xs"
	"net/http"
	"strings"
)

func RegisterOverwriteService(fileProps *xs.FileProperties, config *xs.Config, router *Router) ([]*RouteDescription, error) {
	fmt.Printf("Registering overwrite %s route for %s at %s\n", fileProps.Method, fileProps.ServiceName,
		fileProps.Resource)

	resources := []string{fileProps.Resource}
	res := make([]*RouteDescription, 0)

	var indexName string
	if fileProps.FileName == "index"+fileProps.Extension && fileProps.Extension == ".html" || fileProps.Extension == ".json" {
		indexName = strings.Replace(fileProps.Resource, "/index"+fileProps.Extension, "", 1)
		resources = append(resources, indexName)
		resources = append(resources, indexName+"/")
	}

	if indexName == "" {
		indexName = fileProps.Resource
	}
	res = append(res, &RouteDescription{
		Method: fileProps.Method,
		Path:   indexName,
		Type:   "overwrite",
		File:   fileProps,
	})

	for _, resource := range resources {
		router.Method(fileProps.Method, resource, createOverwriteResponseHandler(fileProps, config))
	}

	return res, nil
}

func createOverwriteResponseHandler(fileProps *xs.FileProperties, config *xs.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if handled := handleErrorAndLatency(fileProps.ServiceName, config, w); handled {
			return
		}

		http.ServeFile(w, r, fileProps.FilePath)
	}
}
