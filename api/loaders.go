package api

import (
	"github.com/cubahno/connexions"
	"github.com/cubahno/connexions/contexts"
	"github.com/cubahno/connexions/internal"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// loadServices loads all services from the `services` directory.
func loadServices(router *Router) error {
	wg := &sync.WaitGroup{}
	var mu sync.Mutex

	openAPIFiles := make([]*connexions.FileProperties, 0)
	fixedFiles := make([]*connexions.FileProperties, 0)
	appCfg := router.Config.App

	err := filepath.Walk(appCfg.Paths.Services, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if it's a directory
		if info.IsDir() {
			return nil
		}

		fileProps, err := connexions.GetPropertiesFromFilePath(filePath, appCfg)
		if err != nil {
			log.Printf("Failed to get file properties from %s: %s\n", filePath, err.Error())
			// don't return error, as we have more files to process
			return nil
		}

		if fileProps.IsOpenAPI {
			openAPIFiles = append(openAPIFiles, fileProps)
		} else {
			fixedFiles = append(fixedFiles, fileProps)
		}

		return nil
	})

	services := map[string]*ServiceItem{}

	log.Println("Registering OpenAPI services...")
	for _, fileProps := range openAPIFiles {
		wg.Add(1)

		go func(props *connexions.FileProperties) {
			defer wg.Done()

			mu.Lock()
			defer mu.Unlock()

			// TODO(cubahno): collect first, then register
			rs := registerOpenAPIRoutes(props, router)

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

	log.Printf("Registering fixed services...\n")
	for _, fileProps := range fixedFiles {
		wg.Add(1)

		go func(props *connexions.FileProperties) {
			defer wg.Done()
			mu.Lock()
			defer mu.Unlock()

			route := registerFixedRoute(props, router)

			svc, ok := services[props.ServiceName]
			if !ok {
				svc = &ServiceItem{
					Name: props.ServiceName,
				}
				services[props.ServiceName] = svc
			}
			svc.AddRoutes(RouteDescriptions{route})
		}(fileProps)
	}

	wg.Wait()
	router.SetServices(services)

	log.Println("Registered routes.")

	return err
}

// loadContexts loads all contexts from the `contexts` directory.
// It implements RouteRegister interface so error return is mandatory.
func loadContexts(router *Router) error {
	wg := &sync.WaitGroup{}

	type parsed struct {
		ctx      *contexts.ParsedContextResult
		err      error
		filePath string
	}
	ch := make(chan parsed)

	// Walk through all files in the contexts directory
	_ = filepath.Walk(router.Config.App.Paths.Contexts, func(filePath string, info os.FileInfo, err error) error {
		if info == nil || info.IsDir() {
			return nil
		}

		wg.Add(1)

		go func(filePath string) {
			defer wg.Done()
			ctx, err := contexts.ParseContextFile(filePath, contexts.Fakes)
			ch <- parsed{
				ctx:      ctx,
				err:      err,
				filePath: filePath,
			}
		}(filePath)
		return nil
	})

	go func() {
		wg.Wait()
		close(ch)
	}()

	// Collect results
	contexts := make(map[string]map[string]any)
	aliases := make(map[string]map[string]string)

	for p := range ch {
		if p.err != nil {
			log.Printf("Failed to parse context file %s: %s\n", p.filePath, p.err.Error())
			continue
		}
		base := filepath.Base(p.filePath)
		ext := filepath.Ext(base)
		name := base[0 : len(base)-len(ext)]
		log.Printf("Adding context: %s from %s", name, filepath.Base(p.filePath))

		contexts[name] = p.ctx.Result
		aliases[name] = p.ctx.Aliases
	}

	// resolve aliases
	for ctxName, requiredAliases := range aliases {
		for ctxSourceKey, aliasTarget := range requiredAliases {
			parts := strings.Split(aliasTarget, ".")
			ns, nsPath := parts[0], strings.Join(parts[1:], ".")
			if res := internal.GetValueByDottedPath(contexts[ns], nsPath); res != nil {
				internal.SetValueByDottedPath(contexts[ctxName], ctxSourceKey, res)
			} else {
				log.Printf("context %s requires alias %s, but it's not defined", ctxName, ctxSourceKey)
			}
		}
	}

	var defaultNamespaces []map[string]string
	res := make(map[string]map[string]any, 0)

	for ctxNamespace, fileCollection := range contexts {
		// take complete namespace
		defaultNamespaces = append(defaultNamespaces, map[string]string{ctxNamespace: ""})

		res[ctxNamespace] = make(map[string]any, 0)
		for name, subCtx := range fileCollection {
			res[ctxNamespace][name] = subCtx
		}
	}

	router.SetContexts(res, defaultNamespaces)

	return nil
}
