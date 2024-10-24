package lib

import (
	"fmt"
	"github.com/cubahno/connexions/config"
	"github.com/cubahno/connexions/openapi"
	"github.com/pb33f/libopenapi"
	v2high "github.com/pb33f/libopenapi/datamodel/high/v2"
	"github.com/pb33f/libopenapi/orderedmap"
	"net/http"
	"sort"
	"strings"
	"sync"
)

// V2Document is a wrapper around libopenapi.DocumentModel
// Implements Document interface
type V2Document struct {
	*libopenapi.DocumentModel[v2high.Swagger]
	ParseConfig *config.ParseConfig
}

// V2Operation is a wrapper around libopenapi.Operation
type V2Operation struct {
	*v2high.Operation
	ParseConfig *config.ParseConfig
	mu          sync.Mutex
}

// Provider returns the SchemaProvider for this document
func (d *V2Document) Provider() config.SchemaProvider {
	return config.LibOpenAPIProvider
}

// GetVersion returns the version of the document
func (d *V2Document) GetVersion() string {
	return d.Model.Swagger
}

// GetResources returns a map of resource names and their methods.
func (d *V2Document) GetResources() map[string][]string {
	res := make(map[string][]string)

	if d.Model.Paths.PathItems == nil {
		for name, path := range d.Model.Paths.PathItems.FromOldest() {
			res[name] = make([]string, 0)
			for method := range path.GetOperations().KeysFromOldest() {
				res[name] = append(res[name], strings.ToUpper(method))
			}
		}
	}
	return res
}

func (d *V2Document) GetSecurity() openapi.SecurityComponents {
	schemes := d.Model.SecurityDefinitions
	if schemes == nil {
		return nil
	}

	definitions := schemes.Definitions
	if definitions == nil {
		return nil
	}

	res := make(openapi.SecurityComponents)
	for name, v := range definitions.FromOldest() {
		if v == nil {
			continue
		}

		paramName := v.Name
		var scheme openapi.AuthScheme

		in := openapi.AuthLocationHeader
		switch strings.ToLower(v.In) {
		case "header":
			in = openapi.AuthLocationHeader
		case "query":
			in = openapi.AuthLocationQuery
		}

		var typ openapi.AuthType
		switch v.Type {
		case "basic":
			typ = openapi.AuthTypeHTTP
			scheme = openapi.AuthSchemeBasic
		case "apiKey":
			scheme = openapi.AuthSchemeBearer
			if paramName == "authorization" {
				typ = openapi.AuthTypeHTTP
			} else {
				typ = openapi.AuthTypeApiKey
			}
		default:
			continue
		}

		res[name] = &openapi.SecurityComponent{
			Type:   typ,
			Scheme: scheme,
			In:     in,
			Name:   paramName,
		}
	}

	return res
}

// FindOperation finds an operation by resource and method.
func (d *V2Document) FindOperation(options *openapi.OperationDescription) openapi.Operation {
	if options == nil {
		return nil
	}
	path, ok := d.Model.Paths.PathItems.Get(options.Resource)
	if !ok {
		return nil
	}

	if pathOps := path.GetOperations(); pathOps != nil {
		for m, op := range pathOps.FromOldest() {
			if strings.EqualFold(m, options.Method) {
				return &V2Operation{
					Operation:   op,
					ParseConfig: d.ParseConfig,
				}
			}
		}
	}

	return nil
}

// ID returns the operation ID
func (op *V2Operation) ID() string {
	return op.Operation.OperationId
}

func (op *V2Operation) GetRequest(securityComponents openapi.SecurityComponents) *openapi.Request {
	params := op.getParameters(securityComponents)
	content, contentType := op.getRequestBody()

	return &openapi.Request{
		Parameters: params,
		Body: &openapi.RequestBody{
			Schema: content,
			Type:   contentType,
		},
	}
}

