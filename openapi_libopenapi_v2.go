package connexions

import (
	"fmt"
	"github.com/pb33f/libopenapi"
	v2high "github.com/pb33f/libopenapi/datamodel/high/v2"
	"net/http"
	"sort"
	"strings"
	"sync"
)

// LibV2Document is a wrapper around libopenapi.DocumentModel
// Implements Document interface
type LibV2Document struct {
	*libopenapi.DocumentModel[v2high.Swagger]
	ParseConfig *ParseConfig
}

// LibV2Operation is a wrapper around libopenapi.Operation
type LibV2Operation struct {
	*v2high.Operation
	ParseConfig *ParseConfig
	mu          sync.Mutex
}

// Provider returns the SchemaProvider for this document
func (d *LibV2Document) Provider() SchemaProvider {
	return LibOpenAPIProvider
}

// GetVersion returns the version of the document
func (d *LibV2Document) GetVersion() string {
	return d.Model.Swagger
}

// GetResources returns a map of resource names and their methods.
func (d *LibV2Document) GetResources() map[string][]string {
	res := make(map[string][]string)

	for name, path := range d.Model.Paths.PathItems {
		res[name] = make([]string, 0)
		for method := range path.GetOperations() {
			res[name] = append(res[name], strings.ToUpper(method))
		}
	}
	return res
}

// FindOperation finds an operation by resource and method.
func (d *LibV2Document) FindOperation(options *OperationDescription) Operationer {
	if options == nil {
		return nil
	}
	path, ok := d.Model.Paths.PathItems[options.Resource]
	if !ok {
		return nil
	}

	for m, op := range path.GetOperations() {
		if strings.EqualFold(m, options.Method) {
			return &LibV2Operation{
				Operation:   op,
				ParseConfig: d.ParseConfig,
			}
		}
	}

	return nil
}

// ID returns the operation ID
func (op *LibV2Operation) ID() string {
	return op.Operation.OperationId
}

// GetParameters returns a list of parameters for this operation
func (op *LibV2Operation) GetParameters() OpenAPIParameters {
	params := make(OpenAPIParameters, 0)

	for _, param := range op.Parameters {
		required := false
		if param.Required != nil {
			required = *param.Required
		}

		params = append(params, &OpenAPIParameter{
			Name:     param.Name,
			In:       param.In,
			Required: required,
			Schema:   op.parseParameter(param),
		})
	}

	// original names not sorted
	sort.Slice(params, func(i, j int) bool {
		return params[i].Name < params[j].Name
	})

	return params
}

// GetResponse returns the response for this operation
func (op *LibV2Operation) GetResponse() *OpenAPIResponse {
	available := op.Responses.Codes

	var responseRef *v2high.Response
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
		return &OpenAPIResponse{}
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
	parsedHeaders := make(OpenAPIHeaders)
	for name, header := range responseRef.Headers {
		var items *Schema

		hItems := header.Items
		if hItems != nil {
			items = &Schema{
				Type:       hItems.Type,
				Format:     hItems.Format,
				Minimum:    float64(hItems.Minimum),
				Maximum:    float64(hItems.Maximum),
				MultipleOf: float64(hItems.MultipleOf),
				MinItems:   int64(hItems.MinItems),
				MaxItems:   int64(hItems.MaxItems),
				Pattern:    hItems.Pattern,
				Enum:       hItems.Enum,
				Default:    hItems.Default,
			}
		}

		schema := &Schema{
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
		parsedHeaders[name] = &OpenAPIParameter{
			Name:   name,
			In:     ParameterInHeader,
			Schema: schema,
		}
	}

	if len(parsedHeaders) == 0 {
		parsedHeaders = nil
	}

	var content *Schema
	if responseRef.Schema != nil {
		schema := responseRef.Schema.Schema()
		content = NewSchemaFromLibOpenAPI(schema, op.ParseConfig)
	}

	return &OpenAPIResponse{
		Headers:     parsedHeaders,
		Content:     content,
		ContentType: contentType,
		StatusCode:  statusCode,
	}
}

// GetRequestBody returns the request body for this operation
func (op *LibV2Operation) GetRequestBody() (*Schema, string) {
	var body *v2high.Parameter
	for _, param := range op.Parameters {
		// https://swagger.io/specification/v2/#parameter-object
		// The payload that's appended to the HTTP request.
		// Since there can only be one payload, there can only be one body parameter.
		if param.In == ParameterInBody {
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
		return NewSchemaFromLibOpenAPI(body.Schema.Schema(), op.ParseConfig), contentType
	}

	return nil, contentType
}

// WithParseConfig sets the ParseConfig for the operation
func (op *LibV2Operation) WithParseConfig(parseConfig *ParseConfig) Operationer {
	op.mu.Lock()
	defer op.mu.Unlock()

	op.ParseConfig = parseConfig
	return op
}

func (op *LibV2Operation) parseParameter(param *v2high.Parameter) *Schema {
	schemaProxy := param.Schema
	if schemaProxy != nil {
		return NewSchemaFromLibOpenAPI(schemaProxy.Schema(), op.ParseConfig)
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

	var items *Schema
	if param.Items != nil {
		items = &Schema{
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
			Enum:       param.Items.Enum,
			Default:    param.Items.Default,
		}
	}

	return &Schema{
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
		Enum:       param.Enum,
		Default:    param.Default,
	}
}
