package schema

import (
	"encoding/json"
	"net/http"
)

// RequestSchema is a struct that represents an OpenAPI request needed to generate a response.
type RequestSchema struct {
	URL     string
	Method  string
	Path    *Path
	Body    *Body
	Query   *Query
	Headers *Headers
}

// ParameterType represents the type/location of a parameter.
type ParameterType string

const (
	ParameterTypePath   ParameterType = "path"
	ParameterTypeQuery  ParameterType = "query"
	ParameterTypeHeader ParameterType = "header"
)

// Parameter represents a single parameter with its type and value.
type Parameter struct {
	Type  ParameterType
	Value any
}

// RequestData represents the actual parsed request data from an HTTP request.
type RequestData struct {
	Method     string                // HTTP method (GET, POST, PUT, etc.)
	ResourceID string                // The operation path pattern with placeholders (e.g., /users/{id})
	Params     map[string]*Parameter // All parameters (path, query, header) with their types
	Body       any                   // Parsed request body (can be map[string]any for objects, []any for arrays, or primitives)
}

// ResponseSchema is a struct that represents a schema needed to generate a response.
type ResponseSchema struct {
	ContentType string
	Body        *Schema
	Headers     map[string]*Schema
	Error       *Schema
}

// ResponseData is a struct that represents a generated response.
type ResponseData struct {
	Body    json.RawMessage `json:"body,omitempty"`
	Headers http.Header     `json:"headers,omitempty"`
	IsError bool            `json:"isError,omitempty"`
}

// Path is a struct that represents a path parameter.
type Path struct {
	Value   string
	Schema  *Schema
	Pattern string
}

// Body is a struct that represents a request body.
type Body struct {
	Value  json.RawMessage
	Schema *Schema
}

// Query is a struct that represents a query parameter.
type Query struct {
	Value  string
	Schema *Schema
}

// Headers is a struct that represents a header parameter.
type Headers struct {
	Value  http.Header
	Schema *Schema
}