// getParameters returns a list of parameters for this operation
func (op *V2Operation) getParameters(securityComponents openapi.SecurityComponents) openapi.Parameters {
	if securityComponents == nil {
		securityComponents = make(openapi.SecurityComponents)
	}

	params := make(openapi.Parameters, 0)

	for _, param := range op.Parameters {
		required := false
		if param.Required != nil {
			required = *param.Required
		}

		params = append(params, &openapi.Parameter{
			Name:     param.Name,
			In:       param.In,
			Required: required,
			Schema:   op.parseParameter(param),
		})
	}

	// loop through the required security components
	for _, name := range op.getSecurity() {
		sec := securityComponents[name]
		if sec == nil {
			continue
		}
		var format string
		switch sec.Type {
		case openapi.AuthTypeHTTP:
			switch sec.Scheme {
			case openapi.AuthSchemeBearer:
				format = "bearer"
			case openapi.AuthSchemeBasic:
				format = "basic"
			default:
				continue

			}
			params = append(params, &openapi.Parameter{
				Name:     "authorization",
				In:       "header",
				Required: true,
				Schema: &openapi.Schema{
					Type:   openapi.TypeString,
					Format: format,
				},
			})
		case openapi.AuthTypeApiKey:
			params = append(params, &openapi.Parameter{
				Name:     sec.Name,
				In:       string(sec.In),
				Required: true,
				Schema: &openapi.Schema{
					Type: openapi.TypeString,
				},
			})
		}
	}

	// original names not sorted
	sort.Slice(params, func(i, j int) bool {
		return params[i].Name < params[j].Name
	})

	return params
}

func (op *V2Operation) getSecurity() []string {
	securityReqs := op.Security
	if securityReqs == nil {
		return nil
	}

	res := make([]string, 0)

	for _, securityReq := range securityReqs {
		if securityReq == nil || securityReq.Requirements == nil {
			continue
		}
		for secName, _ := range securityReq.Requirements.FromOldest() {
			res = append(res, secName)
		}
	}
	return res
}

// GetResponse returns the response for this operation
func (op *V2Operation) GetResponse() *openapi.Response {
	available := op.Responses.Codes

	var responseRef *v2high.Response
	statusCode := http.StatusOK

	if available != nil {
		for _, code := range []int{http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent} {
			ok := false
			responseRef, ok = available.Get(fmt.Sprintf("%v", code))
			if ok {
				statusCode = code
				break
			}
		}
	}

	// Get first defined
	if responseRef == nil && available != nil {
		for codeName, respRef := range available.FromOldest() {
			// There's no default expected in this library implementation
			responseRef = respRef
			statusCode = openapi.TransformHTTPCode(codeName)
			break
		}
	}

	if responseRef == nil {
		responseRef = op.Responses.Default
	}

	if responseRef == nil {
		return &openapi.Response{}
	}

	contentType := "application/json"
	prioTypes := []string{"application/json", "text/plain", "text/html"}
	for _, cType := range prioTypes {
		for _, produces := range op.Produces {
			contentType = produces
			if produces == cType {
				break
			}
		}
	}

	// libopenapi is missing required property for header
	parsedHeaders := make(openapi.Headers)
	responseHeaders := responseRef.Headers
	if responseHeaders == nil {
		responseHeaders = orderedmap.New[string, *v2high.Header]()
	}
	for name, header := range responseHeaders.FromOldest() {
		var items *openapi.Schema

		hItems := header.Items
		if hItems != nil {
			enums := make([]any, 0)
			for _, e := range hItems.Enum {
				enums = append(enums, e)
			}

			items = &openapi.Schema{
				Type:       hItems.Type,
				Format:     hItems.Format,
				Minimum:    float64(hItems.Minimum),
				Maximum:    float64(hItems.Maximum),
				MultipleOf: float64(hItems.MultipleOf),
				MinItems:   int64(hItems.MinItems),
				MaxItems:   int64(hItems.MaxItems),
				Pattern:    hItems.Pattern,
				Enum:       enums,
				Default:    hItems.Default,
			}
		}

		schema := &openapi.Schema{
			Type:       header.Type,
			Items:      items,
			Format:     header.Format,
			Minimum:    float64(header.Minimum),
			Maximum:    float64(header.Maximum),
			MultipleOf: float64(header.MultipleOf),
			MinItems:   int64(header.MinItems),
			MaxItems:   int64(header.MaxItems),
			Pattern:    header.Pattern,
			Enum:       header.Enum,
			Default:    header.Default,
		}

		name = strings.ToLower(name)
		parsedHeaders[name] = &openapi.Parameter{
			Name:   name,
			In:     openapi.ParameterInHeader,
			Schema: schema,
		}
	}

	if len(parsedHeaders) == 0 {
		parsedHeaders = nil
	}

	var content *openapi.Schema
	if responseRef.Schema != nil {
		schema := responseRef.Schema.Schema()
		content = NewSchema(schema, op.ParseConfig)
	}

	return &openapi.Response{
		Headers:     parsedHeaders,
		Content:     content,
		ContentType: contentType,
		StatusCode:  statusCode,
	}
}

