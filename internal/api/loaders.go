package api

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"plugin"
	"strings"
	"sync"

	"github.com/cubahno/connexions/internal/context"
	"github.com/cubahno/connexions/internal/openapi"
	"github.com/cubahno/connexions/internal/types"
)

// loadServices loads all services from the `services` directory.
func loadServices(router *Router) error {
	wg := &sync.WaitGroup{}
	var mu sync.Mutex

	openAPIFiles := make([]*openapi.FileProperties, 0)
	fixedFiles := make([]*openapi.FileProperties, 0)
	appCfg := router.Config.App

	err := filepath.Walk(appCfg.Paths.Services, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if it's a directory
		if info.IsDir() {
			return nil
		}

		fileProps, err := openapi.GetPropertiesFromFilePath(filePath, appCfg)
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

		go func(props *openapi.FileProperties) {
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

		go func(props *openapi.FileProperties) {
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
		ctx      *context.ParsedContextResult
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
			ctx, err := context.ParseContextFile(filePath, context.Fakes)
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
	cts := make(map[string]map[string]any)
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

		cts[name] = p.ctx.Result
		aliases[name] = p.ctx.Aliases
	}

	// resolve aliases
	for ctxName, requiredAliases := range aliases {
		for ctxSourceKey, aliasTarget := range requiredAliases {
			parts := strings.Split(aliasTarget, ".")
			ns, nsPath := parts[0], strings.Join(parts[1:], ".")
			if res := types.GetValueByDottedPath(cts[ns], nsPath); res != nil {
				types.SetValueByDottedPath(cts[ctxName], ctxSourceKey, res)
			} else {
				log.Printf("context %s requires alias %s, but it's not defined", ctxName, ctxSourceKey)
			}
		}
	}

	var defaultNamespaces []map[string]string
	res := make(map[string]map[string]any)

	for ctxNamespace, fileCollection := range cts {
		// take complete namespace
		defaultNamespaces = append(defaultNamespaces, map[string]string{ctxNamespace: ""})

		res[ctxNamespace] = make(map[string]any)
		for name, subCtx := range fileCollection {
			res[ctxNamespace][name] = subCtx
		}
	}

	router.SetContexts(res, defaultNamespaces)

	return nil
}

// loadPlugins loads pre-compiled user plugins.
func loadPlugins(router *Router) error {
	dir := router.Config.App.Paths.Plugins
	if dir == "" {
		return nil
	}

	var plugs []*plugin.Plugin

	// Read the directory
	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read plugin directory: %w", err)
	}

	// Iterate over files and load .so plugins
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".so" {
			continue
		}

		pluginPath := filepath.Join(dir, file.Name())
		p, err := plugin.Open(pluginPath)
		if err != nil {
			log.Printf("Failed to load plugin %s: %v", pluginPath, err)
			continue
		}

		log.Printf("Loaded plugin: %s", pluginPath)
		plugs = append(plugs, p)
	}

	if len(plugs) == 0 {
		return nil
	}

	// TODO: handle multiple plugins
	router.middlewarePlugin = plugs[0]
	log.Println("Plugins loaded successfully")

	return nil
}
