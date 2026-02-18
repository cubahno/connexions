package api

import (
	"bytes"
	"embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"text/template"

	"github.com/cubahno/connexions/v2/cmd/gen/templatehelpers"
	"github.com/cubahno/connexions/v2/internal/files"
	"github.com/cubahno/connexions/v2/pkg/api"
	"github.com/cubahno/connexions/v2/pkg/config"
	"github.com/cubahno/connexions/v2/pkg/typedef"
	"github.com/doordash-oss/oapi-codegen-dd/v3/pkg/codegen"
	"go.yaml.in/yaml/v4"
)

//go:embed templates/*
var templatesFS embed.FS

// ServiceOptions contains options for generating service files.
type ServiceOptions struct {
	Name string // Optional: service name (default: inferred from directory name)

	// Optional: output directory for the service (default: current directory's parent)
	// When provided, setup dir will be <OutputDir>/setup/
	OutputDir string

	// Optional: path or URL to OpenAPI spec (default: openapi.yml/json in setup dir)
	SpecPath string

	// Optional: "openapi" or "static" (inferred from source if not provided)
	ServiceType string

	// Optional: maximum recursion depth for circular schemas
	MaxRecursionDepth int

	// Optional: maximum number of endpoints to process (0 = no limit, for debugging)
	MaxEndpoints int

	// Optional: path to custom codegen.yml to merge with template
	CodegenConfigPath string

	// Optional: path to custom config.yml to merge with template
	ServiceConfigPath string

	// Optional: suppress output messages (useful for automated tests)
	Quiet bool
}

// setupTemplateData contains data for rendering setup templates.
type setupTemplateData struct {
	ServiceName string
	SpecPath    string // Path or URL to OpenAPI spec (empty for local openapi.yml/json)
}

