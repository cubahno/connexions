package internal

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// NewRequestFromOperation creates a new GeneratedRequest from an KinOperation.
// It used to pre-generate payloads from the UI or provide service to generate such.
// It's not part of OpenAPI endpoint handler.
func NewRequestFromOperation(
	options *GenerateRequestOptions,
	securityComponents SecurityComponents,
	replacer ValueReplacer) *GeneratedRequest {
	request := options.Operation.GetRequest(securityComponents)
	payload := request.Body
	reqBody := payload.Schema
	contentType := payload.Type

	state := NewReplaceState(
		WithContentType(contentType),
		WithWriteOnly())
	content := GenerateContentFromSchema(reqBody, replacer, state)
	body, err := EncodeContent(content, contentType)
	if err != nil {
		log.Printf("Error encoding GeneratedRequest: %v", err.Error())
	}

	curlExample, err := CreateCURLBody(content, contentType)
	if err != nil {
		log.Printf("Error creating cURL example body: %v", err.Error())
	}

	params := request.Parameters

	return &GeneratedRequest{
		Headers:       GenerateRequestHeaders(params, replacer),
		Method:        options.Method,
		Path:          options.PathPrefix + GenerateURLFromSchemaParameters(options.Path, replacer, params),
		Query:         GenerateQuery(replacer, params),
		Body:          string(body),
		ContentSchema: reqBody,
		ContentType:   contentType,
		Examples: &ContentExample{
			CURL: curlExample,
		},
	}
}

func NewRequestFromFixedResource(path, method, contentType string, valueReplacer ValueReplacer) *GeneratedRequest {
	// TODO: add cURL example
	return &GeneratedRequest{
		Method:      method,
		Path:        generateURLFromFixedResourcePath(path, valueReplacer),
		ContentType: contentType,
	}
}

// NewResponseFromOperation creates a new response from an KinOperation.
// It used to pre-generate payloads from the UI or provide service to generate such.
func NewResponseFromOperation(req *http.Request, operation Operation, valueReplacer ValueReplacer) *GeneratedResponse {
	response := operation.GetResponse()
	statusCode := response.StatusCode

	headers := GenerateResponseHeaders(response.Headers, valueReplacer)

	contentSchema := response.Content
	contentType := response.ContentType
	if contentType == "" {
		contentType = "application/json"
	}

	headers.Set("content-type", contentType)
	state := NewReplaceState(WithContentType(contentType), WithReadOnly())
	content := GenerateContentFromSchema(contentSchema, valueReplacer, state)

	contentB, err := EncodeContent(content, contentType)
	if err != nil {
		log.Printf("Error encoding response: %v", err.Error())
	}

	return &GeneratedResponse{
		Headers:     headers,
		Content:     contentB,
		ContentType: contentType,
		StatusCode:  statusCode,
		Operation:   operation,
		Request:     req,
	}
}

func NewResponseFromFixedResource(filePath, contentType string, valueReplacer ValueReplacer) *GeneratedResponse {
	content := GenerateContentFromFileProperties(filePath, contentType, valueReplacer)
	hs := make(http.Header)
	hs.Set("content-type", contentType)

	return &GeneratedResponse{
		Headers:     hs,
		Content:     content,
		ContentType: contentType,
		// 200 is the only possible status code for fixed resource
		StatusCode: http.StatusOK,
	}
}

// GenerateURLFromSchemaParameters generates URL from the given path and parameters.
func GenerateURLFromSchemaParameters(path string, valueResolver ValueReplacer, params Parameters) string {
	for _, param := range params {
		// param := paramRef.Parameter
		if param == nil || param.In != ParameterInPath {
			continue
		}

		name := param.Name
		schema := param.Schema

		state := NewReplaceState(WithName(name), WithPath())
		replaced := valueResolver(schema, state)
		replaced = fmt.Sprintf("%v", replaced)
		if replaced == "" {
			log.Printf("Warning: parameter '%s' not replaced in URL path", name)
			continue
		}
		path = strings.Replace(path, "{"+name+"}", fmt.Sprintf("%v", replaced), -1)
	}

	return path
}

