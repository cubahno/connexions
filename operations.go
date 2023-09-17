package connexions

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"gopkg.in/yaml.v3"
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
	Content     []byte      `json:"content,omitempty"`
	ContentType string      `json:"contentType,omitempty"`
	StatusCode  int         `json:"statusCode,omitempty"`
}

// NewRequestFromOperation creates a new request from an operation.
// It used to pre-generate payloads from the UI or provide service to generate such.
// It's not part of OpenAPI endpoint handler.
func NewRequestFromOperation(pathPrefix, path, method string, operation Operationer, valueReplacer ValueReplacer) *Request {
	reqBody, contentType := operation.GetRequestBody()
	state := &ReplaceState{}
	content := GenerateContentFromSchema(reqBody, valueReplacer, state.WithContentType(contentType))
	body, err := EncodeContent(content, contentType)
	if err != nil {
		log.Printf("Error encoding request: %v", err.Error())
	}

	curlExample, err := createCURLBody(content, contentType)
	if err != nil {
		log.Printf("Error creating cURL example body: %v", err.Error())
	}

	params := operation.GetParameters()

	return &Request{
		Headers:     GenerateRequestHeaders(params, valueReplacer),
		Method:      method,
		Path:        pathPrefix + GenerateURLFromSchemaParameters(path, valueReplacer, params),
		Query:       GenerateQuery(valueReplacer, params),
		Body:        string(body),
		ContentType: contentType,
		Examples: &ContentExample{
			CURL: curlExample,
		},
	}
}

func EncodeContent(content any, contentType string) ([]byte, error) {
	if content == nil {
		return nil, nil
	}

	switch contentType {
	case "application/x-www-form-urlencoded", "multipart/form-data", "application/json":
		return json.Marshal(content)

	case "application/xml":
		return xml.Marshal(content)

	case "application/x-yaml":
		return yaml.Marshal(content)

	default:
		switch v := content.(type) {
		case []byte:
			return v, nil
		case string:
			return []byte(v), nil
		}
	}

	return nil, nil
}

