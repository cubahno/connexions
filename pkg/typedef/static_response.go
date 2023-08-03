package typedef

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/doordash-oss/oapi-codegen-dd/v3/pkg/codegen"
)

const extStaticResponse = "x-static-response"

// StaticResponseKey is a unique key for looking up static responses.
// Format: "METHOD /path 200" (e.g., "GET /users 200")
type StaticResponseKey string

// NewStaticResponseKey creates a key for looking up static responses.
func NewStaticResponseKey(method, path string, statusCode int) StaticResponseKey {
	return StaticResponseKey(fmt.Sprintf("%s %s %d", method, path, statusCode))
}

// ExtractStaticResponses extracts all x-static-response values from an OpenAPI spec.
// Returns a map keyed by "METHOD /path statusCode" -> static response content.
func ExtractStaticResponses(specBytes []byte) (map[StaticResponseKey]string, error) {
	doc, err := codegen.LoadDocumentFromContents(specBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI document: %w", err)
	}

	builtModel, err := doc.BuildV3Model()
	if err != nil {
		return nil, fmt.Errorf("failed to build OpenAPI model: %w", err)
	}

	model := &builtModel.Model
	if model == nil || model.Paths == nil || model.Paths.PathItems == nil {
		return make(map[StaticResponseKey]string), nil
	}

	staticResponses := make(map[StaticResponseKey]string)

	for path, pathItem := range model.Paths.PathItems.FromOldest() {
		for method, operation := range pathItem.GetOperations().FromOldest() {
			if operation.Responses == nil || operation.Responses.Codes == nil {
				continue
			}

			for statusCodeStr, response := range operation.Responses.Codes.FromOldest() {
				if response.Content == nil {
					continue
				}

				for _, mediaType := range response.Content.FromOldest() {
					if mediaType.Extensions == nil || mediaType.Extensions.Len() == 0 {
						continue
					}

					// Look for x-static-response extension
					extNode := mediaType.Extensions.Value(extStaticResponse)
					if extNode == nil {
						continue
					}

					// Extract the string value from the YAML node
					staticResponse := strings.TrimSpace(extNode.Value)
					if staticResponse == "" {
						continue
					}

					// Parse status code
					statusCode, err := strconv.Atoi(statusCodeStr)
					if err != nil {
						// Skip non-numeric status codes like "default"
						continue
					}

					// Create key and store (method must be uppercase to match OperationDefinition)
					key := NewStaticResponseKey(strings.ToUpper(method), path, statusCode)
					staticResponses[key] = staticResponse
				}
			}
		}
	}

	return staticResponses, nil
}