func generateURLFromFixedResourcePath(path string, valueReplacer ValueReplacer) string {
	placeHolders := ExtractPlaceholders(path)
	if valueReplacer == nil {
		return path
	}

	for _, placeholder := range placeHolders {
		name := placeholder[1 : len(placeholder)-1]

		state := NewReplaceState(WithName(name), WithPath())
		res := valueReplacer("", state)

		if res != nil {
			replaceWith := fmt.Sprintf("%v", res)
			if len(replaceWith) > 0 {
				path = strings.Replace(path, placeholder, replaceWith, -1)
			} else {
				log.Printf("parameter '%s' not replaced in URL path", name)
			}
		}
	}
	return path
}

// GenerateQuery generates query string from the given parameters.
func GenerateQuery(valueReplacer ValueReplacer, params Parameters) string {
	queryValues := url.Values{}

	// avoid encoding [] in the query
	encode := func(queryValues url.Values) string {
		var ps []string
		for key, values := range queryValues {
			for _, value := range values {
				param := fmt.Sprintf("%s=%s", key, url.QueryEscape(value))
				ps = append(ps, param)
			}
		}
		return strings.Join(ps, "&")
	}

	for _, param := range params {
		if param == nil || param.In != ParameterInQuery {
			continue
		}

		name := param.Name
		state := NewReplaceStateWithName(name)
		replaced := GenerateContentFromSchema(param.Schema, valueReplacer, state)
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

// GenerateContentFromSchema generates content from the given schema.
func GenerateContentFromSchema(schema *Schema, valueReplacer ValueReplacer, state *ReplaceState) any {
	if schema == nil {
		return nil
	}

	if state == nil {
		state = NewReplaceState()
	}

	// nothing to replace
	if !IsMatchSchemaReadWriteToState(schema, state) {
		return nil
	}

	// fast track with value and correctly resolved type for primitive types
	if valueReplacer != nil && len(state.NamePath) > 0 && schema.Type != TypeObject && schema.Type != TypeArray {
		// TODO(cubahno): remove IsCorrectlyReplacedType, resolver should do it.
		if res := valueReplacer(schema, state); res != nil && IsCorrectlyReplacedType(res, schema.Type) {
			if res == NULL {
				return nil
			}
			return res
		}
	}

	if schema.Type == TypeObject {
		obj := GenerateContentObject(schema, valueReplacer, state)
		if obj == nil && !schema.Nullable {
			obj = map[string]any{}
		}
		return obj
	}

	if schema.Type == TypeArray {
		arr := GenerateContentArray(schema, valueReplacer, state)
		if arr == nil && !schema.Nullable {
			arr = []any{}
		}
		return arr
	}

	// try to resolve anything
	if valueReplacer != nil {
		res := valueReplacer(schema, state)
		if res == NULL {
			return nil
		}
		return res
	}

	return nil
}

// GenerateContentObject generates content from the given schema with type `object`.
func GenerateContentObject(schema *Schema, valueReplacer ValueReplacer, state *ReplaceState) any {
	if state == nil {
		state = NewReplaceState()
	}

	res := map[string]any{}

	if len(schema.Properties) == 0 {
		return nil
	}

	for name, schemaRef := range schema.Properties {
		s := state.NewFrom(state).WithOptions(WithName(name))
		value := GenerateContentFromSchema(schemaRef, valueReplacer, s)
		// TODO(cubahno): decide whether config value needed to include null values
		if value == nil {
			continue
		}

		res[name] = value

		if schema.MaxProperties > 0 && len(res) >= int(schema.MaxProperties) {
			break
		}
	}

	return res
}

// GenerateContentArray generates content from the given schema with type `array`.
func GenerateContentArray(schema *Schema, valueReplacer ValueReplacer, state *ReplaceState) any {
	if state == nil {
		state = NewReplaceState()
	}

	// avoid generating too many items
	take := int(schema.MinItems)
	if take == 0 {
		take = 1
	}

	var res []any

	for i := 1; i < 10; i++ {
		if i > take {
			break
		}
		item := GenerateContentFromSchema(schema.Items, valueReplacer,
			state.NewFrom(state).WithOptions(WithElementIndex(i)))
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

// GenerateRequestHeaders generates GeneratedRequest headers from the given parameters.
func GenerateRequestHeaders(parameters Parameters, valueReplacer ValueReplacer) map[string]any {
	res := map[string]any{}

	for _, param := range parameters {
		if param == nil {
			continue
		}

		if in := strings.ToLower(param.In); in != ParameterInHeader {
			continue
		}

		schema := param.Schema
		if schema == nil {
			continue
		}

		name := strings.ToLower(param.Name)
		res[name] = GenerateContentFromSchema(
			schema, valueReplacer, NewReplaceState(WithName(name), WithHeader()))
	}

	if len(res) == 0 {
		return nil
	}

	return res
}

// GenerateResponseHeaders generates response headers from the given headers.
func GenerateResponseHeaders(headers Headers, valueReplacer ValueReplacer) http.Header {
	res := http.Header{}

	for name, params := range headers {
		name = strings.ToLower(name)
		state := NewReplaceState(WithName(name), WithHeader())

		value := GenerateContentFromSchema(params.Schema, valueReplacer, state)
		res.Set(name, fmt.Sprintf("%v", value))
	}
	return res
}

func GenerateContentFromFileProperties(filePath, contentType string, valueReplacer ValueReplacer) []byte {
	if filePath == "" {
		log.Println("file path is empty")
		return nil
	}

	payload, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("Error reading file: %v", err.Error())
		return nil
	}

	if contentType == "application/json" {
		var data any
		if err := json.Unmarshal(payload, &data); err != nil {
			log.Printf("Error unmarshalling JSON: %v", err.Error())
			return nil
		}
		generated := generateContentFromJSON(data, valueReplacer, nil)
		bts, _ := json.Marshal(generated)
		return bts
	}

	return payload
}

func generateContentFromJSON(data any, valueReplacer ValueReplacer, state *ReplaceState) any {
	if valueReplacer == nil {
		return data
	}
	if state == nil {
		state = NewReplaceState()
	}

	resolve := func(key string, val any) any {
		vStr, ok := val.(string)
		// if value not a string, just copy it
		if !ok {
			return val
		}

		placeHolders := ExtractPlaceholders(vStr)
		vs := make(map[string]any)

		for _, placeholder := range placeHolders {
			name := placeholder[1 : len(placeholder)-1]
			res := valueReplacer(name, state.NewFrom(state).WithOptions(WithName(name)))
			if res != nil {
				newKey := fmt.Sprintf("%s%s%s", string(placeholder[0]), name, string(placeholder[len(placeholder)-1]))
				vs[newKey] = res
			}
		}

		if len(vs) == 0 {
			return val
		}

		// return as-is without type conversion
		if len(vs) == 1 {
			for _, res := range vs {
				return res
			}
		}

		// multiple replacements glued together in one string
		for placeholder, newValue := range vs {
			vStr = strings.ReplaceAll(vStr, placeholder, fmt.Sprintf("%v", newValue))
		}

		return vStr
	}

	switch v := data.(type) {
	case map[string]any:
		result := make(map[string]interface{})
		for key, val := range v {
			result[key] = val
			res := resolve(key, val)
			if res != nil {
				result[key] = res
			}
		}
		return result

	case map[any]any:
		result := make(map[any]any)
		for key, val := range v {
			result[key] = val
			res := resolve(key.(string), val)
			if res != nil {
				result[key] = res
			}
		}
		return result

	case []any:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = val
			res := resolve(fmt.Sprintf("%v", i), val)
			if res != nil {
				result[i] = res
			}
		}
		return result
	default:
		return data
	}
}