// GenerateService generates all service files (types, handlers, register.go, middleware.go).
// If OutputDir is provided, creates the service directory structure including setup/.
// If run from an existing setup/ directory, uses that directory for configuration.
// The function:
//   - Ensures setup directory exists with required files (creates if missing)
//   - Generates types in <service>/types/
//   - Generates handlers in <service>/handler/
//   - Generates register.go in <service>/
//   - Generates middleware.go in <service>/ (only if it doesn't exist)
//   - Infers the service name from the directory name
func GenerateService(opts ServiceOptions) error {
	var setupDir string
	var serviceDir string

	// Default output to current directory
	if opts.OutputDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting current directory: %w", err)
		}
		opts.OutputDir = cwd
	}

	// Setup will be at <OutputDir>/setup/
	serviceDir = opts.OutputDir
	setupDir = filepath.Join(opts.OutputDir, "setup")

	// Infer service name from directory name if not provided
	if opts.Name == "" {
		opts.Name = filepath.Base(serviceDir)
	}

	// Ensure setup directory exists with all required files
	if err := ensureSetupDir(opts, serviceDir, setupDir); err != nil {
		return fmt.Errorf("ensuring setup directory: %w", err)
	}

	// File paths relative to setup directory
	configFile := filepath.Join(setupDir, "codegen.yml")
	serviceConfigFile := filepath.Join(setupDir, "config.yml")

	// For static services, regenerate openapi.yml from the data directory
	// This ensures changes to static files are reflected in the spec
	if isStaticService(opts.SpecPath, opts.ServiceType) && opts.SpecPath != "" {
		specContents, err := generateSpecFromStaticDir(opts.SpecPath, opts.Name)
		if err != nil {
			return fmt.Errorf("generating spec from static directory %s: %w", opts.SpecPath, err)
		}

		specPath := filepath.Join(setupDir, "openapi.yml")
		if err := os.WriteFile(specPath, specContents, 0644); err != nil {
			return fmt.Errorf("writing generated spec to %s: %w", specPath, err)
		}
		if !opts.Quiet {
			fmt.Printf("Regenerated OpenAPI spec from static files: %s\n", specPath)
		}
	}

	// Find OpenAPI spec file
	// For static services, the spec was just regenerated above
	// For OpenAPI services with a URL or file path, use that directly
	var specFile string
	if opts.SpecPath != "" && !isStaticService(opts.SpecPath, opts.ServiceType) {
		// Use provided spec path/URL (only for non-static services)
		specFile = opts.SpecPath
	} else {
		// Fall back to local files (try both .yml and .json extensions)
		// This is also used for static services where the spec was generated
		specFile = filepath.Join(setupDir, "openapi.yml")
		if _, err := os.Stat(specFile); os.IsNotExist(err) {
			specFile = filepath.Join(setupDir, "openapi.json")
			if _, err := os.Stat(specFile); os.IsNotExist(err) {
				return fmt.Errorf("openapi spec file not found in %s (tried openapi.yml and openapi.json)", setupDir)
			}
		}
	}

	// Verify required files exist
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return fmt.Errorf("codegen.yml not found in %s", setupDir)
	}
	if _, err := os.Stat(serviceConfigFile); os.IsNotExist(err) {
		return fmt.Errorf("config.yml not found in %s", setupDir)
	}

	// Read the spec file
	specContents, err := files.ReadFileOrURL(specFile)
	if err != nil {
		return fmt.Errorf("reading spec file: %w", err)
	}

	// If spec was fetched from URL, save it locally for //go:embed to work
	if files.IsURL(specFile) {
		specExt := ".yml"
		if files.IsJsonType(specContents) {
			specExt = ".json"
		}
		localSpecPath := filepath.Join(setupDir, "openapi"+specExt)
		if err := os.WriteFile(localSpecPath, specContents, 0644); err != nil {
			return fmt.Errorf("saving spec file to %s: %w", localSpecPath, err)
		}
		if !opts.Quiet {
			fmt.Printf("Saved spec to: %s\n", localSpecPath)
		}
	}

	// Read the codegen config file
	cfg := codegen.Configuration{}
	codegenCfgContents, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("reading config file: %w", err)
	}
	if err = yaml.Unmarshal(codegenCfgContents, &cfg); err != nil {
		return fmt.Errorf("parsing config file: %w", err)
	}
	cfg = cfg.WithDefaults()

	// Read the service config file
	serviceCfgContents, err := os.ReadFile(serviceConfigFile)
	if err != nil {
		return fmt.Errorf("reading service config file: %w", err)
	}
	serviceCfg, err := config.NewServiceConfigFromBytes(serviceCfgContents)
	if err != nil {
		return fmt.Errorf("parsing service config: %w", err)
	}

	// Parse OpenAPI spec
	parseCtx, errs := typedef.CreateParseContext(specContents, cfg, serviceCfg.SpecOptions)
	if len(errs) > 0 {
		return fmt.Errorf("parsing OpenAPI spec: %v", errs[0])
	}
	if len(parseCtx.Operations) == 0 {
		slog.Warn("No operations found in spec")
	}

	// Determine output directories
	// The output directory in codegen.yml is relative to setup directory
	destDir := ""
	if cfg.Output != nil {
		// Make the output directory relative to setup directory, not current working directory
		destDir = filepath.Join(setupDir, cfg.Output.Directory)
		if err = os.MkdirAll(destDir, generatedDirPerm); err != nil {
			return fmt.Errorf("creating directory: %w", err)
		}
		if !cfg.Output.UseSingleFile {
			destDir = filepath.Join(destDir, cfg.PackageName)
			if err = os.MkdirAll(destDir, generatedDirPerm); err != nil {
				return fmt.Errorf("creating directory: %w", err)
			}
		}
	}

	if destDir == "" {
		return fmt.Errorf("no output directory specified")
	}

	// Get absolute path of service directory
	serviceDirAbs, err := filepath.Abs(serviceDir)
	if err != nil {
		return fmt.Errorf("getting absolute path: %w", err)
	}

	packageName := api.NormalizeServiceName(filepath.Base(serviceDirAbs))
	serviceName := packageName
	if opts.Name != "" {
		serviceName = opts.Name
	}

	// Step 1: Generate types
	slog.Info("Generating types...")
	if err := generateTypes(parseCtx, cfg, destDir, specContents); err != nil {
		return fmt.Errorf("generating types: %w", err)
	}
	slog.Info("Types generated")

	// Step 2: Generate handlers
	slog.Info("Generating handlers...")

	// Get module name and root directory from go.mod
	moduleName, moduleRoot, err := files.GetModuleInfo(serviceDirAbs)
	if err != nil {
		return fmt.Errorf("getting module info: %w", err)
	}

	// Get relative path from module root to service directory
	relPath, err := filepath.Rel(moduleRoot, serviceDirAbs)
	if err != nil {
		return fmt.Errorf("getting relative path: %w", err)
	}

	// Convert relative path to forward slashes for Go import paths
	relPath = filepath.ToSlash(relPath)

	// Construct the import path: moduleName/relPath/types
	typesImport := moduleName + "/" + relPath + "/types"
	serviceImport := moduleName + "/" + relPath

	// Fix operations with missing or invalid responses
	fixMissingResponses(parseCtx.Operations)

	// Fix duplicate path parameters (Chi router panics on duplicate param names)
	fixDuplicatePathParams(parseCtx.Operations)

	// Fix wildcard paths (Chi only allows * at the end of a route)
	fixWildcardPaths(parseCtx.Operations)

	// Limit endpoints for debugging if requested
	if opts.MaxEndpoints > 0 && len(parseCtx.Operations) > opts.MaxEndpoints {
		fmt.Fprintf(os.Stderr, "DEBUG: Limiting to first %d endpoints (out of %d total)\n", opts.MaxEndpoints, len(parseCtx.Operations))
		parseCtx.Operations = parseCtx.Operations[:opts.MaxEndpoints]
	}

	slog.Info("Generating TypeDefinitionRegistry", "service", serviceName, "endpoints", len(parseCtx.Operations))
	tdRegistry := typedef.NewTypeDefinitionRegistry(parseCtx, opts.MaxRecursionDepth, specContents)
	slog.Info("TypeDefinitionRegistry generated")

	opsCtx := &codegen.TplOperationsContext{
		Operations: parseCtx.Operations,
		Imports:    parseCtx.Imports,
		Config:     cfg,
		WithHeader: true,
	}

	codes, err := ParseHandlerTemplates(opsCtx, tdRegistry, serviceName, typesImport)
	if err != nil {
		return fmt.Errorf("generating handler code: %w", err)
	}

	// Create handler directory in service root
	handlerDir := filepath.Join(serviceDirAbs, "handler")
	if err = os.MkdirAll(handlerDir, generatedDirPerm); err != nil {
		return fmt.Errorf("creating handler directory: %w", err)
	}

	// Write handler files
	for fileName, code := range codes {
		formatted, err := codegen.FormatCode(code)
		if err != nil {
			return fmt.Errorf("formatting %s: %w", fileName, err)
		}

		savePath := filepath.Join(handlerDir, fileName)
		if err := os.WriteFile(savePath, []byte(formatted), generatedFilePerm); err != nil {
			return fmt.Errorf("writing file %s: %w", fileName, err)
		}
		fmt.Printf("Generated: %s\n", savePath)
	}

	slog.Info("Handlers generated")

	// Step 3: Generate register.go
	slog.Info("Generating register.go...")
	registerCode, err := generateRegisterCode(serviceName, packageName, serviceImport)
	if err != nil {
		return fmt.Errorf("generating register.go: %w", err)
	}

	formatted, err := codegen.FormatCode(registerCode)
	if err != nil {
		return fmt.Errorf("formatting register.go: %w", err)
	}

	registerPath := filepath.Join(serviceDirAbs, ServiceRegistrationFile)
	if err := os.WriteFile(registerPath, []byte(formatted), generatedFilePerm); err != nil {
		return fmt.Errorf("writing %s: %w", ServiceRegistrationFile, err)
	}
	fmt.Printf("Generated: %s\n", registerPath)

	// Step 4: Generate middleware.go (only if it doesn't exist)
	middlewarePath := filepath.Join(serviceDirAbs, "middleware.go")
	if _, err := os.Stat(middlewarePath); os.IsNotExist(err) {
		slog.Info("Generating middleware.go...")
		middlewareCode, err := generateMiddlewareCode(serviceName, packageName)
		if err != nil {
			return fmt.Errorf("generating middleware.go: %w", err)
		}

		formatted, err := codegen.FormatCode(middlewareCode)
		if err != nil {
			return fmt.Errorf("formatting middleware.go: %w", err)
		}

		if err := os.WriteFile(middlewarePath, []byte(formatted), generatedFilePerm); err != nil {
			return fmt.Errorf("writing middleware.go: %w", err)
		}
		fmt.Printf("Generated: %s\n", middlewarePath)
	} else {
		slog.Info("Skipping middleware.go (already exists, preserving user edits)")
	}

	// Step 5: Generate server/main.go (only if enabled in config)
	if serviceCfg.Generate != nil && serviceCfg.Generate.Server != nil {
		slog.Info("Generating server/main.go...")
		serverDir := filepath.Join(serviceDirAbs, "server")
		if err = os.MkdirAll(serverDir, generatedDirPerm); err != nil {
			return fmt.Errorf("creating server directory: %w", err)
		}

		serverCode, err := generateServerCode(serviceName, packageName, serviceImport)
		if err != nil {
			return fmt.Errorf("generating server/main.go: %w", err)
		}

		formatted, err = codegen.FormatCode(serverCode)
		if err != nil {
			return fmt.Errorf("formatting server/main.go: %w", err)
		}

		serverPath := filepath.Join(serverDir, "main.go")
		if err := os.WriteFile(serverPath, []byte(formatted), generatedFilePerm); err != nil {
			return fmt.Errorf("writing server/main.go: %w", err)
		}
		fmt.Printf("Generated: %s\n", serverPath)
	}

	slog.Info("Service generation complete")
	return nil
}

