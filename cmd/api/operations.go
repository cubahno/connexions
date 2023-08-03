package api

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cubahno/connexions/v2/internal/types"
	"github.com/doordash-oss/oapi-codegen-dd/v3/pkg/codegen"
)

const (
	// ServiceRegistrationFile is the name of the file that contains service registration code
	ServiceRegistrationFile = "register.go"

	// File and directory permissions for generated code
	generatedDirPerm  = 0755
	generatedFilePerm = 0644
)

// fixMissingResponses adds default responses to operations that are missing them
func fixMissingResponses(operations []codegen.OperationDefinition) {
	for i := range operations {
		op := &operations[i]

		// If no success response, create a default one
		if op.Response.Success == nil {
			op.Response.SuccessStatusCode = 200
			op.Response.Success = &codegen.ResponseContentDefinition{
				ResponseName: "any",
				ContentType:  "application/json",
				IsSuccess:    true,
				StatusCode:   200,
				Schema: codegen.GoSchema{
					GoType: "any",
				},
			}

			// Also add to All map
			if op.Response.All == nil {
				op.Response.All = make(map[int]*codegen.ResponseContentDefinition)
			}
			op.Response.All[200] = op.Response.Success
		}

		// If success response has no content type, default to application/json
		if op.Response.Success != nil && op.Response.Success.ContentType == "" {
			op.Response.Success.ContentType = "application/json"
		}
	}
}

// fixDuplicatePathParams renames duplicate path parameters in operation paths.
// Chi router panics when registering routes with duplicate parameter names like /foo/{id}/bar/{id}.
// This function renames duplicates to /foo/{id}/bar/{id_2}.
func fixDuplicatePathParams(operations []codegen.OperationDefinition) {
	for i := range operations {
		op := &operations[i]
		op.Path = types.DeduplicatePathParams(op.Path)
	}
}

// fixWildcardPaths converts OpenAPI wildcard paths to Chi-compatible format.
// Chi only allows * at the end of a route.
func fixWildcardPaths(operations []codegen.OperationDefinition) {
	for i := range operations {
		op := &operations[i]
		op.Path = types.SanitizePathForChi(op.Path)
	}
}

// generateTypes generates type definitions from a parse context
func generateTypes(parseCtx *codegen.ParseContext, cfg codegen.Configuration, destDir string, specContents []byte) error {
	parser, err := codegen.NewParser(cfg, parseCtx)
	if err != nil {
		return fmt.Errorf("creating parser: %w", err)
	}

	codes, err := parser.Parse()
	if err != nil {
		return fmt.Errorf("parsing: %w", err)
	}

	// Clean the types directory before writing new files to remove stale files
	// (e.g., unions.go when simplify removes all union types)
	if err := os.RemoveAll(destDir); err != nil {
		return fmt.Errorf("cleaning types directory: %w", err)
	}
	if err := os.MkdirAll(destDir, generatedDirPerm); err != nil {
		return fmt.Errorf("creating types directory: %w", err)
	}

	// Write generated code
	for fileName, code := range codes {
		savePath := filepath.Join(destDir, fileName+".go")
		if err := os.WriteFile(savePath, []byte(code), generatedFilePerm); err != nil {
			return fmt.Errorf("writing file %s: %w", fileName, err)
		}
	}

	return nil
}
