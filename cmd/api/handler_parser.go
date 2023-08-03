package api

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	"sort"
	"text/template"

	"github.com/cubahno/connexions/v2/cmd/gen/templatehelpers"
	"github.com/cubahno/connexions/v2/pkg/api"
	"github.com/cubahno/connexions/v2/pkg/typedef"
	"github.com/doordash-oss/oapi-codegen-dd/v3/pkg/codegen"
)

// ParseHandlerTemplates generates handler code from operations context and type registry
func ParseHandlerTemplates(data *codegen.TplOperationsContext, tdRegistry *typedef.TypeDefinitionRegistry, serviceName, typesImport string) (map[string]string, error) {
	// Template files to load
	templateFiles := []string{
		"templates/handler/handler.tmpl",
		"templates/handler/service.tmpl",
		"templates/handler/registry.tmpl",
		"templates/handler/errors.tmpl",
	}

	tdsLookUp := tdRegistry.GetTypeDefinitionLookup()

	// Get base template functions and add our custom ones
	customFuncs := template.FuncMap{
		"renderSchema":    renderSchema,
		"resolveTypeName": createResolveTypeName(tdsLookUp),
		"renderEncoding":  renderParameterEncoding,
	}
	fns := templatehelpers.GetFuncMapWithCustom(customFuncs)
	tpl := template.New("handler-templates").Funcs(fns)

	// Load and parse all templates
	for _, tmplPath := range templateFiles {
		tmplBytes, err := templatesFS.ReadFile(tmplPath)
		if err != nil {
			return nil, fmt.Errorf("reading template %s: %w", tmplPath, err)
		}

		// Extract template name from path (e.g., "handler.tmpl" from "templates/handler/handler.tmpl")
		tmplName := tmplPath[len("templates/handler/"):]
		tmplInstance := tpl.New(tmplName)
		if _, err := tmplInstance.Parse(string(tmplBytes)); err != nil {
			return nil, fmt.Errorf("parsing template '%s': %w", tmplName, err)
		}
	}

	allOperations := tdRegistry.Operations()

	// Sort operations by path and method to match Routes() output
	sortedOperations := make([]codegen.OperationDefinition, len(data.Operations))
	copy(sortedOperations, data.Operations)
	sortOperations(sortedOperations)

	res := make(map[string]string)

	// Template context for all files
	tplContext := map[string]any{
		"Operations":         sortedOperations,
		"RegistryOperations": allOperations,
		"Config":             data.Config,
		"ServiceName":        serviceName,
		"TypesImport":        typesImport,
	}

	// Generate handler, service, registry builder, and errors files
	fileMapping := map[string]string{
		"handler.tmpl":  "handler.go",
		"service.tmpl":  "service.go",
		"registry.tmpl": "registry.go",
		"errors.tmpl":   "errors.go",
	}

	for tmplName, outputName := range fileMapping {
		var buf bytes.Buffer
		w := bufio.NewWriter(&buf)

		if err := tpl.ExecuteTemplate(w, tmplName, tplContext); err != nil {
			return nil, fmt.Errorf("error generating %s: %s", tmplName, err)
		}
		if err := w.Flush(); err != nil {
			return nil, fmt.Errorf("error flushing output buffer for %s: %s", tmplName, err.Error())
		}

		res[outputName] = buf.String()
	}

	return res, nil
}

// sortOperations sorts operations by path and method to match the Routes() Sort() behavior.
// The order is: path (alphabetically), then method (GET, POST, others alphabetically).
func sortOperations(ops []codegen.OperationDefinition) {
	sort.SliceStable(ops, func(i, j int) bool {
		return api.ComparePathMethod(ops[i].Path, ops[i].Method, ops[j].Path, ops[j].Method)
	})
}
