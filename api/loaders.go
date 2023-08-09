package api

import (
	"github.com/cubahno/xs"
	"github.com/go-chi/chi/v5"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FileProperties contains inferred properties of a file that is being loaded from service directory.
type FileProperties struct {
	ServiceName       string
	IsPossibleOpenAPI bool
	Method            string
	Resource          string
	FilePath          string
	FileName          string
	Extension         string
}

func handleErrorAndLatency(service string, config *xs.Config, w http.ResponseWriter) bool {
	svcConfig := config.GetServiceConfig(service)
	if svcConfig.Latency > 0 {
		log.Printf("Latency of %s is %s\n", service, svcConfig.Latency)
		time.Sleep(svcConfig.Latency)
	}

	err := svcConfig.Errors.GetError()
	if err != 0 {
		NewResponse(err, []byte("Random config error"), w)
		return true
	}

	return false
}

func LoadServices(router *chi.Mux) error {
	wg := &sync.WaitGroup{}

	config, err := xs.NewConfigFromFile()
	if err != nil {
		log.Printf("Failed to load config file: %s\n", err.Error())
		config = xs.NewDefaultConfig()
	}
	possibleOpenAPIFiles := make([]*FileProperties, 0)
	overwriteFiles := make([]*FileProperties, 0)

	err = filepath.Walk(xs.ServicePath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if it's a directory
		if info.IsDir() {
			return nil
		}

		fileProps := GetPropertiesFromFilePath(filePath)
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

		go func(props *FileProperties) {
			defer wg.Done()
			err := RegisterOverwriteService(props, config, router)
			if err != nil {
				println(err.Error())
			}
		}(fileProps)
	}

	wg.Wait()

	println("Registering OpenAPI services...")
	for _, fileProps := range possibleOpenAPIFiles {
		wg.Add(1)

		go func(props *FileProperties) {
			defer wg.Done()

			err := RegisterOpenAPIService(props, config, router)
			if err != nil {
				println(err.Error())
				// try to register as overwrite service
				err := RegisterOverwriteService(props, config, router)
				if err != nil {
					println(err.Error())
				}
			}
		}(fileProps)
	}

	wg.Wait()

	return err
}
