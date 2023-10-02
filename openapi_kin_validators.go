package connexions

import (
	"context"
	"encoding/json"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"net/http"
	"net/url"
)

type kinOpenAPIValidator struct {
	supportedRequestContentTypes  map[string]bool
	supportedResponseContentTypes map[string]bool
}

// NewKinOpenAPIValidator creates a new validator from kin-openapi document.
func NewKinOpenAPIValidator(_ Document) OpenAPIValidator {
	return &kinOpenAPIValidator{
		supportedRequestContentTypes: map[string]bool{
			"application/json":                  true,
			"application/x-www-form-urlencoded": true,
			"multipart/form-data":               true,
		},
		supportedResponseContentTypes: map[string]bool{
			"application/json":                  true,
			"application/x-www-form-urlencoded": true,
			"multipart/form-data":               true,
		},
	}
}

// ValidateRequest validates request against a schema.
func (v *kinOpenAPIValidator) ValidateRequest(req *Request) []error {
	// our request might contain service name in the path,
	// so we need to replace it.
	newReq := new(http.Request)
	*newReq = *req.request
	newReq.URL = newReq.URL.ResolveReference(&url.URL{Path: req.Path})

	inp := &openapi3filter.RequestValidationInput{Request: newReq}
	operation := req.operation

	bodySchema, contentType := operation.GetRequestBody()
	if _, supported := v.supportedRequestContentTypes[req.ContentType]; !supported {
		return nil
	}

	// convert to openapi3.Schema
	schema := openapi3.NewSchema()
	if bodySchema != nil {
		current, _ := json.Marshal(bodySchema)
		_ = schema.UnmarshalJSON(current)
	}

	reqBody := openapi3.NewRequestBody().WithSchema(
		schema,
		[]string{contentType},
	)

	err := openapi3filter.ValidateRequestBody(context.Background(), inp, reqBody)
	if err != nil {
		return []error{err}
	}
	return nil
}

// ValidateResponse validates a response against an operation.
// Response must contain non-empty headers or it'll fail validation.
func (v *kinOpenAPIValidator) ValidateResponse(res *Response) []error {
	operation := res.operation
	if operation == nil {
		return nil
	}

	kin, isKinOpenAPI := operation.(*KinOperation)
	if !isKinOpenAPI || len(res.Headers) == 0 {
		return nil
	}

	// fast track for no response
	resSchema := operation.GetResponse()
	if (resSchema == nil || resSchema.Content == nil) && res.Content == nil {
		return nil
	}

	// TODO(cubahno): add support for other content types
	// we don't generate binary files for example, now
	// form types should work but that's to be added in libopenapi validator
	if _, supported := v.supportedResponseContentTypes[res.ContentType]; !supported {
		return nil
	}

	inp := &openapi3filter.RequestValidationInput{
		Request: res.request,
		Route: &routers.Route{
			Method:    res.request.Method,
			Operation: kin.Operation,
		},
	}

	responseValidationInput := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: inp,
		Status:                 res.StatusCode,
		Header:                 res.Headers,
	}
	responseValidationInput.SetBodyBytes(res.Content)

	err := openapi3filter.ValidateResponse(context.Background(), responseValidationInput)
	if err != nil {
		return []error{err}
	}

	return nil
}
