package xs

import (
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"net/http"
	"net/url"
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

func NewRequest(pathPrefix, path, method string, operation *openapi3.Operation, valueMaker ValueResolver) *Request {
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

func NewResponse(operation *openapi3.Operation, valueMaker ValueResolver) *Response {
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

func GetContentType(content openapi3.Content) (string, *openapi3.Schema) {
	prioTypes := []string{"application/json", "text/plain", "text/html"}
	for _, contentType := range prioTypes {
		if _, ok := content[contentType]; ok {
			return contentType, content[contentType].Schema.Value
		}
	}

	for contentType, mediaType := range content {
		return contentType, mediaType.Schema.Value
	}

	return "", nil
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

func GenerateURL(path string, valueMaker ValueResolver, params openapi3.Parameters) string {
	for _, paramRef := range params {
		param := paramRef.Value
		if param == nil || param.In != openapi3.ParameterInPath {
			continue
		}

		name := param.Name
		state := &ResolveState{NamePath: []string{name}}
		replaced := valueMaker(param.Schema.Value, state)
		path = strings.Replace(path, "{"+name+"}", fmt.Sprintf("%v", replaced), -1)
	}

	return path
}

func GenerateQuery(valueMaker ValueResolver, params openapi3.Parameters) string {
	queryValues := url.Values{}

	// avoid encoding [] in the query
	encode := func(queryValues url.Values) string {
		var params []string
		for key, values := range queryValues {
			for _, value := range values {
				param := fmt.Sprintf("%s=%s", key, url.QueryEscape(value))
				params = append(params, param)
			}
		}
		return strings.Join(params, "&")
	}

	for _, paramRef := range params {
		param := paramRef.Value
		if param == nil || param.In != openapi3.ParameterInQuery {
			continue
		}

		name := param.Name
		state := &ResolveState{NamePath: []string{name}}
		replaced := GenerateContent(param.Schema.Value, valueMaker, state)
		if replaced == nil {
			replaced = ""
		}

		if slice, ok := replaced.([]interface{}); ok {
			for _, item := range slice {
				queryValues.Add(fmt.Sprintf("%s[]", name), fmt.Sprintf("%v", item))
			}
		} else {
			queryValues.Add(name, fmt.Sprintf("%v", replaced))
		}
	}
	return encode(queryValues)
}

func GenerateContent(schema *openapi3.Schema, valueMaker ValueResolver, state *ResolveState) any {
	if state == nil {
		state = &ResolveState{}
	}
	// fast track with value and correctly resolved type
	if len(state.NamePath) > 0 {
		if res := valueMaker(schema, state); res != nil && IsCorrectlyResolvedType(res, schema.Type) {
			return res
		}
	}

	if schema.Type == openapi3.TypeObject {
		return generateContentObject(schema, valueMaker, state)
	}

	if schema.Type == openapi3.TypeArray {
		return generateContentArray(schema, valueMaker, state)
	}

	for _, s := range schema.AllOf {
		return GenerateContent(s.Value, valueMaker, state)
	}

	if len(schema.AnyOf) > 0 {
		return GenerateContent(schema.AnyOf[0].Value, valueMaker, state)
	}

	if len(schema.OneOf) > 0 {
		return GenerateContent(schema.OneOf[0].Value, valueMaker, state)
	}

	// handle Not case

	// try to resolve anything
	return valueMaker(schema, state)
}

func GenerateRequestBody(bodyRef *openapi3.RequestBodyRef, valueMaker ValueResolver, state *ResolveState) (any, string) {
	if state == nil {
		state = &ResolveState{}
	}

	if bodyRef == nil {
		return nil, ""
	}
	contentTypes := bodyRef.Value.Content
	if len(contentTypes) == 0 {
		return nil, ""
	}

	typesOrder := []string{"application/json", "multipart/form-data", "application/x-www-form-urlencoded"}
	for _, contentType := range typesOrder {
		if _, ok := contentTypes[contentType]; ok {
			// TODO(igor): handle correctly content types
			return GenerateContent(
					contentTypes[contentType].Schema.Value, valueMaker, state.setContentType(contentType)),
				contentType
		}
	}

	for contentType, mediaType := range contentTypes {
		return GenerateContent(mediaType.Schema.Value, valueMaker, state.setContentType(contentType)), contentType
	}

	return nil, ""
}

func GenerateRequestHeaders(parameters openapi3.Parameters, valueMaker ValueResolver) any {
	res := map[string]interface{}{}

	for _, paramRef := range parameters {
		param := paramRef.Value
		if param == nil {
			continue
		}

		in := strings.ToLower(param.In)
		if in != openapi3.ParameterInHeader {
			continue
		}

		schemaRef := param.Schema
		if schemaRef == nil {
			continue
		}

		schema := schemaRef.Value
		if schema == nil {
			continue
		}

		for paramName, schemaRef := range schema.Properties {
			state := &ResolveState{NamePath: []string{paramName}, IsHeader: true}
			res[paramName] = GenerateContent(schemaRef.Value, valueMaker, state)
		}
	}

	return res
}

func GenerateResponseHeaders(headers openapi3.Headers, valueMaker ValueResolver) map[string]any {
	res := map[string]any{}

	for name, headerRef := range headers {
		state := &ResolveState{NamePath: []string{name}, IsHeader: true}
		header := headerRef.Value
		params := header.Parameter
		res[name] = GenerateContent(params.Schema.Value, valueMaker, state)
	}
	return res
}

func generateContentObject(schema *openapi3.Schema, valueMaker ValueResolver, state *ResolveState) any {
	if state == nil {
		state = &ResolveState{}
	}
	res := map[string]interface{}{}

	if schema.Properties == nil {
		return res
	}

	for name, prop := range schema.Properties {
		res[name] = GenerateContent(prop.Value, valueMaker, state.addPath(name))
	}

	return res
}

func generateContentArray(schema *openapi3.Schema, valueMaker ValueResolver, state *ResolveState) any {
	if state == nil {
		state = &ResolveState{}
	}
	minItems := int(schema.MinItems)
	if minItems == 0 {
		minItems = 1
	}
	var res []any

	for i := 0; i < minItems+1; i++ {
		res = append(res, GenerateContent(schema.Items.Value, valueMaker, state))
	}

	return res
}
