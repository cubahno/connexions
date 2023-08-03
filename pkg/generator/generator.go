package generator

import (
	"encoding/json"
	"strings"

	"github.com/cubahno/connexions/v2/internal/replacer"
	"github.com/cubahno/connexions/v2/internal/types"
	"github.com/cubahno/connexions/v2/pkg/api"
	"github.com/cubahno/connexions/v2/pkg/schema"
)

type Generate interface {
	Request(req *api.GenerateRequest, op *schema.Operation) json.RawMessage
	Response(respSchema *schema.ResponseSchema) schema.ResponseData
	Error(errSchema *schema.Schema, errPath, error string) []byte
}

type ResponseGenerator struct {
	serviceContexts []map[string]any
	valueReplacer   replacer.ValueReplacer
}

func (g *ResponseGenerator) Request(req *api.GenerateRequest, op *schema.Operation) json.RawMessage {
	valueReplacer := g.valueReplacer

	// static resources.
	if op == nil {
		props := map[string]*schema.Schema{}

		// extract path params from the path
		for _, param := range types.ExtractPlaceholders(req.Path) {
			param = strings.Trim(param, "{}")
			props[param] = &schema.Schema{Type: "any"}
		}

		staticOp := &schema.Operation{
			Path:   req.Path,
			Method: req.Method,
			PathParams: &schema.Schema{
				Type:       "object",
				Properties: props,
			},
		}
		path := generatePath(staticOp, valueReplacer)
		res := map[string]any{
			"path": path,
		}
		jsonBytes, _ := json.Marshal(res)
		return jsonBytes
	}

	// user could have provided custom context.
	// this should be prepended and we need new replacer instance.
	if len(req.Context) > 0 {
		orderedCtx := append([]map[string]any{req.Context}, g.serviceContexts...)
		valueReplacer = replacer.CreateValueReplacer(replacer.Replacers, orderedCtx)
	}

	res := map[string]any{
		"path":        generatePath(op, valueReplacer),
		"contentType": op.ContentType,
	}

	if op.Headers != nil {
		state := replacer.NewReplaceState(replacer.WithWriteOnly())
		headers := generateContentFromSchema(op.Headers, valueReplacer, state)
		if headers != nil {
			res["headers"] = headers
		}
	}

	if op.Body != nil {
		state := replacer.NewReplaceState(replacer.WithWriteOnly())
		body := generateContentFromSchema(op.Body, valueReplacer, state)
		if body != nil {
			// For form-encoded content, encode the body as a form string
			if strings.Contains(strings.ToLower(op.ContentType), "application/x-www-form-urlencoded") {
				formStr, err := types.EncodeFormData(body, op.BodyEncoding)
				if err == nil {
					res["body"] = formStr
				} else {
					// Fallback to the original body if encoding fails
					res["body"] = body
				}
			} else {
				res["body"] = body
			}
		}
	}

	jsonBytes, err := json.Marshal(res)
	if err == nil {
		return jsonBytes
	}

	return nil
}

func (g *ResponseGenerator) Response(respSchema *schema.ResponseSchema) schema.ResponseData {
	// no response respSchema, nothing to generate
	if respSchema == nil {
		return schema.ResponseData{}
	}

	state := replacer.NewReplaceState(
		replacer.WithContentType(respSchema.ContentType),
		replacer.WithReadOnly())

	content := generateContentFromSchema(respSchema.Body, g.valueReplacer, state)
	headers := generateHeaders(respSchema.Headers, g.valueReplacer)

	isError := false
	enc, err := encodeContent(content, respSchema.ContentType)
	if err != nil {
		enc = []byte(err.Error())
		isError = true
	}

	return schema.ResponseData{
		Body:    enc,
		Headers: headers,
		IsError: isError,
	}
}

func (g *ResponseGenerator) Error(errSchema *schema.Schema, errPath, error string) []byte {
	// Default case: no schema or path, return error string directly
	if errSchema == nil || errPath == "" {
		return []byte(error)
	}

	// Response the base structure from the schema
	state := replacer.NewReplaceState()
	content := generateContentFromSchema(errSchema, g.valueReplacer, state)

	// If no content was generated, create an empty object
	var result map[string]any
	if content == nil {
		result = make(map[string]any)
	} else {
		// Try to convert content to map
		var ok bool
		result, ok = content.(map[string]any)
		if !ok {
			// If content is not a map, return error string
			return []byte(error)
		}
	}

	// Inject the error message at the specified path
	types.SetValueByDottedPath(result, errPath, error)

	// Encode the result as JSON (assuming JSON content type for errors)
	encoded, err := encodeContent(result, "application/json")
	if err != nil {
		return []byte(error)
	}

	return encoded
}

func NewGenerator(orderedCtx []map[string]any) (*ResponseGenerator, error) {
	valueReplacer := replacer.CreateValueReplacer(replacer.Replacers, orderedCtx)

	return &ResponseGenerator{
		serviceContexts: orderedCtx,
		valueReplacer:   valueReplacer,
	}, nil
}
