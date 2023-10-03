package lib

import (
	"bytes"
	"github.com/cubahno/connexions/openapi"
	validator "github.com/pb33f/libopenapi-validator"
	"github.com/pb33f/libopenapi-validator/errors"
	"io"
	"net/http"
)

type Validator struct {
	innerValidator validator.Validator
}

// NewValidator creates a new Validator using the libopenapi-validator library
func NewValidator(doc openapi.Document) openapi.Validator {
	d, ok := doc.(*V3Document)
	if !ok {
		return nil
	}

	v := validator.NewValidatorFromV3Model(&d.Model)

	return &Validator{
		innerValidator: v,
	}
}

// ValidateRequest validates a GeneratedRequest against the OpenAPI document.
// Implements Validator interface.
func (v *Validator) ValidateRequest(req *openapi.GeneratedRequest) []error {
	ok, valErrs := v.innerValidator.ValidateHttpRequest(req.Request)
	if ok {
		return nil
	}

	return v.getErrors(valErrs)
}

// ValidateResponse validates a response against the OpenAPI document.
// Implements Validator interface.
func (v *Validator) ValidateResponse(res *openapi.GeneratedResponse) []error {
	readCloser := io.NopCloser(bytes.NewReader(res.Content))

	httpResponse := &http.Response{
		StatusCode:    res.StatusCode,
		Header:        res.Headers,
		Body:          readCloser,
		Request:       res.Request,
		ContentLength: -1,
	}
	ok, valErrs := v.innerValidator.ValidateHttpResponse(res.Request, httpResponse)
	if ok {
		return nil
	}

	return v.getErrors(valErrs)
}

// getErrors converts the errors.ValidationError to []error
func (v *Validator) getErrors(src []*errors.ValidationError) []error {
	var res []error
	for _, err := range src {
		if len(err.SchemaValidationErrors) > 0 {
			for _, schemaErr := range err.SchemaValidationErrors {
				res = append(res, schemaErr)
			}
		} else {
			res = append(res, err)
		}
	}
	return res
}
