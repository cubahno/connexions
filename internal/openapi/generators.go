package openapi

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/cubahno/connexions/internal/replacer"
	"github.com/cubahno/connexions/internal/types"
)

// NewRequestFromOperation creates a new GeneratedRequest.
// It is used to pre-generate payloads from the UI or provide service to generate such.
// It's not part of OpenAPI endpoint handler.
func NewRequestFromOperation(
	options *GenerateRequestOptions,
	securityComponents SecurityComponents,
	valueReplacer replacer.ValueReplacer) *GeneratedRequest {
	request := options.Operation.GetRequest(securityComponents)
	payload := request.Body
	reqBody := payload.Schema
	contentType := payload.Type

	state := replacer.NewReplaceState(
		replacer.WithContentType(contentType),
		replacer.WithWriteOnly())
	content := GenerateContentFromSchema(reqBody, valueReplacer, state)
	body, err := EncodeContent(content, contentType)
	if err != nil {
		slog.Error("Error encoding GeneratedRequest", "error", err)
	}

	curlExample, err := CreateCURLBody(content, contentType)
	if err != nil {
		slog.Error("Error creating cURL example body", "error", err)
	}

	params := request.Parameters

	return &GeneratedRequest{
		Headers:       GenerateRequestHeaders(params, valueReplacer),
		Method:        options.Method,
		Path:          options.PathPrefix + generateURLFromSchemaParameters(options.Path, valueReplacer, params),
		Query:         generateQuery(valueReplacer, params),
		Body:          string(body),
		ContentSchema: reqBody,
		ContentType:   contentType,
		Examples: &ContentExample{
			CURL: curlExample,
		},
	}
}

func NewRequestFromFixedResource(path, method, contentType string, valueReplacer replacer.ValueReplacer) *GeneratedRequest {
	// TODO: add cURL example
	return &GeneratedRequest{
		Method:      method,
		Path:        generateURLFromFixedResourcePath(path, valueReplacer),
		ContentType: contentType,
	}
}

