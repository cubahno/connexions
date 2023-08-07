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

func NewRequest(pathPrefix, path, method string, operation *openapi3.Operation, valueResolver ValueResolver) *Request {
	body, contentType := GenerateRequestBody(operation.RequestBody, valueResolver, nil)

	return &Request{
		Headers:     GenerateRequestHeaders(operation.Parameters, valueResolver),
		Method:      method,
		Path:        pathPrefix + GenerateURL(path, valueResolver, operation.Parameters),
		Query:       GenerateQuery(valueResolver, operation.Parameters),
		Body:        body,
		ContentType: contentType,
	}
}

func NewResponse(operation *openapi3.Operation, valueResolver ValueResolver) *Response {
	response, statusCode := ExtractResponse(operation)

	headers := GenerateResponseHeaders(response.Headers, valueResolver)

	contentType, contentSchema := GetContentType(response.Content)
	if contentType == "" {
		return &Response{
			StatusCode: statusCode,
			Headers:    headers,
		}
	}

	headers["content-type"] = contentType

	return &Response{
		Headers:     headers,
		Content:     GenerateContent(contentSchema, valueResolver, nil),
		ContentType: contentType,
		StatusCode:  statusCode,
	}
}

func ExtractResponse(operation *openapi3.Operation) (*openapi3.Response, int) {
	available := operation.Responses

	var responseRef *openapi3.ResponseRef
	for _, code := range []int{http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent} {
		responseRef = available.Get(code)
		if responseRef != nil {
			return responseRef.Value, code
		}
	}

	// Get first defined
	for codeName, respRef := range available {
		if codeName == "default" {
			continue
		}
		return respRef.Value, TransformHTTPCode(codeName)
	}

	return available.Default().Value, 200
}

