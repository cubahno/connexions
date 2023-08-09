package api

import (
	"github.com/cubahno/xs"
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

		select {
		case <-time.After(svcConfig.Latency):
		}
	}

	err := svcConfig.Errors.GetError()
	if err != 0 {
		NewResponse(err, []byte("Random config error"), w)
		return true
	}

	return false
}

func LoadServices(router *Router) error {
	wg := &sync.WaitGroup{}

	config, err := xs.NewConfigFromFile()
	if err != nil {
		log.Printf("Failed to load config file: %s\n", err.Error())
		config = xs.NewDefaultConfig()
	}
	possibleOpenAPIFiles := make([]*FileProperties, 0)
	overwriteFiles := make([]*FileProperties, 0)
	serviceRoutes := make(map[string][]*RouteDescription)

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

	services := map[string]*ServiceItem{}

	// these are more specific and should be registered first
	println("Registering overwrite services...")
	for _, fileProps := range overwriteFiles {
		wg.Add(1)

		go func(props *FileProperties) {
			defer wg.Done()
			rs, err := RegisterOverwriteService(props, config, router)
			if err != nil {
				println(err.Error())
			} else {
				services[props.ServiceName] = &ServiceItem{
					Name: props.ServiceName,
					Type: "overwrite",
				}
				if _, ok := serviceRoutes[props.ServiceName]; !ok {
					serviceRoutes[props.ServiceName] = make([]*RouteDescription, 0)
				}
				serviceRoutes[props.ServiceName] = append(serviceRoutes[props.ServiceName], rs...)
			}
		}(fileProps)
	}

	wg.Wait()

	println("Registering OpenAPI services...")
	for _, fileProps := range possibleOpenAPIFiles {
		wg.Add(1)

		go func(props *FileProperties) {
			defer wg.Done()

			spec, rs, err := RegisterOpenAPIService(props, config, router)
			if err != nil {
				println(err.Error())
				// try to register as overwrite service
				rs, err = RegisterOverwriteService(props, config, router)
				if err != nil {
					println(err.Error())
				} else {
					services[props.ServiceName] = &ServiceItem{
						Name: props.ServiceName,
						Type: "overwrite",
					}
				}
			} else {
				services[props.ServiceName] = &ServiceItem{
					Name:             props.ServiceName,
					Type:             "openapi",
					HasOpenAPISchema: true,
					Spec:             spec,
				}
			}
			if _, ok := serviceRoutes[props.ServiceName]; !ok {
				serviceRoutes[props.ServiceName] = make([]*RouteDescription, 0)
			}
			serviceRoutes[props.ServiceName] = append(serviceRoutes[props.ServiceName], rs...)
		}(fileProps)
	}

	wg.Wait()

	for _, service := range services {
		service.Routes = serviceRoutes[service.Name]
	}

	router.Services = services

	return err
}