// generateServerCode generates the server/main.go file content for standalone service server
func generateServerCode(serviceName, packageName, serviceImport string) (string, error) {
	tmplContent, err := templatesFS.ReadFile("templates/handler/server.tmpl")
	if err != nil {
		return "", fmt.Errorf("reading template: %w", err)
	}

	funcMap := templatehelpers.GetFuncMap()

	tmpl, err := template.New("server").Funcs(funcMap).Parse(string(tmplContent))
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}

	data := struct {
		PackageName   string
		ServiceName   string
		ServiceImport string
	}{
		PackageName:   packageName,
		ServiceName:   serviceName,
		ServiceImport: serviceImport,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}

	return buf.String(), nil
}

// generateRegisterCode generates the register.go file content
func generateRegisterCode(serviceName, packageName, serviceImport string) (string, error) {
	tmplContent, err := templatesFS.ReadFile("templates/handler/register.tmpl")
	if err != nil {
		return "", fmt.Errorf("reading template: %w", err)
	}

	funcMap := templatehelpers.GetFuncMap()

	tmpl, err := template.New("register").Funcs(funcMap).Parse(string(tmplContent))
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}

	data := struct {
		PackageName   string
		ServiceName   string
		ServiceImport string
	}{
		PackageName:   packageName,
		ServiceName:   serviceName,
		ServiceImport: serviceImport,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}

	return buf.String(), nil
}

