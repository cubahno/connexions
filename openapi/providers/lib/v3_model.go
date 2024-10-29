package lib

import (
	"fmt"
	"github.com/cubahno/connexions/config"
	"github.com/cubahno/connexions/openapi"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"net/http"
	"sort"
	"strings"
	"sync"
)

// V3Document is a wrapper around libopenapi.DocumentModel
// Implements Document interface
type V3Document struct {
	*libopenapi.DocumentModel[v3high.Document]
}

// V3Operation is a wrapper around libopenapi.Operation
type V3Operation struct {
	*v3high.Operation
	parseConfig *config.ParseConfig
	mu          sync.Mutex
}

// Provider returns the SchemaProvider for this document
func (d *V3Document) Provider() config.SchemaProvider {
	return config.LibOpenAPIProvider
}

// GetVersion returns the version of the document
func (d *V3Document) GetVersion() string {
	return d.Model.Version
}

// GetResources returns a map of resource names and their methods.
func (d *V3Document) GetResources() map[string][]string {
	res := make(map[string][]string)

	if d.DocumentModel == nil || d.Model.Paths == nil {
		return res
	}

	if d.Model.Paths.PathItems != nil {
		for name, path := range d.Model.Paths.PathItems.FromOldest() {
			res[name] = make([]string, 0)
			for method := range path.GetOperations().KeysFromOldest() {
				res[name] = append(res[name], strings.ToUpper(method))
			}
		}
	}
	return res
}

func (d *V3Document) GetSecurity() openapi.SecurityComponents {
	components := d.Model.Components
	if components == nil {
		return nil
	}
	schemes := components.SecuritySchemes
	if schemes == nil {
		return nil
	}

	res := make(openapi.SecurityComponents)
	for name, v := range schemes.FromOldest() {
		if v == nil {
			continue
		}

		in := openapi.AuthLocationHeader
		switch v.In {
		case "header":
			in = openapi.AuthLocationHeader
		case "query":
			in = openapi.AuthLocationQuery
		}

		var typ openapi.AuthType
		switch v.Type {
		case "http":
			typ = openapi.AuthTypeHTTP
		case "apiKey":
			typ = openapi.AuthTypeApiKey
		default:
			continue
		}

		var scheme openapi.AuthScheme
		switch v.Scheme {
		case "bearer":
			scheme = openapi.AuthSchemeBearer
		case "basic":
			scheme = openapi.AuthSchemeBasic
		}

		res[name] = &openapi.SecurityComponent{
			Type:   typ,
			Scheme: scheme,
			In:     in,
			Name:   v.Name,
		}
	}

	return res
}

// FindOperation finds an operation by resource and method.
func (d *V3Document) FindOperation(options *openapi.OperationDescription) openapi.Operation {
	if options == nil {
		return nil
	}
	path, ok := d.Model.Paths.PathItems.Get(options.Resource)
	if !ok {
		return nil
	}

	pathOps := path.GetOperations()
	if pathOps == nil {
		return nil
	}
	for m, op := range pathOps.FromOldest() {
		if strings.EqualFold(m, options.Method) {
			return &V3Operation{
				Operation: op,
			}
		}
	}

	return nil
}

// ID returns the operation ID
func (op *V3Operation) ID() string {
	return op.Operation.OperationId
}

