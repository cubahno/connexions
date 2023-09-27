package connexions

// Document is an interface that represents an OpenAPI document needed for content generation.
// It is implemented by the LibV2Document and LibV3Document types.
type Document interface {
	Provider() SchemaProvider
	GetVersion() string
	GetResources() map[string][]string
	FindOperation(options *OperationDescription) Operationer
}

// Operationer is an interface that represents an OpenAPI operation needed for content generation.
type Operationer interface {
	ID() string
	GetParameters() OpenAPIParameters
	GetRequestBody() (*Schema, string)
	GetResponse() *OpenAPIResponse
	WithParseConfig(*ParseConfig) Operationer
}

// OpenAPIResponse is a struct that represents an OpenAPI response.
type OpenAPIResponse struct {
	Headers     OpenAPIHeaders
	Content     *Schema
	ContentType string
	StatusCode  int
}

// OperationDescription is a struct that used to find an operation in an OpenAPI document.
type OperationDescription struct {
	Service  string
	Resource string
	Method   string
}

// OpenAPIParameter is a struct that represents an OpenAPI parameter.
type OpenAPIParameter struct {
	Name     string      `json:"name,omitempty" yaml:"name,omitempty"`
	In       string      `json:"in,omitempty" yaml:"in,omitempty"`
	Required bool        `json:"required,omitempty" yaml:"required,omitempty"`
	Schema   *Schema     `json:"schema,omitempty" yaml:"schema,omitempty"`
	Example  interface{} `json:"example,omitempty" yaml:"example,omitempty"`
}

// OpenAPIParameters is a slice of OpenAPIParameter.
type OpenAPIParameters []*OpenAPIParameter

// OpenAPIHeaders is a map of OpenAPIParameter.
type OpenAPIHeaders map[string]*OpenAPIParameter

// Schema is a struct that represents an OpenAPI schema.
// It is compatible with all versions of OpenAPI.
// All schema providers should implement the Document and Operationer interfaces.
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
	ParameterInBody = "body"
)

// NewDocumentFromFileFactory returns a function that creates a new Document from a file.
func NewDocumentFromFileFactory(provider SchemaProvider) func(filePath string) (Document, error) {
	switch provider {
	case KinOpenAPIProvider:
		return NewKinDocumentFromFile
	case LibOpenAPIProvider:
		return NewLibOpenAPIDocumentFromFile
	default:
		return NewLibOpenAPIDocumentFromFile
	}
}

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
