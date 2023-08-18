package xs

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

func LoadServices(router *Router) error {
	wg := &sync.WaitGroup{}
	var mu sync.Mutex

	openAPIFiles := make([]*FileProperties, 0)
	overwriteFiles := make([]*FileProperties, 0)

	err := filepath.Walk(ServicePath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if it's a directory
		if info.IsDir() {
			return nil
		}

		fileProps, err := GetPropertiesFromFilePath(filePath)
		if err != nil {
			log.Printf("Failed to get file properties from %s: %s\n", filePath, err.Error())
			// don't return error, as we have more files to process
			return nil
		}

		if fileProps.IsOpenAPI {
			openAPIFiles = append(openAPIFiles, fileProps)
		} else {
			overwriteFiles = append(overwriteFiles, fileProps)
		}

		return nil
	})

	services := map[string]*ServiceItem{}

	log.Println("Registering OpenAPI services...")
	for _, fileProps := range openAPIFiles {
		wg.Add(1)

		go func(props *FileProperties) {
			defer wg.Done()

			mu.Lock()
			defer mu.Unlock()

			rs, err := RegisterOpenAPIRoutes(props, router)
			if err != nil {
				log.Println(err.Error())
				return
			}

			svc, ok := services[props.ServiceName]
			if !ok {
				svc = &ServiceItem{
					Name: props.ServiceName,
				}
				services[props.ServiceName] = svc
			}
			svc.AddOpenAPIFile(props)
			svc.AddRoutes(rs)
		}(fileProps)
	}

	wg.Wait()

	println("Registering overwrite services...")
	for _, fileProps := range overwriteFiles {
		wg.Add(1)

		go func(props *FileProperties) {
			defer wg.Done()
			mu.Lock()
			defer mu.Unlock()

			rs, err := RegisterOverwriteService(props, router)
			if err != nil {
				log.Println(err.Error())
				return
			}

			svc, ok := services[props.ServiceName]
			if !ok {
				svc = &ServiceItem{
					Name: props.ServiceName,
				}
				services[props.ServiceName] = svc
			}
			svc.AddRoutes(rs)
		}(fileProps)
	}

	wg.Wait()
	router.Services = services

	println("Registered routes:")
	_ = chi.Walk(router, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		fmt.Printf("[%s]:\t%s\n", method, route)
		return nil
	})
	println()

	return err
}

func LoadContexts(router *Router) error {
	wg := &sync.WaitGroup{}

	type parsed struct {
		ctx      map[string]any
		err      error
		filePath string
	}
	ch := make(chan parsed, 0)

	err := filepath.Walk(ContextPath, func(filePath string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			wg.Add(1)

			go func(filePath string) {
				defer wg.Done()
				ctx, err := ParseContextFile(filePath)
				ch <- parsed{
					ctx:      ctx,
					err:      err,
					filePath: filePath,
				}
			}(filePath)
		}
		return nil
	})

	go func() {
		wg.Wait()
		close(ch)
	}()

	// Collect errors
	for p := range ch {
		if p.err != nil {
			log.Printf("Failed to parse context file %s: %s\n", p.filePath, p.err.Error())
		}
		base := filepath.Base(p.filePath)
		ext := filepath.Ext(base)
		name := base[0 : len(base)-len(ext)]
		log.Printf("Adding context: %s from %s", name, filepath.Base(p.filePath))
		router.AddContext(name, p.ctx)
	}

	if err != nil {
		return err
	}

	return nil
}
