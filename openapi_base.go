package connexions

type Document interface {
	GetVersion() string
	GetResources() map[string][]string
	FindOperation(options *FindOperationOptions) Operationer
}

type Operationer interface {
	GetParameters() OpenAPIParameters
	GetRequestBody() (*Schema, string)
	GetResponse() *OpenAPIResponse
	WithParseConfig(*ParseConfig) Operationer
}

type OpenAPIResponse struct {
	Headers     OpenAPIHeaders
	Content     *Schema
	ContentType string
	StatusCode  int
}

type FindOperationOptions struct {
	Service     string
	Resource    string
	Method      string
	ParseConfig *ParseConfig
}

type OpenAPIParameter struct {
	Name     string      `json:"name,omitempty" yaml:"name,omitempty"`
	In       string      `json:"in,omitempty" yaml:"in,omitempty"`
	Required bool        `json:"required,omitempty" yaml:"required,omitempty"`
	Schema   *Schema     `json:"schema,omitempty" yaml:"schema,omitempty"`
	Example  interface{} `json:"example,omitempty" yaml:"example,omitempty"`
}

type OpenAPIParameters []*OpenAPIParameter
type OpenAPIHeaders map[string]*OpenAPIParameter

type BaseOperation struct {
}

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
	Default       any                `json:"default,omitempty" yaml:"default,omitempty"`
	Nullable      bool               `json:"nullable,omitempty" yaml:"nullable,omitempty"`
	ReadOnly      bool               `json:"readOnly,omitempty" yaml:"readOnly,omitempty"`
	WriteOnly     bool               `json:"writeOnly,omitempty" yaml:"writeOnly,omitempty"`
	Example       any                `json:"example,omitempty" yaml:"example,omitempty"`
	Deprecated    bool               `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
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
)

func NewDocumentFromFileFactory(provider SchemaProvider) func(string) (Document, error) {
	switch provider {
	case KinOpenAPIProvider:
		return NewKinDocumentFromFile
	case LibOpenAPIProvider:
		return NewLibOpenAPIDocumentFromFile
	default:
		return NewLibOpenAPIDocumentFromFile
	}
}

func (op *BaseOperation) WithCache() Operationer {
	return nil
}
