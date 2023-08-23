package connexions

import (
	"context"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/pb33f/libopenapi"
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
	"net/http"
	"os"
)

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

func NewContentWithMultipartFormDataSchema(schema *Schema) OpenAPIContent {
	return OpenAPIContent{
		"multipart/form-data": NewMediaType().WithSchema(schema),
	}
}

func NewMediaType() *MediaType {
	return &MediaType{}
}

func ValidateRequest(req *http.Request, body *RequestBody) error {
	inp := &openapi3filter.RequestValidationInput{Request: req}
	return openapi3filter.ValidateRequestBody(context.Background(), inp, body)
}

func ValidateResponse(req *http.Request, res *Response, operation *Operation) error {
	inp := &openapi3filter.RequestValidationInput{
		Request: req,
		Route: &routers.Route{
			Method:    req.Method,
			Operation: operation,
		},
	}
	responseValidationInput := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: inp,
		Status:                 res.StatusCode,
		Header:                 res.Headers,
	}

	responseValidationInput.SetBodyBytes(res.Content)
	return openapi3filter.ValidateResponse(context.Background(), responseValidationInput)
}

func NewLibDocument(filePath string) (libopenapi.Document, error) {
	src, _ := os.ReadFile(filePath)

	// create a new document from specification bytes
	return libopenapi.NewDocument(src)
}

func NewLibModel(doc libopenapi.Document) (*libopenapi.DocumentModel[v3high.Document], []error) {
	return doc.BuildV3Model()
}
