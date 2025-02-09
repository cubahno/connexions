package internal

import (
	"strconv"
	"strings"

	"github.com/cubahno/connexions/internal/config"
)

// Document is an interface that represents an OpenAPI document needed for content generation.
// It is implemented by the LibV2Document and LibV3Document types.
type Document interface {
	GetVersion() string
	GetResources() map[string][]string
	GetSecurity() SecurityComponents
	FindOperation(options *OperationDescription) Operation
}

// Operation is an interface that represents an OpenAPI operation needed for content generation.
type Operation interface {
	ID() string
	Unwrap() Operation
	GetRequest(securityComponents SecurityComponents) *Request
	GetResponse() *Response
	WithParseConfig(*config.ParseConfig) Operation
}

type Request struct {
	Parameters Parameters
	Body       *RequestBody
}

// Response is a struct that represents an OpenAPI response.
type Response struct {
	Headers     Headers
	Content     *Schema
	ContentType string
	StatusCode  int
}

type SecurityComponents map[string]*SecurityComponent

type SecurityComponent struct {
	Type   AuthType
	Scheme AuthScheme
	In     AuthLocation
	Name   string
}

type AuthScheme string

const (
	AuthSchemeBearer AuthScheme = "bearer"
	AuthSchemeBasic  AuthScheme = "basic"
)

type AuthType string

const (
	AuthTypeHTTP   AuthType = "http"
	AuthTypeApiKey AuthType = "apiKey"
)

type AuthLocation string

const (
	AuthLocationHeader AuthLocation = "header"
	AuthLocationQuery  AuthLocation = "query"
)

// OperationDescription is a struct that used to find an operation in an OpenAPI document.
type OperationDescription struct {
	Service  string
	Resource string
	Method   string
}

// Parameter is a struct that represents an OpenAPI parameter.
type Parameter struct {
	Name     string  `json:"name,omitempty" yaml:"name,omitempty"`
	In       string  `json:"in,omitempty" yaml:"in,omitempty"`
	Required bool    `json:"required,omitempty" yaml:"required,omitempty"`
	Schema   *Schema `json:"schema,omitempty" yaml:"schema,omitempty"`
	Example  any     `json:"example,omitempty" yaml:"example,omitempty"`
}

// Parameters is a slice of Parameter.
type Parameters []*Parameter

// Headers is a map of Parameter.
type Headers map[string]*Parameter

type RequestBody struct {
	Schema *Schema
	Type   string
}

// Schema is a struct that represents an OpenAPI schema.
// It is compatible with all versions of OpenAPI.
// All schema provider should implement the Document and KinOperation interfaces.
// This provides a unified way to work with different OpenAPI parsers.
type Schema struct {
	Type string `json:"type,omitempty" yaml:"type,omitempty"`

	// in 3.1 examples can be an array (which is recommended)
	Examples []any `json:"examples,omitempty" yaml:"examples,omitempty"`

	// items can be a schema in 2.0, 3.0 and 3.1 or a bool in 3.1
	Items *Schema `json:"items,omitempty" yaml:"items,omitempty"`

	// Compatible with all versions
	MultipleOf    float64            `json:"multipleOf,omitempty" yaml:"multipleOf,omitempty"`
	Maximum       float64            `json:"maximum,omitempty" yaml:"maximum,omitempty"`
	Minimum       float64            `json:"minimum,omitempty" yaml:"minimum,omitempty"`
	MaxLength     int64              `json:"maxLength,omitempty" yaml:"maxLength,omitempty"`
	MinLength     int64              `json:"minLength,omitempty" yaml:"minLength,omitempty"`
	Pattern       string             `json:"pattern,omitempty" yaml:"pattern,omitempty"`
	Format        string             `json:"format,omitempty" yaml:"format,omitempty"`
	MaxItems      int64              `json:"maxItems,omitempty" yaml:"maxItems,omitempty"`
	MinItems      int64              `json:"minItems,omitempty" yaml:"minItems,omitempty"`
	MaxProperties int64              `json:"maxProperties,omitempty" yaml:"maxProperties,omitempty"`
	MinProperties int64              `json:"minProperties,omitempty" yaml:"minProperties,omitempty"`
	Required      []string           `json:"required,omitempty" yaml:"required,omitempty"`
	Enum          []any              `json:"enum,omitempty" yaml:"enum,omitempty"`
	Properties    map[string]*Schema `json:"properties,omitempty" yaml:"properties,omitempty"`
	Not           *Schema            `json:"not,omitempty" yaml:"not,omitempty"`
	Default       any                `json:"default,omitempty" yaml:"default,omitempty"`
	Nullable      bool               `json:"nullable,omitempty" yaml:"nullable,omitempty"`
	ReadOnly      bool               `json:"readOnly,omitempty" yaml:"readOnly,omitempty"`
	WriteOnly     bool               `json:"writeOnly,omitempty" yaml:"writeOnly,omitempty"`
	Example       any                `json:"example,omitempty" yaml:"example,omitempty"`
	Deprecated    bool               `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	// AdditionalProperties not used as they are merged into the properties map when present
}

const (
	TypeArray   = "array"
	TypeBoolean = "boolean"
	TypeInteger = "integer"
	TypeNumber  = "number"
	TypeObject  = "object"
	TypeString  = "string"
)

const (
	ParameterInPath   = "path"
	ParameterInQuery  = "query"
	ParameterInHeader = "header"
	// ParameterInBody v2 Swagger only
)

// FixSchemaTypeTypos fixes common typos in schema types.
func FixSchemaTypeTypos(typ string) string {
	switch typ {
	case "int":
		return TypeInteger
	case "float":
		return TypeNumber
	case "bool":
		return TypeBoolean
	}

	return typ
}

func GetOpenAPITypeFromValue(value any) string {
	switch value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return TypeInteger
	case float32, float64:
		return TypeNumber
	case bool:
		return TypeBoolean
	case string:
		return TypeString
	}

	return ""
}

// TransformHTTPCode transforms HTTP code from the OpenAPI spec to the real HTTP code.
func TransformHTTPCode(httpCode string) int {
	httpCode = strings.ToLower(httpCode)
	httpCode = strings.Replace(httpCode, "x", "0", -1)

	switch httpCode {
	case "*":
		fallthrough
	case "default":
		fallthrough
	case "000":
		return 200
	}

	codeInt, err := strconv.Atoi(httpCode)
	if err != nil {
		return 0
	}

	return codeInt
}
