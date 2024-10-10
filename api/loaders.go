package api

import (
	"bytes"
	"fmt"
	"github.com/cubahno/connexions"
	"github.com/cubahno/connexions/contexts"
	"github.com/cubahno/connexions/internal"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
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
			if res := internal.GetValueByDottedPath(cts[ns], nsPath); res != nil {
				internal.SetValueByDottedPath(cts[ctxName], ctxSourceKey, res)
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

	// Set fake contexts
	res["fake"] = make(map[string]any)
	defaultNamespaces = append(defaultNamespaces, map[string]string{"fake": ""})
	for name, fakeFunc := range contexts.Fakes {
		// this allows to use names like {fake:uuid.v4} in response templates
		res["fake"]["fake:"+name] = fakeFunc
	}

	router.SetContexts(res, defaultNamespaces)

	return nil
}

// LoadCallbacks compiles user-provided Go code, including dependencies.
func loadCallbacks(router *Router) error {
	dir := router.Config.App.Paths.Callbacks
	if dir == "" {
		return nil
	}

	p, err := compilePlugin(dir)
	if err != nil {
		return fmt.Errorf("failed to open callbacks plugin: %v", err)
	}

	router.callbacksPlugin = p
	log.Println("Callbacks loaded successfully")

	return nil
}

func compilePlugin(dir string) (*plugin.Plugin, error) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "usergo")
	if err != nil {
		return nil, err
	}
	// Clean up
	defer os.RemoveAll(tmpDir)

	// Copy user-provided Go files into the temporary directory
	if err := filepath.Walk(dir, func(src string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		dst := filepath.Join(tmpDir, info.Name())
		content, err := os.ReadFile(src)
		if err != nil {
			return err
		}

		return os.WriteFile(dst, content, 0644)
	}); err != nil {
		return nil, err
	}

	if err := initModuleIfNone(tmpDir); err != nil {
		return nil, fmt.Errorf("failed to initialize module: %v", err)
	}

	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = tmpDir

	// Create buffers to capture output and errors
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to tidy callback modules: %v", err)
	}

	soName := "userlib.so"

	// Build the user code into a shared library
	// Change working directory to the temporary directory
	cmdArgs := []string{"build", "-buildmode=plugin"}

	// Check if the environment variable is set
	if os.Getenv("DEBUG_BUILD") == "true" {
		cmdArgs = append(cmdArgs, "-gcflags", "all=-N -l")
	}

	cmdArgs = append(cmdArgs, "-o", soName, ".")

	cmd = exec.Command("go", cmdArgs...)
	cmd.Env = append(os.Environ(), "CGO_ENABLED=1", "GOOS=linux", "GOARCH=amd64", "GO111MODULE=on")
	cmd.Dir = tmpDir
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to build callbacks: %s", out.String())
	}

	return plugin.Open(filepath.Join(tmpDir, soName))
}

func initModuleIfNone(tmpDir string) error {
	goModPath := tmpDir + "/go.mod"

	if _, err := os.Stat(goModPath); err == nil {
		// go.mod already exists, nothing to do
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check for go.mod: %v", err)
	}

	// Initialize module with a name
	cmd := exec.Command("go", "mod", "init", "callbacks")
	cmd.Dir = tmpDir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to initialize module: %s", out.String())
	}

	return nil
}