// generateMiddlewareCode generates the middleware.go file content
func generateMiddlewareCode(serviceName, packageName string) (string, error) {
	tmplContent, err := templatesFS.ReadFile("templates/handler/middleware.tmpl")
	if err != nil {
		return "", fmt.Errorf("reading template: %w", err)
	}

	funcMap := templatehelpers.GetFuncMap()

	tmpl, err := template.New("middleware").Funcs(funcMap).Parse(string(tmplContent))
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}

	data := struct {
		PackageName string
		ServiceName string
	}{
		PackageName: packageName,
		ServiceName: serviceName,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}

	return buf.String(), nil
}

// ensureSetupDir creates the setup directory with all necessary files if it doesn't exist.
// generate.go is placed in serviceDir (service root), other config files go in setupDir.
func ensureSetupDir(opts ServiceOptions, serviceDir, setupDir string) error {
	// Check if setup dir already exists with required files
	configFile := filepath.Join(setupDir, "codegen.yml")
	serviceConfigFile := filepath.Join(setupDir, "config.yml")
	generateFile := filepath.Join(serviceDir, "generate.go") // generate.go goes in service root

	// Check if all required files exist
	configExists := fileExists(configFile)
	serviceConfigExists := fileExists(serviceConfigFile)
	generateExists := fileExists(generateFile)

	// If all required files exist, setup is complete
	if configExists && serviceConfigExists && generateExists {
		return nil
	}

	// Normalize service name
	serviceName := api.NormalizeServiceName(opts.Name)
	if serviceName == "" {
		return fmt.Errorf("service name is required for initial setup")
	}

	// Determine service type
	serviceType := opts.ServiceType
	if opts.SpecPath != "" {
		// Check if source is a URL
		isURL := files.IsURL(opts.SpecPath)

		if isURL {
			// URLs are always OpenAPI specs
			inferredType := "openapi"
			if serviceType != "" && serviceType != inferredType {
				return fmt.Errorf("type mismatch: URLs must be OpenAPI specs, but type was specified as %s", serviceType)
			}
			serviceType = inferredType
		} else {
			// Infer type from local file/directory
			info, err := os.Stat(opts.SpecPath)
			if err != nil {
				return fmt.Errorf("accessing source path %s: %w", opts.SpecPath, err)
			}
			inferredType := "openapi"
			if info.IsDir() {
				inferredType = "static"
			}

			// Validate against provided type if specified
			if serviceType != "" && serviceType != inferredType {
				return fmt.Errorf("type mismatch: specified %s but source appears to be %s", serviceType, inferredType)
			}
			serviceType = inferredType
		}
	} else {
		// No source - default to openapi
		if serviceType == "" {
			serviceType = "openapi"
		}
		if serviceType != "openapi" && serviceType != "static" {
			return fmt.Errorf("service type must be 'openapi' or 'static', got: %s", serviceType)
		}
	}

	// Get template files
	tplFiles, err := getSetupTemplateFiles(serviceType)
	if err != nil {
		return fmt.Errorf("getting templates: %w", err)
	}

	// Create directory structure
	if err := os.MkdirAll(setupDir, 0755); err != nil {
		return fmt.Errorf("creating setup directory: %w", err)
	}

	// Prepare template data
	// For URLs, we keep the URL as SpecPath so it's embedded in generate.go
	// For static services, we keep the directory path so regeneration works
	// For local OpenAPI files, we leave SpecPath empty (will fallback to local openapi.yml/json)
	specPath := ""
	if files.IsURL(opts.SpecPath) {
		specPath = opts.SpecPath
	} else if serviceType == "static" && opts.SpecPath != "" {
		specPath = opts.SpecPath
	}
	data := setupTemplateData{
		ServiceName: serviceName,
		SpecPath:    specPath,
	}

	// Render and write template files
	for filename, templatePath := range tplFiles {
		// Skip openapi.yml template if we have a source file (we'll copy it later)
		if filename == "openapi.yml" && opts.SpecPath != "" {
			continue
		}

		var content []byte
		var renderErr error

		// Check if it's a template file (.tmpl) or a static file
		if filepath.Ext(templatePath) == ".tmpl" {
			// Render template
			rendered, err := renderSetupTemplate(templatePath, data)
			if err != nil {
				return fmt.Errorf("rendering template %s: %w", templatePath, err)
			}
			content = []byte(rendered)
		} else {
			// Copy static file as-is
			content, renderErr = templatesFS.ReadFile(templatePath)
			if renderErr != nil {
				return fmt.Errorf("reading file %s: %w", templatePath, renderErr)
			}
		}

		// generate.go goes in service root, other files go in setup dir
		var path string
		if filename == "generate.go" {
			path = filepath.Join(serviceDir, filename)
		} else {
			path = filepath.Join(setupDir, filename)
		}

		// Ensure directory exists before writing
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return fmt.Errorf("creating directory for %s: %w", filename, err)
		}
		if err := os.WriteFile(path, content, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", filename, err)
		}
		if !opts.Quiet {
			fmt.Printf("Generated: %s\n", path)
		}
	}

	// Merge custom codegen config if provided
	if opts.CodegenConfigPath != "" {
		currentPath := filepath.Join(setupDir, "codegen.yml")
		err := mergeSetupYAMLConfigs(opts.CodegenConfigPath, currentPath, "codegen", func(templateData, customData []byte) ([]byte, error) {
			var templateCfg, customCfg codegen.Configuration
			if err := yaml.Unmarshal(templateData, &templateCfg); err != nil {
				return nil, fmt.Errorf("parsing template codegen config: %w", err)
			}
			if err := yaml.Unmarshal(customData, &customCfg); err != nil {
				return nil, fmt.Errorf("parsing custom codegen config: %w", err)
			}

			mergedCfg := templateCfg.WithDefaults().OverwriteWith(customCfg)
			return yaml.Marshal(mergedCfg)
		})
		if err != nil {
			return err
		}
	}

	// Merge custom service config if provided
	if opts.ServiceConfigPath != "" {
		currentPath := filepath.Join(setupDir, "config.yml")
		err := mergeSetupYAMLConfigs(opts.ServiceConfigPath, currentPath, "service", func(templateData, customData []byte) ([]byte, error) {
			templateCfg, err := config.NewServiceConfigFromBytes(templateData)
			if err != nil {
				return nil, fmt.Errorf("parsing template service config: %w", err)
			}
			customCfg, err := config.NewServiceConfigFromBytes(customData)
			if err != nil {
				return nil, fmt.Errorf("parsing custom service config: %w", err)
			}

			mergedCfg := templateCfg.OverwriteWith(customCfg)
			return yaml.Marshal(mergedCfg)
		})
		if err != nil {
			return err
		}
	}

	// Handle source based on type
	if err := handleSetupSourceSpec(opts, setupDir, serviceName, serviceType); err != nil {
		return err
	}

	if !opts.Quiet {
		fmt.Printf("\n✅ Setup directory created at %s\n", setupDir)
	}

	return nil
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// isStaticService checks if the service is a static service based on the spec path and service type.
func isStaticService(specPath, serviceType string) bool {
	if serviceType == "static" {
		return true
	}
	// If no explicit type, check if the path is a directory
	if specPath != "" && !files.IsURL(specPath) {
		info, err := os.Stat(specPath)
		if err == nil && info.IsDir() {
			return true
		}
	}
	return false
}