// NewResponseFromOperation creates generated response.
// It used to pre-generate payloads from the UI or provide service to generate such.
func NewResponseFromOperation(operation Operation, valueReplacer replacer.ValueReplacer, req *http.Request) *GeneratedResponse {
	response := operation.GetResponse()
	statusCode := response.StatusCode

	headers := GenerateResponseHeaders(response.Headers, valueReplacer)

	contentSchema := response.Content
	contentType := response.ContentType
	if contentType == "" {
		contentType = "application/json"
	}

	headers.Set("content-type", contentType)
	state := replacer.NewReplaceState(replacer.WithContentType(contentType), replacer.WithReadOnly())
	content := GenerateContentFromSchema(contentSchema, valueReplacer, state)

	contentB, err := EncodeContent(content, contentType)
	if err != nil {
		slog.Error("Error encoding response", "error", err)
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

func NewResponseFromFixedResource(filePath, contentType string, valueReplacer replacer.ValueReplacer) *GeneratedResponse {
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

// generateURLFromSchemaParameters generates URL from the given path and parameters.
func generateURLFromSchemaParameters(path string, valueResolver replacer.ValueReplacer, params Parameters) string {
	for _, param := range params {
		// param := paramRef.Parameter
		if param == nil || param.In != ParameterInPath {
			continue
		}

		name := param.Name
		schema := param.Schema

		state := replacer.NewReplaceState(replacer.WithName(name), replacer.WithPath())
		replaced := valueResolver(schema, state)
		replaced = fmt.Sprintf("%v", replaced)
		if replaced == "" {
			slog.Warn(fmt.Sprintf("Warning: parameter '%s' not replaced in URL path", name))
			continue
		}
		path = strings.Replace(path, "{"+name+"}", fmt.Sprintf("%v", replaced), -1)
	}

	return path
}

func generateURLFromFixedResourcePath(path string, valueReplacer replacer.ValueReplacer) string {
	placeHolders := types.ExtractPlaceholders(path)
	if valueReplacer == nil {
		return path
	}

	for _, placeholder := range placeHolders {
		name := placeholder[1 : len(placeholder)-1]

		state := replacer.NewReplaceState(replacer.WithName(name), replacer.WithPath())
		res := valueReplacer("", state)

		if res != nil {
			replaceWith := fmt.Sprintf("%v", res)
			if len(replaceWith) > 0 {
				path = strings.Replace(path, placeholder, replaceWith, -1)
			} else {
				slog.Warn(fmt.Sprintf("parameter '%s' not replaced in URL path", name))
			}
		}
	}
	return path
}

// generateQuery generates query string from the given parameters.
func generateQuery(valueReplacer replacer.ValueReplacer, params Parameters) string {
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
		state := replacer.NewReplaceStateWithName(name)
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
func GenerateContentFromSchema(schema *types.Schema, valueReplacer replacer.ValueReplacer, state *replacer.ReplaceState) any {
	if schema == nil {
		return nil
	}

	if state == nil {
		state = replacer.NewReplaceState()
	}

	// nothing to replace
	if !replacer.IsMatchSchemaReadWriteToState(schema, state) {
		return nil
	}

	// fast track with value and correctly resolved type for primitive types
	if valueReplacer != nil && len(state.NamePath) > 0 && schema.Type != types.TypeObject && schema.Type != types.TypeArray {
		// TODO(cubahno): remove IsCorrectlyReplacedType, resolver should do it.
		if res := valueReplacer(schema, state); res != nil && replacer.IsCorrectlyReplacedType(res, schema.Type) {
			if res == replacer.NULL {
				return nil
			}
			return res
		}
	}

	if schema.Type == types.TypeObject {
		obj := GenerateContentObject(schema, valueReplacer, state)
		if obj == nil && !schema.Nullable {
			obj = map[string]any{}
		}
		return obj
	}

	if schema.Type == types.TypeArray {
		arr := GenerateContentArray(schema, valueReplacer, state)
		if arr == nil && !schema.Nullable {
			arr = []any{}
		}
		return arr
	}

	// try to resolve anything
	if valueReplacer != nil {
		res := valueReplacer(schema, state)
		if res == replacer.NULL {
			return nil
		}
		return res
	}

	return nil
}

// GenerateContentObject generates content from the given schema with type `object`.
func GenerateContentObject(schema *types.Schema, valueReplacer replacer.ValueReplacer, state *replacer.ReplaceState) any {
	if state == nil {
		state = replacer.NewReplaceState()
	}

	res := map[string]any{}

	if len(schema.Properties) == 0 {
		return nil
	}

	for name, schemaRef := range schema.Properties {
		s := state.NewFrom(state).WithOptions(replacer.WithName(name))
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
func GenerateContentArray(schema *types.Schema, valueReplacer replacer.ValueReplacer, state *replacer.ReplaceState) any {
	if state == nil {
		state = replacer.NewReplaceState()
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
			state.NewFrom(state).WithOptions(replacer.WithElementIndex(i)))
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
func GenerateRequestHeaders(parameters Parameters, valueReplacer replacer.ValueReplacer) map[string]any {
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
			schema, valueReplacer, replacer.NewReplaceState(replacer.WithName(name), replacer.WithHeader()))
	}

	if len(res) == 0 {
		return nil
	}

	return res
}

// GenerateResponseHeaders generates response headers from the given headers.
func GenerateResponseHeaders(headers Headers, valueReplacer replacer.ValueReplacer) http.Header {
	res := http.Header{}

	for name, params := range headers {
		name = strings.ToLower(name)
		state := replacer.NewReplaceState(replacer.WithName(name), replacer.WithHeader())

		value := GenerateContentFromSchema(params.Schema, valueReplacer, state)
		res.Set(name, fmt.Sprintf("%v", value))
	}
	return res
}

func GenerateContentFromFileProperties(filePath, contentType string, valueReplacer replacer.ValueReplacer) []byte {
	if filePath == "" {
		log.Println("file path is empty")
		return nil
	}

	payload, err := os.ReadFile(filePath)
	if err != nil {
		slog.Error("Error reading file", "error", err)
		return nil
	}

	if contentType == "application/json" {
		var data any
		if err := json.Unmarshal(payload, &data); err != nil {
			slog.Error("Error unmarshalling JSON", "error", err)
			return nil
		}
		generated := generateContentFromJSON(data, valueReplacer, nil)
		bts, _ := json.Marshal(generated)
		return bts
	}

	return payload
}

func generateContentFromJSON(data any, valueReplacer replacer.ValueReplacer, state *replacer.ReplaceState) any {
	if valueReplacer == nil {
		return data
	}
	if state == nil {
		state = replacer.NewReplaceState()
	}

	resolve := func(key string, val any) any {
		vStr, ok := val.(string)
		// if value not a string, just copy it
		if !ok {
			return val
		}

		placeHolders := types.ExtractPlaceholders(vStr)
		vs := make(map[string]any)

		for _, placeholder := range placeHolders {
			name := placeholder[1 : len(placeholder)-1]
			res := valueReplacer(name, state.NewFrom(state).WithOptions(replacer.WithName(name)))
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