func TransformHTTPCode(httpCode string) int {
	httpCode = strings.ToLower(httpCode)
	httpCode = strings.Replace(httpCode, "x", "0", -1)

	switch httpCode {
	case "*":
		fallthrough
	case "default":
		fallthrough
	case "000":
		return 200
	}

	codeInt, err := strconv.Atoi(httpCode)
	if err != nil {
		return 0
	}

	return codeInt
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

func GenerateContent(schema *openapi3.Schema, valueResolver ValueResolver, state *ResolveState) any {
	if state == nil {
		state = &ResolveState{}
	}
	// fast track with value and correctly resolved type
	if valueResolver != nil && len(state.NamePath) > 0 {
		if res := valueResolver(schema, state); res != nil && IsCorrectlyResolvedType(res, schema.Type) {
			return res
		}
	}

	mergedSchema := MergeSubSchemas(schema)

	if mergedSchema.Type == openapi3.TypeObject {
		return GenerateContentObject(mergedSchema, valueResolver, state)
	}

	if mergedSchema.Type == openapi3.TypeArray {
		return GenerateContentArray(mergedSchema, valueResolver, state)
	}

	// try to resolve anything
	if valueResolver != nil {
		return valueResolver(mergedSchema, state)
	}

	return nil
}

func GenerateContentObject(schema *openapi3.Schema, valueMaker ValueResolver, state *ResolveState) any {
	if state == nil {
		state = &ResolveState{}
	}

	res := map[string]interface{}{}

	if schema.Properties == nil {
		return nil
	}

	if state.IsCircularObjectTrip() {
		return nil
	}

	for name, schemaRef := range schema.Properties {
		item := GenerateContent(schemaRef.Value, valueMaker, state.NewFrom(state).WithName(name))
		if item == nil {
			continue
		}
		res[name] = item
	}

	if len(res) == 0 {
		return nil
	}

	return res
}

func GenerateContentArray(schema *openapi3.Schema, valueMaker ValueResolver, state *ResolveState) any {
	if state == nil {
		state = &ResolveState{}
	}

	minItems := int(schema.MinItems)
	if minItems == 0 {
		minItems = 1
	}

	var res []any

	for i := 0; i < minItems+1; i++ {
		if state.IsCircularArrayTrip(i) {
			return nil
		}
		item := GenerateContent(schema.Items.Value, valueMaker, state.NewFrom(state).WithElementIndex(i))
		if item == nil {
			continue
		}
		res = append(res, item)
	}

	if len(res) == 0 {
		return nil
	}

	return res
}

func MergeSubSchemas(schema *openapi3.Schema) *openapi3.Schema {
	allOf := schema.AllOf
	anyOf := schema.AnyOf
	oneOf := schema.OneOf
	not := schema.Not

	if len(schema.Properties) == 0 {
		schema.Properties = make(map[string]*openapi3.SchemaRef)
	}

	schema.AllOf = make([]*openapi3.SchemaRef, 0)
	schema.AnyOf = make([]*openapi3.SchemaRef, 0)
	schema.OneOf = make([]*openapi3.SchemaRef, 0)
	schema.Not = nil

	for _, schemaRef := range allOf {
		sub := schemaRef.Value
		if sub == nil {
			continue
		}

		for propertyName, property := range sub.Properties {
			schema.Properties[propertyName] = property
		}

		schema.AllOf = append(schema.AllOf, sub.AllOf...)
		schema.AnyOf = append(schema.AnyOf, sub.AnyOf...)
		schema.OneOf = append(schema.OneOf, sub.OneOf...)
		schema.Required = append(schema.Required, sub.Required...)
	}

	// pick first from each if present
	schemaRefs := []openapi3.SchemaRefs{anyOf, oneOf}
	for _, schemaRef := range schemaRefs {
		if len(schemaRef) == 0 {
			continue
		}

		sub := schemaRef[0].Value
		if sub == nil {
			continue
		}

		for propertyName, property := range sub.Properties {
			schema.Properties[propertyName] = property
		}

		schema.AllOf = append(schema.AllOf, sub.AllOf...)
		schema.AnyOf = append(schema.AnyOf, sub.AnyOf...)
		schema.OneOf = append(schema.OneOf, sub.OneOf...)
		schema.Required = append(schema.Required, sub.Required...)
	}

	// exclude properties from `not`
	if not != nil && not.Value != nil {
		notSchema := not.Value
		deletes := map[string]bool{}
		for propertyName, _ := range notSchema.Properties {
			delete(schema.Properties, propertyName)
			deletes[propertyName] = true
		}

		// remove from required properties
		for i, propertyName := range schema.Required {
			if _, ok := deletes[propertyName]; ok {
				schema.Required = append(schema.Required[:i], schema.Required[i+1:]...)
			}
		}
	}

	if len(schema.AllOf) > 0 || len(schema.AnyOf) > 0 || len(schema.OneOf) > 0 {
		return MergeSubSchemas(schema)
	}

	if schema.Type == "" {
		schema.Type = "object"
	}

	return schema
}

func GenerateRequestBody(bodyRef *openapi3.RequestBodyRef, valueResolver ValueResolver, state *ResolveState) (any, string) {
	if state == nil {
		state = &ResolveState{}
	}

	if bodyRef == nil {
		return nil, ""
	}

	schema := bodyRef.Value
	if schema == nil {
		return nil, ""
	}

	contentTypes := schema.Content
	if len(contentTypes) == 0 {
		return nil, ""
	}

	typesOrder := []string{"application/json", "multipart/form-data", "application/x-www-form-urlencoded"}
	for _, contentType := range typesOrder {
		if _, ok := contentTypes[contentType]; ok {
			// TODO(igor): handle correctly content types
			return GenerateContent(
					contentTypes[contentType].Schema.Value, valueResolver, state.WithContentType(contentType)),
				contentType
		}
	}

	var res any
	var typ string

	// Get first defined
	for contentType, mediaType := range contentTypes {
		typ = contentType
		res = GenerateContent(mediaType.Schema.Value, valueResolver, state.WithContentType(contentType))
		break
	}

	return res, typ
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

		name := strings.ToLower(param.Name)
		res[name] = GenerateContent(schema, valueMaker, &ResolveState{NamePath: []string{name}, IsHeader: true})
	}

	if len(res) == 0 {
		return nil
	}

	return res
}

func GenerateResponseHeaders(headers openapi3.Headers, valueMaker ValueResolver) map[string]any {
	res := map[string]any{}

	for name, headerRef := range headers {
		name = strings.ToLower(name)
		state := &ResolveState{NamePath: []string{name}, IsHeader: true}
		header := headerRef.Value
		params := header.Parameter
		res[name] = GenerateContent(params.Schema.Value, valueMaker, state)
	}
	return res
}