// handleSetupSourceSpec handles the source spec based on type (static, URL, or local file).
func handleSetupSourceSpec(opts ServiceOptions, setupDir, serviceName, serviceType string) error {
	if opts.SpecPath == "" {
		return nil
	}

	if serviceType == "static" {
		specContents, err := generateSpecFromStaticDir(opts.SpecPath, serviceName)
		if err != nil {
			return fmt.Errorf("generating spec from static directory %s: %w", opts.SpecPath, err)
		}
		specPath := filepath.Join(setupDir, "openapi.yml")
		if err := os.WriteFile(specPath, specContents, 0644); err != nil {
			return fmt.Errorf("writing generated spec to %s: %w", specPath, err)
		}
		if !opts.Quiet {
			fmt.Printf("Generated OpenAPI spec from static files: %s\n", specPath)
		}
		return nil
	}

	if files.IsURL(opts.SpecPath) {
		if !opts.Quiet {
			fmt.Printf("Using remote spec URL: %s\n", opts.SpecPath)
		}
		return nil
	}

	// Local file - copy it
	specContents, err := files.ReadFileOrURL(opts.SpecPath)
	if err != nil {
		return fmt.Errorf("reading spec file from %s: %w", opts.SpecPath, err)
	}

	specExt := ".yml"
	if files.IsJsonType(specContents) {
		specExt = ".json"
	}

	specPath := filepath.Join(setupDir, "openapi"+specExt)
	if err := os.MkdirAll(filepath.Dir(specPath), 0755); err != nil {
		return fmt.Errorf("creating directory for spec file: %w", err)
	}

	if err := os.WriteFile(specPath, specContents, 0644); err != nil {
		return fmt.Errorf("writing spec file to %s: %w", specPath, err)
	}

	if !opts.Quiet {
		fmt.Printf("Copied spec file to: %s\n", specPath)
	}
	return nil
}