// GetRequestBody returns the request body for this operation
func (op *V2Operation) getRequestBody() (*openapi.Schema, string) {
	var body *v2high.Parameter
	for _, param := range op.Parameters {
		// https://swagger.io/specification/v2/#parameter-object
		// The payload that's appended to the HTTP request.
		// Since there can only be one payload, there can only be one body parameter.
		if param.In == openapi.ParameterInBody {
			body = param
			continue
		}
	}

	if body == nil {
		return nil, ""
	}

	contentType := "application/json"
	typesOrder := []string{"application/json", "multipart/form-data", "application/x-www-form-urlencoded"}
	for _, cType := range typesOrder {
		for _, consumes := range op.Consumes {
			contentType = consumes
			if consumes == cType {
				break
			}
		}
	}

	if body.Schema != nil {
		return NewSchema(body.Schema.Schema(), op.ParseConfig), contentType
	}

	return nil, contentType
}

// WithParseConfig sets the ParseConfig for the operation
func (op *V2Operation) WithParseConfig(parseConfig *config.ParseConfig) openapi.Operation {
	op.mu.Lock()
	defer op.mu.Unlock()

	op.ParseConfig = parseConfig
	return op
}

func (op *V2Operation) parseParameter(param *v2high.Parameter) *openapi.Schema {
	schemaProxy := param.Schema
	if schemaProxy != nil {
		return NewSchema(schemaProxy.Schema(), op.ParseConfig)
	}

	minimum := 0
	if param.Minimum != nil {
		minimum = *param.Minimum
	}

	maximum := 0
	if param.Maximum != nil {
		maximum = *param.Maximum
	}

	multipleOf := 0
	if param.MultipleOf != nil {
		multipleOf = *param.MultipleOf
	}

	minItems := 0
	if param.MinItems != nil {
		minItems = *param.MinItems
	}

	maxItems := 0
	if param.MaxItems != nil {
		maxItems = *param.MaxItems
	}

	minLength := 0
	if param.MinLength != nil {
		minLength = *param.MinLength
	}

	maxLength := 0
	if param.MaxLength != nil {
		maxLength = *param.MaxLength
	}

	var items *openapi.Schema
	if param.Items != nil {
		enums := make([]any, 0)
		for _, e := range param.Items.Enum {
			enums = append(enums, e)
		}
		items = &openapi.Schema{
			Type:       param.Items.Type,
			Format:     param.Items.Format,
			Minimum:    float64(param.Items.Minimum),
			Maximum:    float64(param.Items.Maximum),
			MultipleOf: float64(param.Items.MultipleOf),
			MinItems:   int64(param.Items.MinItems),
			MaxItems:   int64(param.Items.MaxItems),
			MinLength:  int64(param.Items.MinLength),
			MaxLength:  int64(param.Items.MaxLength),
			Pattern:    param.Items.Pattern,
			Enum:       enums,
			Default:    param.Items.Default,
		}
	}

	enums := make([]any, 0)
	for _, e := range param.Enum {
		enums = append(enums, e)
	}

	return &openapi.Schema{
		Items:      items,
		Type:       param.Type,
		Format:     param.Format,
		Minimum:    float64(minimum),
		Maximum:    float64(maximum),
		MultipleOf: float64(multipleOf),
		MinItems:   int64(minItems),
		MaxItems:   int64(maxItems),
		MinLength:  int64(minLength),
		MaxLength:  int64(maxLength),
		Pattern:    param.Pattern,
		Enum:       enums,
		Default:    param.Default,
	}
}
