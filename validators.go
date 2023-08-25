package connexions

import (
	"context"
	"encoding/json"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"net/http"
	"regexp"
	"strings"
)

var PlaceholderRegex = regexp.MustCompile(`\{[^\}]*\}`)

func IsValidHTTPVerb(verb string) bool {
	validVerbs := map[string]bool{
		http.MethodGet:     true,
		http.MethodHead:    true,
		http.MethodPost:    true,
		http.MethodPut:     true,
		http.MethodPatch:   true,
		http.MethodDelete:  true,
		http.MethodConnect: true,
		http.MethodOptions: true,
		http.MethodTrace:   true,
	}

	// Convert the input verb to uppercase for case-insensitive comparison
	verb = strings.ToUpper(verb)

	return validVerbs[verb]
}

// IsValidURLResource checks if the given URL resource pattern is valid:
// placeholders contain only alphanumeric characters, underscores, and hyphens
func IsValidURLResource(urlPattern string) bool {
	// Find all pairs of curly brackets in the URL pattern
	matches := ExtractPlaceholders(urlPattern)

	// Validate each pair of curly brackets
	for _, match := range matches {
		// Extract content within curly brackets
		content := match[1 : len(match)-1]
		if content == "" {
			return false
		}

		// Regular expression to match invalid characters within curly brackets
		invalidContentPattern := `[^a-zA-Z0-9_\-/]`
		contentRe := regexp.MustCompile(invalidContentPattern)

		if contentRe.MatchString(content) {
			return false
		}
	}

	return true
}

func ExtractPlaceholders(urlPattern string) []string {
	return PlaceholderRegex.FindAllString(urlPattern, -1)
}

func ValidateRequest(req *http.Request, body *Schema, contentType string) error {
	inp := &openapi3filter.RequestValidationInput{Request: req}
	schema := openapi3.NewSchema()
	current, _ := json.Marshal(body)
	_ = schema.UnmarshalJSON(current)

	reqBody := openapi3.NewRequestBody().WithSchema(
		schema,
		[]string{contentType},
	)
	return openapi3filter.ValidateRequestBody(context.Background(), inp, reqBody)
}

// ValidateResponse validates a response against an operation.
func ValidateResponse(req *http.Request, res *Response, operation Operationer) error {
	kin, isKinOpenAPI := operation.(*KinOperation)
	// if not kin openapi, skip validation for now
	if !isKinOpenAPI {
		return nil
	}

	inp := &openapi3filter.RequestValidationInput{
		Request: req,
		Route: &routers.Route{
			Method:    req.Method,
			Operation: kin.Operation,
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
