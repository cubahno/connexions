package types

import "strings"

const (
	TypeString  = "string"
	TypeInteger = "integer"
	TypeNumber  = "number"
	TypeBoolean = "boolean"
	TypeObject  = "object"
	TypeArray   = "array"
)

// GoTypeToOpenAPIType converts a Go type string to an OpenAPI type string.
// This is used when converting generated Go types back to OpenAPI schema types.
func GoTypeToOpenAPIType(goType string) string {
	// Check for array types first (e.g., []string, []SomeType)
	if strings.HasPrefix(goType, "[]") {
		return TypeArray
	}

	switch goType {
	case "string":
		return TypeString
	case "bool":
		return TypeBoolean
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64":
		return TypeInteger
	case "float32", "float64":
		return TypeNumber
	case "any", "interface{}":
		// Default to string for any/interface{} types
		return TypeString
	case "struct{}":
		// struct{} is used by oapi-codegen for empty schemas (no type, no properties)
		// In OpenAPI, an empty schema means "any value is valid"
		// For data generation, we treat this as 'any' type
		return "any"
	default:
		// For complex types or unknown types, default to object
		// (e.g., custom structs, maps with additionalProperties)
		return TypeObject
	}
}
