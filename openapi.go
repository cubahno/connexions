package xs

import "github.com/getkin/kin-openapi/openapi3"

type Document struct {
	*openapi3.T
}

type (
	Schema            = openapi3.Schema
	Operation         = openapi3.Operation
	OpenAPIResponse   = openapi3.Response
	ResponseRef       = openapi3.ResponseRef
	OpenAPIContent    = openapi3.Content
	OpenAPIParameter  = openapi3.Parameter
	OpenAPIParameters = openapi3.Parameters
	SchemaRef         = openapi3.SchemaRef
	SchemaRefs        = openapi3.SchemaRefs
	RequestBody       = openapi3.RequestBody
	RequestBodyRef    = openapi3.RequestBodyRef
	OpenAPIHeader     = openapi3.Header
	OpenAPIHeaders    = openapi3.Headers
	MediaType         = openapi3.MediaType
)

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
	ParameterInCookie = "cookie"
)

func NewDocumentFromFile(filePath string) (*Document, error) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(filePath)
	if err != nil {
		return nil, err
	}
	return &Document{
		T: doc,
	}, err
}

func NewDocumentFromString(src string) (*Document, error) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData([]byte(src))
	if err != nil {
		return nil, err
	}
	return &Document{doc}, nil
}

func NewContentWithJSONSchema(schema *Schema) OpenAPIContent {
	return OpenAPIContent{
		"application/json": NewMediaType().WithSchema(schema),
	}
}

func NewMediaType() *MediaType {
	return &MediaType{}
}
