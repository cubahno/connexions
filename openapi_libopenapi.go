package connexions

import (
	"fmt"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	v2high "github.com/pb33f/libopenapi/datamodel/high/v2"
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/resolver"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
)

type LibV2Document struct {
	libopenapi.DocumentModel[v2high.Swagger]
}

type LibV3Document struct {
	*libopenapi.DocumentModel[v3high.Document]
	ParseConfig *ParseConfig
}

type LibV3Operation struct {
	*v3high.Operation
	ParseConfig *ParseConfig
	withCache   bool
	mu          sync.Mutex
}

func NewLibOpenAPIDocumentFromFile(filePath string) (Document, error) {
	src, _ := os.ReadFile(filePath)

	lib, err := libopenapi.NewDocument(src)
	if err != nil {
		return nil, err
	}

	model, errs := lib.BuildV3Model()
	if len(errs) > 0 {
		for i := range errs {
			if circError, ok := errs[i].(*resolver.ResolvingError); ok {
				log.Printf("Message: %s\n--> Loop starts line %d | Polymorphic? %v\n\n",
					circError.Error(),
					circError.CircularReference.LoopPoint.Node.Line,
					circError.CircularReference.IsPolymorphicResult)
				return &LibV3Document{
					DocumentModel: model,
				}, nil
			}
		}
		return nil, errs[0]
	}

	return &LibV3Document{
		DocumentModel: model,
	}, nil
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
				Operation:   op,
				ParseConfig: d.ParseConfig,
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
			schema = NewSchemaFromLibOpenAPI(px.Schema(), op.ParseConfig)
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
		return &OpenAPIResponse{}
	}

	parsedHeaders := make(OpenAPIHeaders)
	for name, header := range responseRef.Headers {
		if header.Schema == nil {
			continue
		}
		hSchema := header.Schema.Schema()
		schema := NewSchemaFromLibOpenAPI(hSchema, op.ParseConfig)

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
	content := NewSchemaFromLibOpenAPI(libContent, op.ParseConfig)

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
			return NewSchemaFromLibOpenAPI(px.Schema(), op.ParseConfig), contentType
		}
	}

	// Get first defined
	for contentType, mediaType := range contentTypes {
		px := mediaType.Schema
		return NewSchemaFromLibOpenAPI(px.Schema(), op.ParseConfig), contentType
	}

	return nil, ""
}

func (op *LibV3Operation) WithParseConfig(parseConfig *ParseConfig) Operationer {
	op.mu.Lock()
	defer op.mu.Unlock()

	op.ParseConfig = parseConfig
	return op
}

func NewSchemaFromLibOpenAPI(schema *base.Schema, parseConfig *ParseConfig) *Schema {
	if parseConfig == nil {
		parseConfig = &ParseConfig{}
	}
	return newSchemaFromLibOpenAPI(schema, parseConfig, nil, nil)
}

func newSchemaFromLibOpenAPI(schema *base.Schema, parseConfig *ParseConfig, refPath []string, namePath []string) *Schema {
	if schema == nil {
		return nil
	}

	if len(refPath) == 0 {
		refPath = make([]string, 0)
	}

	if len(namePath) == 0 {
		namePath = make([]string, 0)
	}

	if parseConfig.MaxLevels > 0 && len(namePath) > parseConfig.MaxLevels {
		return nil
	}

	if !IsSliceUnique(refPath) {
		return nil
	}

	schema, mergedReference := mergeLibOpenAPISubSchemas(schema)

	typ := ""
	if len(schema.Type) > 0 {
		typ = schema.Type[0]
	}

	var items *Schema
	if schema.Items != nil && schema.Items.IsA() {
		libItems := schema.Items.A
		sub := libItems.Schema()
		ref := libItems.GetReference()
		items = newSchemaFromLibOpenAPI(sub,
			parseConfig,
			AppendSliceFirstNonEmpty(refPath, ref, mergedReference),
			namePath)
	}

	properties := make(map[string]*Schema)
	for propName, sProxy := range schema.Properties {
		if parseConfig.OnlyRequired && !SliceContains(schema.Required, propName) {
			continue
		}
		properties[propName] = newSchemaFromLibOpenAPI(sProxy.Schema(),
			parseConfig,
			AppendSliceFirstNonEmpty(refPath, sProxy.GetReference(), mergedReference),
			append(namePath, propName))
	}

	return &Schema{
		Type:          typ,
		Items:         items,
		MultipleOf:    RemovePointer(schema.MultipleOf),
		Maximum:       RemovePointer(schema.Maximum),
		Minimum:       RemovePointer(schema.Minimum),
		MaxLength:     RemovePointer(schema.MaxLength),
		MinLength:     RemovePointer(schema.MinLength),
		Pattern:       schema.Pattern,
		Format:        schema.Format,
		MaxItems:      RemovePointer(schema.MaxItems),
		MinItems:      RemovePointer(schema.MinItems),
		MaxProperties: RemovePointer(schema.MaxProperties),
		MinProperties: RemovePointer(schema.MinProperties),
		Required:      schema.Required,
		Enum:          schema.Enum,
		Properties:    properties,
		Default:       schema.Default,
		Nullable:      RemovePointer(schema.Nullable),
		ReadOnly:      schema.ReadOnly,
		WriteOnly:     schema.WriteOnly,
		Example:       schema.Example,
		Deprecated:    RemovePointer(schema.Deprecated),
	}
}

