package openapi

import (
	"strconv"
	"strings"

	"github.com/cubahno/connexions/internal/config"
	"github.com/cubahno/connexions/internal/types"
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
	Content     *types.Schema
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
	Name     string        `json:"name,omitempty" yaml:"name,omitempty"`
	In       string        `json:"in,omitempty" yaml:"in,omitempty"`
	Required bool          `json:"required,omitempty" yaml:"required,omitempty"`
	Schema   *types.Schema `json:"schema,omitempty" yaml:"schema,omitempty"`
	Example  any           `json:"example,omitempty" yaml:"example,omitempty"`
}

// Parameters is a slice of Parameter.
type Parameters []*Parameter

// Headers is a map of Parameter.
type Headers map[string]*Parameter

type RequestBody struct {
	Schema *types.Schema
	Type   string
}

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
		return types.TypeInteger
	case "float":
		return types.TypeNumber
	case "bool":
		return types.TypeBoolean
	}

	return typ
}

func GetOpenAPITypeFromValue(value any) string {
	switch value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return types.TypeInteger
	case float32, float64:
		return types.TypeNumber
	case bool:
		return types.TypeBoolean
	case string:
		return types.TypeString
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
