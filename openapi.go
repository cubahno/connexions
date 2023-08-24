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

type Operation struct {
	*openapi3.Operation
}

type OpenAPIParameter struct {
	*openapi3.Parameter
}

type OpenAPIParameters []*OpenAPIParameter

type OpenAPIResponse struct {
	*openapi3.Response
}

type RequestBody struct {
	*openapi3.RequestBody
}

type (
	Schema         = openapi3.Schema
	OpenAPIContent = openapi3.Content
	SchemaRef      = openapi3.SchemaRef
	SchemaRefs     = openapi3.SchemaRefs
	OpenAPIHeader  = openapi3.Header
	OpenAPIHeaders = openapi3.Headers
	MediaType      = openapi3.MediaType
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

func NewOpenAPIParameter(name, in string, schema *Schema) *OpenAPIParameter {
	return &OpenAPIParameter{
		Parameter: &openapi3.Parameter{
			Name:     name,
			In:       in,
			Schema:   &SchemaRef{Value: schema},
			Required: true,
		},
	}
}

func NewRequestBodyFromContent(content map[string]*MediaType) *RequestBody {
	return &RequestBody{&openapi3.RequestBody{
		Content: content,
	}}
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

func (d *Document) FindOperation(resourceName, method string) *Operation {
	path := d.Paths.Find(resourceName)
	if path == nil {
		return nil
	}
	op := path.GetOperation(method)
	if op == nil {
		return nil
	}
	return &Operation{op}
}

func (o *Operation) GetRequestBody() *RequestBody {
	if o.RequestBody == nil {
		return nil
	}

	return &RequestBody{o.RequestBody.Value}
}

func (o *Operation) GetResponse() (*OpenAPIResponse, int) {
	available := o.Responses

	var responseRef *openapi3.ResponseRef
	for _, code := range []int{http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent} {
		responseRef = available.Get(code)
		if responseRef != nil {
			return &OpenAPIResponse{responseRef.Value}, code
		}
	}

	// Get first defined
	for codeName, respRef := range available {
		if codeName == "default" {
			continue
		}
		return &OpenAPIResponse{respRef.Value}, TransformHTTPCode(codeName)
	}

	return &OpenAPIResponse{available.Default().Value}, 200
}

func (o *Operation) GetParameters() OpenAPIParameters {
	var res []*OpenAPIParameter
	for _, param := range o.Parameters {
		res = append(res, &OpenAPIParameter{param.Value})
	}
	return res
}

func ValidateRequest(req *http.Request, body *RequestBody) error {
	inp := &openapi3filter.RequestValidationInput{Request: req}
	return openapi3filter.ValidateRequestBody(context.Background(), inp, body.RequestBody)
}

func ValidateResponse(req *http.Request, res *Response, operation *Operation) error {
	inp := &openapi3filter.RequestValidationInput{
		Request: req,
		Route: &routers.Route{
			Method:    req.Method,
			Operation: operation.Operation,
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
