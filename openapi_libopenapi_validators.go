package connexions

import (
	"bytes"
	validator "github.com/pb33f/libopenapi-validator"
	"github.com/pb33f/libopenapi-validator/errors"
	"io"
	"net/http"
)

type libOpenAPIValidator struct {
	innerValidator validator.Validator
}

// NewLibOpenAPIValidator creates a new OpenAPIValidator using the libopenapi-validator library
func NewLibOpenAPIValidator(doc Document) OpenAPIValidator {
	d, ok := doc.(*LibV3Document)
	if !ok {
		return nil
	}

	v := validator.NewValidatorFromV3Model(&d.Model)

	return &libOpenAPIValidator{
		innerValidator: v,
	}
}

// ValidateRequest validates a request against the OpenAPI document.
// Implements OpenAPIValidator interface.
func (v *libOpenAPIValidator) ValidateRequest(req *Request) []error {
	ok, valErrs := v.innerValidator.ValidateHttpRequest(req.request)
	if ok {
		return nil
	}

	return v.getErrors(valErrs)
}

// ValidateResponse validates a response against the OpenAPI document.
// Implements OpenAPIValidator interface.
func (v *libOpenAPIValidator) ValidateResponse(res *Response) []error {
	readCloser := io.NopCloser(bytes.NewReader(res.Content))

	httpResponse := &http.Response{
		StatusCode:    res.StatusCode,
		Header:        res.Headers,
		Body:          readCloser,
		Request:       res.request,
		ContentLength: -1,
	}
	ok, valErrs := v.innerValidator.ValidateHttpResponse(res.request, httpResponse)
	if ok {
		return nil
	}

	return v.getErrors(valErrs)
}

// getErrors converts the errors.ValidationError to []error
func (v *libOpenAPIValidator) getErrors(src []*errors.ValidationError) []error {
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
