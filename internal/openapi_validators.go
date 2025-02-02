package internal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
)

type KinValidator struct {
	supportedRequestContentTypes  map[string]bool
	supportedResponseContentTypes map[string]bool
}

// NewValidator creates a new KinValidator from kin-openapi document.
func NewValidator(_ Document) *KinValidator {
	return &KinValidator{
		supportedRequestContentTypes: map[string]bool{
			"application/json":    true,
			"multipart/form-data": true,
		},
		supportedResponseContentTypes: map[string]bool{
			"application/json":                  true,
			"application/x-www-form-urlencoded": true,
			"multipart/form-data":               true,
		},
	}
}

// ValidateRequest validates GeneratedRequest against a schema.
func (v *KinValidator) ValidateRequest(req *GeneratedRequest) []error {
	// our GeneratedRequest might contain service name in the path,
	// so we need to replace it.
	newReq := new(http.Request)
	*newReq = *req.Request
	newReq.URL = newReq.URL.ResolveReference(&url.URL{Path: req.Path})

	inp := &openapi3filter.RequestValidationInput{Request: newReq}
	bodySchema := req.ContentSchema

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
		[]string{req.ContentType},
	)

	err := openapi3filter.ValidateRequestBody(context.Background(), inp, reqBody)
	if err != nil {
		return []error{err}
	}
	return nil
}

// ValidateResponse validates a response against an KinOperation.
// GeneratedResponse must contain non-empty headers or it'll fail validation.
func (v *KinValidator) ValidateResponse(res *GeneratedResponse) []error {
	if res.Operation == nil {
		return nil
	}
	operation := res.Operation.Unwrap()
	if operation == nil {
		return nil
	}

	if len(res.Headers) == 0 {
		return nil
	}

	var op *openapi3.Operation
	switch o := operation.(type) {
	case *KinOperation:
		op = o.Operation
	default:
		return nil
	}

	// fast track for no response
	resSchema := operation.GetResponse()
	if (resSchema == nil || resSchema.Content == nil) && res.Content == nil {
		return nil
	}

	// TODO: add support for other content types
	// we don't generate binary files for example, now
	// form types should work but that's to be added in libopenapi KinValidator
	if _, supported := v.supportedResponseContentTypes[res.ContentType]; !supported {
		return nil
	}

	inp := &openapi3filter.RequestValidationInput{
		Request: res.Request,
		Route: &routers.Route{
			Method:    res.Request.Method,
			Operation: op,
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
