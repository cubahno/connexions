package main

import (
	"github.com/cubahno/xs"
	"github.com/cubahno/xs/api"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

func main() {
	readSpec()
}

func readSpec() {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	err := loadServices(xs.ServicePath, r)
	if err != nil {
		panic(err)
	}

	http.ListenAndServe(":2200", r)
	println("Server started on port 2200")
}

func loadServices(serviceDirPath string, router *chi.Mux) error {
	wg := &sync.WaitGroup{}

	config := xs.MustConfig()
	possibleOpenAPIFiles := make([]*api.FileProperties, 0)
	overwriteFiles := make([]*api.FileProperties, 0)

	err := filepath.Walk(serviceDirPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if it's a directory
		if info.IsDir() {
			return nil
		}

		fileProps := api.GetPropertiesFromFilePath(filePath)
		if fileProps.IsPossibleOpenAPI {
			possibleOpenAPIFiles = append(possibleOpenAPIFiles, fileProps)
		} else {
			overwriteFiles = append(overwriteFiles, fileProps)
		}

		return nil
	})

	// these are more specific and should be registered first
	println("Registering overwrite services...")
	for _, fileProps := range overwriteFiles {
		wg.Add(1)

		go func(props *api.FileProperties) {
			defer wg.Done()
			err := api.RegisterOverwriteService(props, config, router)
			if err != nil {
				println(err.Error())
			}
		}(fileProps)
	}

	wg.Wait()


	println("Registering OpenAPI services...")
	for _, fileProps := range possibleOpenAPIFiles {
		wg.Add(1)

		go func(props *api.FileProperties) {
			defer wg.Done()

			err := api.RegisterOpenAPIService(props, config, router)
			if err != nil {
				println(err.Error())
				// try to register as overwrite service
				err := api.RegisterOverwriteService(props, config, router)
				if err != nil {
					println(err.Error())
				}
			}
		}(fileProps)
	}

	wg.Wait()


	return err
}