func (op *V3Operation) GetRequest(securityComponents openapi.SecurityComponents) *openapi.Request {
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

// getParameters returns a list of parameters for the operation
func (op *V3Operation) getParameters(securityComponents openapi.SecurityComponents) openapi.Parameters {
	params := make(openapi.Parameters, 0)
	if securityComponents == nil {
		securityComponents = make(openapi.SecurityComponents)
	}

	for _, param := range op.Parameters {
		var schema *openapi.Schema
		if param.Schema != nil {
			px := param.Schema
			schema = NewSchema(px.Schema(), op.parseConfig)
		}
		required := false
		if param.Required != nil {
			required = *param.Required
		}
		params = append(params, &openapi.Parameter{
			Name:     param.Name,
			In:       param.In,
			Required: required,
			Schema:   schema,
			Example:  param.Example,
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

func (op *V3Operation) getSecurity() []string {
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

// GetResponse returns the response for the operation.
// If no response is defined, a default response is returned.
// Responses are prioritized by status code, with 200 being the highest priority.
func (op *V3Operation) GetResponse() *openapi.Response {
	if op.Responses == nil {
		return &openapi.Response{
			StatusCode: http.StatusOK,
		}
	}
	available := op.Responses.Codes
	if available == nil {
		available = orderedmap.New[string, *v3high.Response]()
	}

	var responseRef *v3high.Response
	statusCode := http.StatusOK

	for _, code := range []int{http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent} {
		ok := false
		responseRef, ok = available.Get(fmt.Sprintf("%v", code))
		if ok {
			statusCode = code
			break
		}
	}

	// Get first defined
	if responseRef == nil {
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
		return &openapi.Response{
			StatusCode: statusCode,
		}
	}

	parsedHeaders := make(openapi.Headers)
	headers := responseRef.Headers
	if headers != nil {
		for name, header := range headers.FromOldest() {
			var schema *openapi.Schema
			if header.Schema != nil {
				hSchema := header.Schema.Schema()
				schema = NewSchema(hSchema, op.parseConfig)
			}

			name = strings.ToLower(name)
			parsedHeaders[name] = &openapi.Parameter{
				Name:     name,
				In:       openapi.ParameterInHeader,
				Required: header.Required,
				Schema:   schema,
			}
		}
	}

	if len(parsedHeaders) == 0 {
		parsedHeaders = nil
	}

	contentTypes := make(map[string]*v3high.MediaType)
	if responseRef.Content != nil {
		for k, v := range responseRef.Content.FromOldest() {
			contentTypes[k] = v
		}
	}

	libContent, contentType := op.getContent(contentTypes)
	content := NewSchema(libContent, op.parseConfig)

	return &openapi.Response{
		Headers:     parsedHeaders,
		Content:     content,
		ContentType: contentType,
		StatusCode:  statusCode,
	}
}

func (op *V3Operation) getContent(contentTypes map[string]*v3high.MediaType) (*base.Schema, string) {
	if len(contentTypes) == 0 {
		contentTypes = make(map[string]*v3high.MediaType)
	}

	prioTypes := []string{"application/json", "text/plain", "text/html"}
	for _, contentType := range prioTypes {
		if _, ok := contentTypes[contentType]; ok {
			schemaRef := contentTypes[contentType].Schema
			if schemaRef == nil {
				continue
			}
			return contentTypes[contentType].Schema.Schema(), contentType
		}
	}

	// If none of the priority types are found, return the first available media type
	for contentType, mediaType := range contentTypes {
		schemaRef := mediaType.Schema
		if schemaRef == nil {
			continue
		}
		return schemaRef.Schema(), contentType
	}

	return nil, ""
}

// GetRequestBody returns the request body for the operation.
func (op *V3Operation) getRequestBody() (*openapi.Schema, string) {
	if op.RequestBody == nil {
		return nil, ""
	}

	contentTypes := op.RequestBody.Content
	if contentTypes == nil {
		contentTypes = orderedmap.New[string, *v3high.MediaType]()
	}

	typesOrder := []string{
		"application/json",
		"multipart/form-data",
		"application/x-www-form-urlencoded",
		"application/octet-stream",
	}
	for _, contentType := range typesOrder {
		if v, ok := contentTypes.Get(contentType); ok {
			px := v.Schema
			if px == nil {
				continue
			}
			return NewSchema(px.Schema(), op.parseConfig), contentType
		}
	}

	// Get first defined
	for contentType, mediaType := range contentTypes.FromOldest() {
		px := mediaType.Schema
		if px == nil {
			continue
		}
		return NewSchema(px.Schema(), op.parseConfig), contentType
	}

	return nil, ""
}

// WithParseConfig sets the ParseConfig for the operation.
func (op *V3Operation) WithParseConfig(parseConfig *config.ParseConfig) openapi.Operation {
	op.mu.Lock()
	defer op.mu.Unlock()

	op.parseConfig = parseConfig
	return op
}
