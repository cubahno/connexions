package api

import (
	"fmt"
	"github.com/cubahno/xs"
	"github.com/go-chi/chi/v5"
	"net/http"
	"strings"
)

func RegisterOverwriteService(fileProps *FileProperties, config *xs.Config, router *chi.Mux) error {
	fmt.Printf("Registering overwrite %s route for %s at %s\n", fileProps.Method, fileProps.ServiceName,
		fileProps.Resource)

	resources := []string{fileProps.Resource}

	if fileProps.FileName == "index"+fileProps.Extension && fileProps.Extension == ".html" || fileProps.Extension == ".json" {
		indexName := strings.Replace(fileProps.Resource, "/index"+fileProps.Extension, "", 1)
		resources = append(resources, indexName)
		resources = append(resources, indexName+"/")
	}

	for _, resource := range resources {
		router.Method(fileProps.Method, resource, createOverwriteResponseHandler(fileProps, config))
	}

	return nil
}

func createOverwriteResponseHandler(fileProps *FileProperties, config *xs.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if handled := handleErrorAndLatency(fileProps.ServiceName, config, w); handled {
			return
		}

		http.ServeFile(w, r, fileProps.FilePath)
	}
}