// getSetupTemplateFiles returns the appropriate template files based on service type.
func getSetupTemplateFiles(serviceType string) (map[string]string, error) {
	// Determine which openapi template to use
	openapiTemplate := "templates/setup/openapi.yml"
	if serviceType == "static" {
		openapiTemplate = "templates/setup/openapi.static.yml"
	}

	templates := map[string]string{
		"generate.go": "templates/setup/generate.go.tmpl",
		"codegen.yml": "templates/setup/codegen.yml",
		"config.yml":  "templates/setup/config.yml.tmpl",
		"context.yml": "templates/setup/context.yml",
		"openapi.yml": openapiTemplate,
	}

	return templates, nil
}

// mergeSetupYAMLConfigs reads custom and template YAML files, unmarshals them,
// merges using the provided merge function, and writes back the result
func mergeSetupYAMLConfigs(customPath, templatePath, configType string, mergeFn func(templateData, customData []byte) ([]byte, error)) error {
	customContent, err := os.ReadFile(customPath)
	if err != nil {
		return fmt.Errorf("reading custom %s config from %s: %w", configType, customPath, err)
	}

	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("reading template %s config: %w", configType, err)
	}

	mergedContent, err := mergeFn(templateContent, customContent)
	if err != nil {
		return err
	}

	if err := os.WriteFile(templatePath, mergedContent, 0644); err != nil {
		return fmt.Errorf("writing merged %s config to %s: %w", configType, templatePath, err)
	}

	return nil
}

// renderSetupTemplate renders a setup template file with the given data.
func renderSetupTemplate(templatePath string, data setupTemplateData) (string, error) {
	content, err := templatesFS.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("reading template file: %w", err)
	}

	tmpl, err := template.New(templatePath).Funcs(templatehelpers.GetFuncMap()).Parse(string(content))
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}

	return buf.String(), nil
}
