package xs

import (
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Request struct {
	Headers     interface{}     `json:"headers,omitempty"`
	Method      string          `json:"method,omitempty"`
	Path        string          `json:"path,omitempty"`
	Query       interface{}     `json:"query,omitempty"`
	Body        string          `json:"body,omitempty"`
	ContentType string          `json:"contentType,omitempty"`
	Examples    *ContentExample `json:"examples,omitempty"`
}

type ContentExample struct {
	CURL string `json:"curl,omitempty"`
}

type Response struct {
	Headers     http.Header `json:"headers,omitempty"`
	Content     interface{} `json:"content,omitempty"`
	ContentType string      `json:"contentType,omitempty"`
	StatusCode  int         `json:"statusCode,omitempty"`
	IsBase64    bool        `json:"isBase64,omitempty"`
}

// NewRequestFromOperation creates a new request from an operation.
// It used to pre-generate payloads from the UI or provide service to generate such.
// It's not part of OpenAPI endpoint handler.
func NewRequestFromOperation(pathPrefix, path, method string, operation *Operation, valueReplacer ValueReplacer) *Request {
	content, contentType := GenerateRequestBody(operation.RequestBody, valueReplacer, nil)
	body, err := EncodeContent(content, contentType)
	if err != nil {
		log.Printf("Error encoding request: %v", err.Error())
	}

	curlExample, err := CreateCURLBody(content, contentType)
	if err != nil {
		log.Printf("Error creating cURL example body: %v", err.Error())
	}

	return &Request{
		Headers:     GenerateRequestHeaders(operation.Parameters, valueReplacer),
		Method:      method,
		Path:        pathPrefix + GenerateURLFromSchemaParameters(path, valueReplacer, operation.Parameters),
		Query:       GenerateQuery(valueReplacer, operation.Parameters),
		Body:        body,
		ContentType: contentType,
		Examples: &ContentExample{
			CURL: curlExample,
		},
	}
}

func EncodeContent(content any, contentType string) (string, error) {
	if content == nil {
		return "", nil
	}

	switch content.(type) {
	case string:
		return content.(string), nil
	case []byte:
		return string(content.([]byte)), nil
	}

	switch contentType {
	case "application/x-www-form-urlencoded", "multipart/form-data", "application/json":
		res, err := json.Marshal(content)
		if err != nil {
			return "", ErrUnexpectedFormURLEncodedType
		}
		return string(res), nil

	case "application/xml":
		res, err := xml.Marshal(content)
		if err != nil {
			return "", err
		}
		return string(res), nil
	}

	return "", nil
}

func CreateCURLBody(content any, contentType string) (string, error) {
	if content == nil {
		return "", nil
	}

	switch contentType {
	case "application/x-www-form-urlencoded":
		data, ok := content.(map[string]any)
		if !ok {
			return "", ErrUnexpectedFormURLEncodedType
		}
		builder := &strings.Builder{}
		for key, value := range data {
			builder.WriteString("--data-urlencode ")
			builder.WriteString(fmt.Sprintf(`'%s=%v'`, url.QueryEscape(key), url.QueryEscape(fmt.Sprintf("%v", value))))
			builder.WriteString(" \\\n")
		}
		return strings.TrimSuffix(builder.String(), " \\\n"), nil

	case "multipart/form-data":
		data, ok := content.(map[string]any)
		if !ok {
			return "", ErrUnexpectedFormDataType
		}
		builder := &strings.Builder{}
		for key, value := range data {
			builder.WriteString("--form ")
			builder.WriteString(fmt.Sprintf(`'%s="%v"'`, url.QueryEscape(key), url.QueryEscape(fmt.Sprintf("%v", value))))
			builder.WriteString(" \\\n")
		}
		return strings.TrimSuffix(builder.String(), " \\\n"), nil

	case "application/json":
		enc, err := json.Marshal(content)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("--data-raw '%s'", string(enc)), nil

	case "application/xml":
		enc, err := xml.Marshal(content)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("--data '%s'", string(enc)), nil
	}

	return "", nil
}

func NewRequestFromFileProperties(path, method, contentType string, valueReplacer ValueReplacer) *Request {
	return &Request{
		Method:      method,
		Path:        GenerateURLFromFileProperties(path, valueReplacer),
		ContentType: contentType,
	}
}

func NewResponseFromOperation(operation *Operation, valueReplacer ValueReplacer) *Response {
	response, statusCode := ExtractResponse(operation)

	headers := GenerateResponseHeaders(response.Headers, valueReplacer)

	contentType, contentSchema := GetContentType(response.Content)
	if contentType == "" {
		return &Response{
			StatusCode: statusCode,
			Headers:    headers,
		}
	}

	headers.Set("content-type", contentType)

	return &Response{
		Headers:     headers,
		Content:     GenerateContentFromSchema(contentSchema, valueReplacer, nil),
		ContentType: contentType,
		StatusCode:  statusCode,
	}
}

func NewResponseFromFileProperties(
	filePath, contentType string, valueReplacer ValueReplacer) *Response {
	content, isBase64 := GenerateContentFromFileProperties(filePath, contentType, valueReplacer)
	hs := http.Header{}
	hs.Set("content-type", contentType)

	return &Response{
		Headers:     hs,
		Content:     content,
		ContentType: contentType,
		IsBase64:    isBase64,
		StatusCode:  http.StatusOK,
	}
}

func ExtractResponse(operation *Operation) (*OpenAPIResponse, int) {
	available := operation.Responses

	var responseRef *ResponseRef
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

func GetContentType(content OpenAPIContent) (string, *Schema) {
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

func GenerateURLFromSchemaParameters(path string, valueResolver ValueReplacer, params OpenAPIParameters) string {
	for _, paramRef := range params {
		param := paramRef.Value
		if param == nil || param.In != ParameterInPath {
			continue
		}

		name := param.Name
		state := &ReplaceState{NamePath: []string{name}}
		replaced := valueResolver(param.Schema.Value, state)
		path = strings.Replace(path, "{"+name+"}", fmt.Sprintf("%v", replaced), -1)
	}

	return path
}

func GenerateURLFromFileProperties(path string, valueReplacer ValueReplacer) string {
	placeHolders := ExtractPlaceholders(path)
	if valueReplacer == nil {
		return path
	}

	for _, placeholder := range placeHolders {
		name := placeholder[1 : len(placeholder)-1]
		res := valueReplacer("", (&ReplaceState{}).WithName(name).WithURLParam())
		if res != nil {
			path = strings.Replace(path, placeholder, fmt.Sprintf("%v", res), -1)
		}
	}
	return path
}

func GenerateQuery(valueReplacer ValueReplacer, params OpenAPIParameters) string {
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
		if param == nil || param.In != ParameterInQuery {
			continue
		}

		name := param.Name
		state := &ReplaceState{NamePath: []string{name}}
		replaced := GenerateContentFromSchema(param.Schema.Value, valueReplacer, state)
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

func GenerateContentFromSchema(schema *Schema, valueResolver ValueReplacer, state *ReplaceState) any {
	if state == nil {
		state = &ReplaceState{}
	}
	// fast track with value and correctly resolved type
	if valueResolver != nil && len(state.NamePath) > 0 {
		// TODO(igor): remove IsCorrectlyReplacedType, resolver should do it.
		if res := valueResolver(schema, state); res != nil && IsCorrectlyReplacedType(res, schema.Type) {
			return res
		}
	}

	mergedSchema := MergeSubSchemas(schema)

	if mergedSchema.Type == TypeObject {
		return GenerateContentObject(mergedSchema, valueResolver, state)
	}

	if mergedSchema.Type == TypeArray {
		return GenerateContentArray(mergedSchema, valueResolver, state)
	}

	// try to resolve anything
	if valueResolver != nil {
		return valueResolver(mergedSchema, state)
	}

	return nil
}

func GenerateContentObject(schema *Schema, valueReplacer ValueReplacer, state *ReplaceState) any {
	if state == nil {
		state = &ReplaceState{}
	}

	res := map[string]interface{}{}

	if schema.Properties == nil {
		return nil
	}

	for name, schemaRef := range schema.Properties {
		if state.IsReferenceVisited(schemaRef.Ref) {
			continue
		}
		s := state.NewFrom(state).WithName(name).WithReference(schemaRef.Ref)
		res[name] = GenerateContentFromSchema(schemaRef.Value, valueReplacer, s)
	}

	if len(res) == 0 {
		return nil
	}

	return res
}

func GenerateContentArray(schema *Schema, valueReplacer ValueReplacer, state *ReplaceState) any {
	if state == nil {
		state = &ReplaceState{}
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
		item := GenerateContentFromSchema(schema.Items.Value, valueReplacer, state.NewFrom(state).WithElementIndex(i))
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

func MergeSubSchemas(schema *Schema) *Schema {
	allOf := schema.AllOf
	anyOf := schema.AnyOf
	oneOf := schema.OneOf
	not := schema.Not

	if len(schema.Properties) == 0 {
		schema.Properties = make(map[string]*SchemaRef)
	}

	schema.AllOf = make([]*SchemaRef, 0)
	schema.AnyOf = make([]*SchemaRef, 0)
	schema.OneOf = make([]*SchemaRef, 0)
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
	schemaRefs := []SchemaRefs{anyOf, oneOf}
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

func GenerateRequestBody(bodyRef *RequestBodyRef, valueResolver ValueReplacer, state *ReplaceState) (any, string) {
	if state == nil {
		state = &ReplaceState{}
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
			s := contentTypes[contentType].Schema.Value
			return GenerateContentFromSchema(s, valueResolver, state.WithContentType(contentType)), contentType
		}
	}

	var res any
	var typ string

	// Get first defined
	for contentType, mediaType := range contentTypes {
		typ = contentType
		res = GenerateContentFromSchema(mediaType.Schema.Value, valueResolver, state.WithContentType(contentType))
		break
	}

	return res, typ
}

func GenerateRequestHeaders(parameters OpenAPIParameters, valueReplacer ValueReplacer) any {
	res := map[string]interface{}{}

	for _, paramRef := range parameters {
		param := paramRef.Value
		if param == nil {
			continue
		}

		in := strings.ToLower(param.In)
		if in != ParameterInHeader {
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
		res[name] = GenerateContentFromSchema(
			schema, valueReplacer, &ReplaceState{NamePath: []string{name}, IsHeader: true})
	}

	if len(res) == 0 {
		return nil
	}

	return res
}

func GenerateResponseHeaders(headers OpenAPIHeaders, valueReplacer ValueReplacer) http.Header {
	res := http.Header{}

	for name, headerRef := range headers {
		name = strings.ToLower(name)
		state := &ReplaceState{NamePath: []string{name}, IsHeader: true}
		header := headerRef.Value
		params := header.Parameter
		value := GenerateContentFromSchema(params.Schema.Value, valueReplacer, state)
		res.Set(name, fmt.Sprintf("%v", value))
	}
	return res
}

func GenerateContentFromFileProperties(
	filePath, contentType string, valueReplacer ValueReplacer) (any, bool) {
	if filePath == "" {
		return nil, false
	}

	payload, err := os.ReadFile(filePath)
	if err != nil {
		return nil, false
	}

	if contentType == "application/octet-stream" {
		return base64.StdEncoding.EncodeToString(payload), true
	}

	return GenerateContentFromBytes(payload, contentType, valueReplacer), false
}

func GenerateContentFromBytes(payload []byte, contentType string, valueReplacer ValueReplacer) any {
	if len(payload) == 0 {
		return ""
	}

	if valueReplacer == nil {
		return string(payload)
	}

	switch contentType {
	case "application/json":
		var data any
		err := json.Unmarshal(payload, &data)
		if err != nil {
			return ""
		}
		return GenerateContentFromJSON(data, valueReplacer, nil)
	default:
		return string(payload)
	}
}

func GenerateContentFromJSON(data any, valueReplacer ValueReplacer, state *ReplaceState) any {
	if state == nil {
		state = &ReplaceState{}
	}

	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, val := range v {
			resolved := valueReplacer(val, state.NewFrom(state).WithName(key))
			if resolved != nil {
				result[key] = resolved
			} else {
				result[key] = GenerateContentFromJSON(val, valueReplacer, state.NewFrom(state).WithName(key))
			}
		}
		return result

	case map[interface{}]interface{}:
		result := make(map[interface{}]interface{})
		for key, val := range v {
			resolved := valueReplacer(val, state.NewFrom(state).WithName(key.(string)))
			if resolved != nil {
				result[key] = resolved
			} else {
				result[key] = GenerateContentFromJSON(val, valueReplacer, state.NewFrom(state).WithName(key.(string)))
			}
		}
		return result

	case []interface{}:
		result := make([]interface{}, len(v))
		for i, val := range v {
			resolved := valueReplacer(val, state.NewFrom(state).WithName(fmt.Sprintf("%v", i)))
			if resolved != nil {
				result[i] = resolved
			} else {
				result[i] = GenerateContentFromJSON(val, valueReplacer, state.NewFrom(state).WithName(fmt.Sprintf("%v", i)))
			}
		}
		return result
	default:
		return data
	}
}
