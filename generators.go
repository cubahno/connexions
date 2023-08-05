package xs

import (
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"net/url"
	"strings"
)

type ValueMaker func(schema *openapi3.Schema, state *GeneratorState) any

type GeneratorState struct {
	NamePath    []string
	Example     any
	IsHeader    bool
	ContentType string
}

func (s *GeneratorState) addPath(name string) *GeneratorState {
	namePath := s.NamePath
	if len(namePath) == 0 {
		namePath = []string{}
	}

	return &GeneratorState{
		NamePath:    append(namePath, name),
		Example:     s.Example,
		IsHeader:    s.IsHeader,
		ContentType: s.ContentType,
	}
}

func (s *GeneratorState) markAsHeader() *GeneratorState {
	return &GeneratorState{
		NamePath:    s.NamePath,
		Example:     s.Example,
		IsHeader:    true,
		ContentType: s.ContentType,
	}
}

func (s *GeneratorState) setContentType(value string) *GeneratorState {
	return &GeneratorState{
		NamePath:    s.NamePath,
		Example:     s.Example,
		IsHeader:    s.IsHeader,
		ContentType: value,
	}
}

func GenerateURL(path string, valueMaker ValueMaker, params openapi3.Parameters) string {
	for _, paramRef := range params {
		param := paramRef.Value
		if param == nil || param.In != "path" {
			continue
		}

		name := param.Name
		state := &GeneratorState{NamePath: []string{name}}
		replaced := valueMaker(param.Schema.Value, state)
		path = strings.Replace(path, "{"+name+"}", fmt.Sprintf("%v", replaced), -1)
	}

	return path
}

func GenerateQuery(valueMaker ValueMaker, params openapi3.Parameters) string {
	queryValues := url.Values{}

	for _, paramRef := range params {
		param := paramRef.Value
		if param == nil || param.In != "query" {
			continue
		}

		name := param.Name
		state := &GeneratorState{NamePath: []string{name}}
		replaced := valueMaker(param.Schema.Value, state)
		queryValues.Add(name, fmt.Sprintf("%v", replaced))
	}
	return queryValues.Encode()
}

func GenerateResponseHeaders(headers openapi3.Headers, valueMaker ValueMaker, state *GeneratorState) any {
	if state == nil {
		state = &GeneratorState{}
	}

	res := map[string]interface{}{}

	for name, headerRef := range headers {
		header := headerRef.Value
		params := header.Parameter
		res[name] = GenerateContent(params.Schema.Value, valueMaker, state.addPath(name).markAsHeader())
	}
	return res
}

func GenerateContent(schema *openapi3.Schema, valueMaker ValueMaker, state *GeneratorState) any {
	if state == nil {
		state = &GeneratorState{}
	}
	// fast track with value and correctly resolved type
	if len(state.NamePath) > 0 {
		if res := valueMaker(schema, state); res != nil { // add correct resolved type check
			return res
		}
	}

	if schema.Type == "object" {
		return generateContentObject(schema, valueMaker, state)
	}

	if schema.Type == "array" {
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

func GenerateRequestBody(bodyRef *openapi3.RequestBodyRef, valueMaker ValueMaker, state *GeneratorState) (any, string) {
	if state == nil {
		state = &GeneratorState{}
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

func generateContentObject(schema *openapi3.Schema, valueMaker ValueMaker, state *GeneratorState) any {
	if state == nil {
		state = &GeneratorState{}
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

func generateContentArray(schema *openapi3.Schema, valueMaker ValueMaker, state *GeneratorState) any {
	if state == nil {
		state = &GeneratorState{}
	}
	minItems := int(schema.MinItems)
	if minItems == 0 {
		minItems = 1
	}
	var res []any

	for i := 0; i < minItems; i++ {
		res = append(res, GenerateContent(schema.Items.Value, valueMaker, state))
	}

	return res
}