// PicklLibOpenAPISchemaProxy returns the first non-nil schema proxy with reference
// or the first non-nil schema proxy if none of them have reference.
func PicklLibOpenAPISchemaProxy(items []*base.SchemaProxy) *base.SchemaProxy {
	if len(items) == 0 {
		return nil
	}

	for _, item := range items {
		if item == nil {
			continue
		}

		if item.GetReference() != "" {
			return item
		}
	}

	return items[0]
}

func mergeLibOpenAPISubSchemas(schema *base.Schema) (*base.Schema, string) {
	allOf := schema.AllOf
	anyOf := schema.AnyOf
	oneOf := schema.OneOf
	not := schema.Not

	// base case: schema is flat
	if len(allOf) == 0 && len(anyOf) == 0 && len(oneOf) == 0 && not == nil {
		return schema, ""
	}

	// reset
	schema.AllOf = nil
	schema.AnyOf = nil
	schema.OneOf = nil
	schema.Not = nil

	properties := schema.Properties
	if len(properties) == 0 {
		properties = make(map[string]*base.SchemaProxy)
	}
	required := schema.Required
	if len(required) == 0 {
		required = make([]string, 0)
	}

	impliedType := ""
	if len(allOf) > 0 {
		impliedType = TypeObject
	}

	// pick one from each
	allOf = append(allOf,
		PicklLibOpenAPISchemaProxy(anyOf),
		PicklLibOpenAPISchemaProxy(oneOf),
	)

	subRef := ""
	for _, schemaRef := range allOf {
		if schemaRef == nil {
			continue
		}

		subSchema := schemaRef.Schema()
		if subSchema == nil {
			continue
		}

		if subRef == "" && schemaRef.IsReference() {
			subRef = schemaRef.GetReference()
		}

		if impliedType == "" {
			if len(subSchema.Type) > 0 {
				impliedType = subSchema.Type[0]
			}
			if impliedType == "" && subSchema.Items.IsA() && len(subSchema.Items.A.Schema().Properties) > 0 {
				impliedType = TypeArray
			}
			if impliedType == "" {
				impliedType = TypeObject
			}
		}

		if impliedType == TypeObject {
			for propertyName, property := range subSchema.Properties {
				if subRef == "" {
					subRef = property.GetReference()
				}
				properties[propertyName] = property
			}
		}

		if impliedType == TypeArray && subSchema.Items.IsA() {
			if subRef == "" {
				subRef = subSchema.Items.A.GetReference()
			}
			schema.Items = subSchema.Items
		}

		required = append(required, subSchema.Required...)
	}

	// make required unique
	required = SliceUnique(required)

	// exclude properties from `not`
	if not != nil {
		notSchema := not.Schema()
		deletes := map[string]bool{}
		for propertyName, _ := range notSchema.Properties {
			delete(properties, propertyName)
			deletes[propertyName] = true
		}

		// remove from required properties
		for i, propertyName := range required {
			if _, ok := deletes[propertyName]; ok {
				required = append(required[:i], required[i+1:]...)
			}
		}
	}

	schema.Properties = properties
	schema.Type = []string{impliedType}
	schema.Required = required

	return schema, subRef
}
