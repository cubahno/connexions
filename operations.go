package xs

import (
	"github.com/getkin/kin-openapi/openapi3"
	"net/http"
	"strconv"
	"strings"
)

type Request struct {
	Headers     interface{} `json:"headers,omitempty"`
	Method      string      `json:"method,omitempty"`
	Path        string      `json:"path,omitempty"`
	Query       interface{} `json:"query,omitempty"`
	Body        interface{} `json:"body,omitempty"`
	ContentType string      `json:"contentType,omitempty"`
}

type Response struct {
	Headers     interface{} `json:"headers,omitempty"`
	Content     interface{} `json:"content,omitempty"`
	ContentType string      `json:"contentType,omitempty"`
	StatusCode  int         `json:"statusCode,omitempty"`
}

func NewRequest(pathPrefix, path, method string, operation *openapi3.Operation, valueMaker ValueMaker) *Request {
	body, contentType := GenerateRequestBody(operation.RequestBody, valueMaker, nil)

	return &Request{
		Headers:     GenerateRequestHeaders(operation.Parameters, valueMaker),
		Method:      method,
		Path:        pathPrefix + GenerateURL(path, valueMaker, operation.Parameters),
		Query:       GenerateQuery(valueMaker, operation.Parameters),
		Body:        body,
		ContentType: contentType,
	}
}

func NewResponse(operation *openapi3.Operation, valueMaker ValueMaker) *Response {
	response, statusCode := extractResponse(operation)

	contentType, contentSchema := GetContentType(response.Content)
	if contentType == "" {
		return &Response{
			StatusCode: statusCode,
		}
	}

	headers := GenerateResponseHeaders(response.Headers, valueMaker)
	headers["Content-Type"] = contentType

	return &Response{
		Headers:     headers,
		Content:     GenerateContent(contentSchema, valueMaker, nil),
		ContentType: contentType,
		StatusCode:  statusCode,
	}
}

func extractResponse(operation *openapi3.Operation) (*openapi3.Response, int) {
	available := operation.Responses

	var responseRef *openapi3.ResponseRef
	var statusCode int
	for _, code := range []int{http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent} {
		responseRef = available.Get(code)
		if responseRef != nil {
			statusCode = code
			break
		}
	}

	// Get first defined
	for codeName, respRef := range available {
		if codeName == "default" {
			continue
		}
		responseRef = respRef
		statusCode = transformHTTPCode(codeName)
		break
	}

	if responseRef == nil {
		responseRef = available.Default()
	}

	return responseRef.Value, statusCode
}

func transformHTTPCode(httpCode string) int {
	httpCode = strings.ToLower(httpCode)

	switch httpCode {
	case "*":
		return 200
	case "3xx":
		return 300
	case "4xx":
		return 400
	case "5xx":
		return 500
	case "xxx":
		return 200
	}

	codeInt, err := strconv.Atoi(httpCode)
	if err != nil {
		return 0
	}

	return codeInt
}