func createCURLBody(content any, contentType string) (string, error) {
	if content == nil {
		return "", nil
	}

	switch contentType {
	case "application/x-www-form-urlencoded":
		data, ok := content.(map[string]any)
		if !ok {
			return "", ErrUnexpectedFormURLEncodedType
		}

		keys := GetSortedMapKeys(data)
		builder := &strings.Builder{}

		for _, key := range keys {
			value := data[key]
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

		keys := GetSortedMapKeys(data)
		builder := &strings.Builder{}

		for _, key := range keys {
			value := data[key]
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

func newRequestFromFixedResource(path, method, contentType string, valueReplacer ValueReplacer) *Request {
	// TODO(cubahno): add cURL example
	return &Request{
		Method:      method,
		Path:        generateURLFromFixedResourcePath(path, valueReplacer),
		ContentType: contentType,
	}
}

func NewResponseFromOperation(operation Operationer, valueReplacer ValueReplacer) *Response {
	response := operation.GetResponse()
	statusCode := response.StatusCode

	headers := GenerateResponseHeaders(response.Headers, valueReplacer)

	contentSchema := response.Content
	contentType := response.ContentType
	if contentType == "" {
		contentType = "application/json"
	}

	headers.Set("content-type", contentType)
	content := GenerateContentFromSchema(contentSchema, valueReplacer, nil)
	contentB, err := EncodeContent(content, contentType)
	if err != nil {
		log.Printf("Error encoding response: %v", err.Error())
	}

	return &Response{
		Headers:     headers,
		Content:     contentB,
		ContentType: contentType,
		StatusCode:  statusCode,
	}
}

func newResponseFromFixedResource(filePath, contentType string, valueReplacer ValueReplacer) *Response {
	content := generateContentFromFileProperties(filePath, contentType, valueReplacer)
	hs := make(http.Header)
	hs.Set("content-type", contentType)

	contentB, err := EncodeContent(content, contentType)
	if err != nil {
		log.Printf("Error encoding response: %v", err.Error())
	}

	return &Response{
		Headers:     hs,
		Content:     contentB,
		ContentType: contentType,
		// 200 is the only possible status code for fixed resource
		StatusCode: http.StatusOK,
	}
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

func GenerateURLFromSchemaParameters(path string, valueResolver ValueReplacer, params OpenAPIParameters) string {
	for _, param := range params {
		// param := paramRef.Parameter
		if param == nil || param.In != ParameterInPath {
			continue
		}

		name := param.Name
		schema := param.Schema

		state := (&ReplaceState{}).WithName(name).WithPathParam()
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
		res := valueReplacer("", (&ReplaceState{}).WithName(name).WithPathParam())
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

func GenerateQuery(valueReplacer ValueReplacer, params OpenAPIParameters) string {
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
		state := &ReplaceState{NamePath: []string{name}}
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

func GenerateContentFromSchema(schema *Schema, valueResolver ValueReplacer, state *ReplaceState) any {
	if schema == nil {
		return nil
	}

	if state == nil {
		state = &ReplaceState{}
	}

	// fast track with value and correctly resolved type
	if valueResolver != nil && len(state.NamePath) > 0 {
		// TODO(cubahno): remove IsCorrectlyReplacedType, resolver should do it.
		if res := valueResolver(schema, state); res != nil && IsCorrectlyReplacedType(res, schema.Type) {
			return res
		}
	}

	if schema.Type == TypeObject {
		return GenerateContentObject(schema, valueResolver, state)
	}

	if schema.Type == TypeArray {
		return GenerateContentArray(schema, valueResolver, state)
	}

	// try to resolve anything
	if valueResolver != nil {
		return valueResolver(schema, state)
	}

	return nil
}

func GenerateContentObject(schema *Schema, valueReplacer ValueReplacer, state *ReplaceState) any {
	if state == nil {
		state = &ReplaceState{}
	}

	res := map[string]any{}

	if len(schema.Properties) == 0 {
		return nil
	}

	for name, schemaRef := range schema.Properties {
		s := state.NewFrom(state).WithName(name)
		res[name] = GenerateContentFromSchema(schemaRef, valueReplacer, s)
	}

	return res
}

func GenerateContentArray(schema *Schema, valueReplacer ValueReplacer, state *ReplaceState) any {
	if state == nil {
		state = &ReplaceState{}
	}

	// avoid generating too many items
	take := int(schema.MinItems)
	if take == 0 {
		take = 1
	}

	var res []any

	for i := 1; i < 10; i++ {
		item := GenerateContentFromSchema(schema.Items, valueReplacer, state.NewFrom(state).WithElementIndex(i))
		if item == nil {
			continue
		}
		res = append(res, item)
		if i >= take {
			break
		}
	}

	if len(res) == 0 {
		return nil
	}

	return res
}

func GenerateRequestHeaders(parameters OpenAPIParameters, valueReplacer ValueReplacer) any {
	res := map[string]interface{}{}

	for _, param := range parameters {
		if param == nil {
			continue
		}

		in := strings.ToLower(param.In)
		if in != ParameterInHeader {
			continue
		}

		schema := param.Schema
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

	for name, params := range headers {
		name = strings.ToLower(name)
		state := &ReplaceState{NamePath: []string{name}, IsHeader: true}

		value := GenerateContentFromSchema(params.Schema, valueReplacer, state)
		res.Set(name, fmt.Sprintf("%v", value))
	}
	return res
}

func generateContentFromFileProperties(filePath, contentType string, valueReplacer ValueReplacer) any {
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
		err := json.Unmarshal(payload, &data)
		if err != nil {
			log.Printf("Error unmarshalling JSON: %v", err.Error())
			return nil
		}
		return generateContentFromJSON(data, valueReplacer, nil)
	}

	return payload
}

func generateContentFromJSON(data any, valueReplacer ValueReplacer, state *ReplaceState) any {
	if valueReplacer == nil {
		return data
	}
	if state == nil {
		state = &ReplaceState{}
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
			res := valueReplacer(name, state.NewFrom(state).WithName(name))
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
