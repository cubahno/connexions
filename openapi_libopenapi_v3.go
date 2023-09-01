package connexions

import (
	"fmt"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
	"net/http"
	"sort"
	"strings"
	"sync"
)

type LibV3Document struct {
	*libopenapi.DocumentModel[v3high.Document]
}

type LibV3Operation struct {
	*v3high.Operation
	parseConfig *ParseConfig
	mu          sync.Mutex
}

func (d *LibV3Document) Provider() SchemaProvider {
	return LibOpenAPIProvider
}

func (d *LibV3Document) GetVersion() string {
	return d.Model.Version
}

func (d *LibV3Document) GetResources() map[string][]string {
	res := make(map[string][]string)

	for name, path := range d.Model.Paths.PathItems {
		res[name] = make([]string, 0)
		for method, _ := range path.GetOperations() {
			res[name] = append(res[name], strings.ToUpper(method))
		}
	}
	return res
}

func (d *LibV3Document) FindOperation(options *FindOperationOptions) Operationer {
	if options == nil {
		return nil
	}
	path, ok := d.Model.Paths.PathItems[options.Resource]
	if !ok {
		return nil
	}

	for m, op := range path.GetOperations() {
		if strings.ToUpper(m) == strings.ToUpper(options.Method) {
			return &LibV3Operation{
				Operation: op,
			}
		}
	}

	return nil
}

func (op *LibV3Operation) GetParameters() OpenAPIParameters {
	params := make(OpenAPIParameters, 0)

	for _, param := range op.Parameters {
		var schema *Schema
		if param.Schema != nil {
			px := param.Schema
			schema = NewSchemaFromLibOpenAPI(px.Schema(), op.parseConfig)
		}

		params = append(params, &OpenAPIParameter{
			Name:     param.Name,
			In:       param.In,
			Required: param.Required,
			Schema:   schema,
			Example:  param.Example,
		})
	}

	// original names not sorted
	sort.Slice(params, func(i, j int) bool {
		return params[i].Name < params[j].Name
	})

	return params
}

func (op *LibV3Operation) GetResponse() *OpenAPIResponse {
	available := op.Responses.Codes

	var responseRef *v3high.Response
	statusCode := http.StatusOK

	for _, code := range []int{http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent} {
		responseRef = available[fmt.Sprintf("%v", code)]
		if responseRef != nil {
			statusCode = code
			break
		}
	}

	// Get first defined
	if responseRef == nil {
		for codeName, respRef := range available {
			// There's no default expected in this library implementation
			responseRef = respRef
			statusCode = TransformHTTPCode(codeName)
			break
		}
	}

	if responseRef == nil {
		responseRef = op.Responses.Default
	}

	if responseRef == nil {
		return &OpenAPIResponse{
			StatusCode: statusCode,
		}
	}

	parsedHeaders := make(OpenAPIHeaders)
	for name, header := range responseRef.Headers {
		var schema *Schema
		if header.Schema != nil {
			hSchema := header.Schema.Schema()
			schema = NewSchemaFromLibOpenAPI(hSchema, op.parseConfig)
		}

		name = strings.ToLower(name)
		parsedHeaders[name] = &OpenAPIParameter{
			Name:     name,
			In:       ParameterInHeader,
			Required: header.Required,
			Schema:   schema,
		}
	}

	if len(parsedHeaders) == 0 {
		parsedHeaders = nil
	}

	libContent, contentType := op.getContent(responseRef.Content)
	content := NewSchemaFromLibOpenAPI(libContent, op.parseConfig)

	return &OpenAPIResponse{
		Headers:     parsedHeaders,
		Content:     content,
		ContentType: contentType,
		StatusCode:  statusCode,
	}
}

func (op *LibV3Operation) getContent(contentTypes map[string]*v3high.MediaType) (*base.Schema, string) {
	if len(contentTypes) == 0 {
		contentTypes = make(map[string]*v3high.MediaType)
	}

	prioTypes := []string{"application/json", "text/plain", "text/html"}
	for _, contentType := range prioTypes {
		if _, ok := contentTypes[contentType]; ok {
			return contentTypes[contentType].Schema.Schema(), contentType
		}
	}

	// If none of the priority types are found, return the first available media type
	for contentType, mediaType := range contentTypes {
		return mediaType.Schema.Schema(), contentType
	}

	return nil, ""
}

func (op *LibV3Operation) GetRequestBody() (*Schema, string) {
	if op.RequestBody == nil {
		return nil, ""
	}

	contentTypes := op.RequestBody.Content
	if len(contentTypes) == 0 {
		contentTypes = make(map[string]*v3high.MediaType)
	}

	typesOrder := []string{"application/json", "multipart/form-data", "application/x-www-form-urlencoded"}
	for _, contentType := range typesOrder {
		if _, ok := contentTypes[contentType]; ok {
			px := contentTypes[contentType].Schema
			return NewSchemaFromLibOpenAPI(px.Schema(), op.parseConfig), contentType
		}
	}

	// Get first defined
	for contentType, mediaType := range contentTypes {
		px := mediaType.Schema
		return NewSchemaFromLibOpenAPI(px.Schema(), op.parseConfig), contentType
	}

	return nil, ""
}

func (op *LibV3Operation) WithParseConfig(parseConfig *ParseConfig) Operationer {
	op.mu.Lock()
	defer op.mu.Unlock()

	op.parseConfig = parseConfig
	return op
}