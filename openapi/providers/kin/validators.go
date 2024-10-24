package kin

import (
	"context"
	"encoding/json"
	"github.com/cubahno/connexions/openapi"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"net/http"
	"net/url"
)

type Validator struct {
	supportedRequestContentTypes  map[string]bool
	supportedResponseContentTypes map[string]bool
}

// NewValidator creates a new Validator from kin-openapi document.
func NewValidator(_ openapi.Document) openapi.Validator {
	return &Validator{
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
func (v *Validator) ValidateRequest(req *openapi.GeneratedRequest) []error {
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

// ValidateResponse validates a response against an Operation.
// GeneratedResponse must contain non-empty headers or it'll fail validation.
func (v *Validator) ValidateResponse(res *openapi.GeneratedResponse) []error {
	operation := res.Operation
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
	// form types should work but that's to be added in libopenapi Validator
	if _, supported := v.supportedResponseContentTypes[res.ContentType]; !supported {
		return nil
	}

	inp := &openapi3filter.RequestValidationInput{
		Request: res.Request,
		Route: &routers.Route{
			Method:    res.Request.Method,
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
