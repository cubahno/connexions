package connexions

import (
	"github.com/getkin/kin-openapi/openapi3"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// KinDocument is a wrapper around openapi3.T
// Implements Document interface
type KinDocument struct {
	*openapi3.T
}

// KinOperation is a wrapper around openapi3.Operation
type KinOperation struct {
	*openapi3.Operation
	parseConfig *ParseConfig
	mu          sync.Mutex
}

// NewKinDocumentFromFile creates a new KinDocument from a file path
func NewKinDocumentFromFile(filePath string) (Document, error) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(filePath)
	if err != nil {
		return nil, err
	}
	return &KinDocument{
		T: doc,
	}, err
}

// Provider returns the SchemaProvider for this document
func (d *KinDocument) Provider() SchemaProvider {
	return KinOpenAPIProvider
}

// GetVersion returns the version of the document
func (d *KinDocument) GetVersion() string {
	return d.OpenAPI
}

// GetResources returns a map of resource names and their methods.
func (d *KinDocument) GetResources() map[string][]string {
	res := make(map[string][]string)
	for resName, pathItem := range d.Paths {
		res[resName] = make([]string, 0)
		for method := range pathItem.Operations() {
			res[resName] = append(res[resName], method)
		}
	}
	return res
}

// FindOperation finds an operation by resource and method.
func (d *KinDocument) FindOperation(options *OperationDescription) Operationer {
	if options == nil {
		return nil
	}

	path := d.Paths.Find(options.Resource)
	if path == nil {
		return nil
	}
	op := path.GetOperation(strings.ToUpper(options.Method))
	if op == nil {
		return nil
	}
	return &KinOperation{
		Operation: op,
	}
}

// ID returns the operation ID
func (op *KinOperation) ID() string {
	return op.Operation.OperationID
}

// GetParameters returns the operation parameters
func (op *KinOperation) GetParameters() OpenAPIParameters {
	var params []*OpenAPIParameter
	for _, param := range op.Parameters {
		p := param.Value
		if p == nil {
			continue
		}

		var schema *Schema
		if p.Schema != nil {
			schema = NewSchemaFromKin(p.Schema.Value, op.parseConfig)
		}

		params = append(params, &OpenAPIParameter{
			Name:     p.Name,
			In:       p.In,
			Required: p.Required,
			Schema:   schema,
		})
	}

	sort.Slice(params, func(i, j int) bool {
		return params[i].Name < params[j].Name
	})

	return params
}

// GetRequestBody returns the operation request body
func (op *KinOperation) GetRequestBody() (*Schema, string) {
	if op.RequestBody == nil {
		return nil, ""
	}

	schema := op.RequestBody.Value
	contentTypes := schema.Content
	if len(contentTypes) == 0 {
		return nil, ""
	}

	typesOrder := []string{"application/json", "multipart/form-data", "application/x-www-form-urlencoded"}
	for _, contentType := range typesOrder {
		if _, ok := contentTypes[contentType]; ok {
			return NewSchemaFromKin(contentTypes[contentType].Schema.Value, op.parseConfig), contentType
		}
	}

	// Get first defined
	var content *Schema
	var contentType string

	for contentTyp, mediaType := range contentTypes {
		content = NewSchemaFromKin(mediaType.Schema.Value, op.parseConfig)
		contentType = contentTyp
		break
	}

	return content, contentType
}

// GetResponse returns the operation response
func (op *KinOperation) GetResponse() *OpenAPIResponse {
	available := op.Responses

	var libResponse *openapi3.Response
	statusCode := 200

	for _, code := range []int{http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent} {
		if codeResp := available.Get(code); codeResp != nil {
			libResponse = codeResp.Value
			statusCode = code
			break
		}
	}

	// Get first defined
	if libResponse == nil {
		for codeName, respRef := range available {
			if codeName == "default" || respRef == nil {
				continue
			}
			statusCode = TransformHTTPCode(codeName)
			libResponse = respRef.Value
			break
		}
	}

	if libResponse == nil && available.Default() != nil {
		libResponse = available.Default().Value
	}

	if libResponse == nil {
		return &OpenAPIResponse{
			StatusCode: statusCode,
		}
	}

	libContent, contentType := op.getContent(libResponse.Content)

	headers := make(OpenAPIHeaders)
	for name, header := range libResponse.Headers {
		ref := header.Value
		if ref == nil {
			continue
		}
		p := ref.Parameter
		var schema *Schema
		if p.Schema != nil && p.Schema.Value != nil {
			schema = NewSchemaFromKin(p.Schema.Value, op.parseConfig)
		}
		name = strings.ToLower(name)
		headers[name] = &OpenAPIParameter{
			Name:     name,
			In:       ParameterInHeader,
			Required: p.Required,
			Schema:   schema,
		}
	}

	if len(headers) == 0 {
		headers = nil
	}

	return &OpenAPIResponse{
		Headers:     headers,
		Content:     NewSchemaFromKin(libContent, op.parseConfig),
		StatusCode:  statusCode,
		ContentType: contentType,
	}
}

// getContent returns the content and content type of the operation response
func (op *KinOperation) getContent(types map[string]*openapi3.MediaType) (*openapi3.Schema, string) {
	if len(types) == 0 {
		return nil, ""
	}

	prioTypes := []string{"application/json", "text/plain", "text/html"}
	for _, contentType := range prioTypes {
		if _, ok := types[contentType]; ok {
			return types[contentType].Schema.Value, contentType
		}
	}

	var content *openapi3.Schema
	var contentType string

	for contentTyp, mediaType := range types {
		content = mediaType.Schema.Value
		contentType = contentTyp
		break
	}

	return content, contentType
}

// WithParseConfig sets the parse config for this operation
func (op *KinOperation) WithParseConfig(config *ParseConfig) Operationer {
	op.mu.Lock()
	defer op.mu.Unlock()

	op.parseConfig = config
	return op
}

// NewSchemaFromKin creates a new Schema from a Kin schema
func NewSchemaFromKin(schema *openapi3.Schema, parseConfig *ParseConfig) *Schema {
	if parseConfig == nil {
		parseConfig = &ParseConfig{}
	}
	return newSchemaFromKin(schema, parseConfig, nil, nil)
}

func newSchemaFromKin(schema *openapi3.Schema, parseConfig *ParseConfig, refPath []string, namePath []string) *Schema {
	if schema == nil {
		return nil
	}

	merged, mergedReference := mergeKinSubSchemas(schema)

	if len(refPath) == 0 {
		refPath = make([]string, 0)
	}

	if len(namePath) == 0 {
		namePath = make([]string, 0)
	}

	if parseConfig.MaxLevels > 0 && len(namePath) > parseConfig.MaxLevels {
		return nil
	}

	if GetSliceMaxRepetitionNumber(refPath) > parseConfig.MaxRecursionLevels {
		return nil
	}

	typ := merged.Type
	typ = FixSchemaTypeTypos(typ)

	var items *Schema
	if merged.Items != nil && merged.Items.Value != nil {
		kinItems := merged.Items
		sub := kinItems.Value
		ref := kinItems.Ref

		// detect circular reference early
		if parseConfig.MaxRecursionLevels == 0 && SliceContains(refPath, ref) {
			return nil
		}

		items = newSchemaFromKin(sub,
			parseConfig,
			AppendSliceFirstNonEmpty(refPath, merged.Items.Ref, mergedReference),
			namePath)
	}

	var properties map[string]*Schema
	if len(merged.Properties) > 0 {
		properties = make(map[string]*Schema)
		for propName, ref := range merged.Properties {
			if parseConfig.OnlyRequired && !SliceContains(merged.Required, propName) {
				continue
			}
			properties[propName] = newSchemaFromKin(ref.Value,
				parseConfig,
				AppendSliceFirstNonEmpty(refPath, ref.Ref, mergedReference),
				AppendSliceFirstNonEmpty(namePath, propName))
		}
	}

	// add additional properties
	additionalProps := getKinAdditionalProperties(merged.AdditionalProperties)
	if additionalProps != nil {
		if properties == nil {
			properties = make(map[string]*Schema)
		}

		// TODO(cubahno): move to config
		additionalNum := 3
		additionalPrefix := "extra-"

		for i := 0; i < additionalNum; i++ {
			propName := additionalPrefix + strconv.Itoa(i+1)
			propSchema := newSchemaFromKin(
				additionalProps,
				parseConfig,
				append(refPath, "additionalProperties"), // this will solve circular reference
				append(namePath, propName),
			)
			if propSchema != nil {
				properties[propName] = propSchema
			}
		}

		if len(properties) == 0 {
			properties = nil
		}
	}

	var not *Schema
	if merged.Not != nil {
		not = newSchemaFromKin(merged.Not.Value, parseConfig, refPath, namePath)
	}

	if merged.Type == TypeArray && items == nil {
		// if no items specified means they could be anything, so let's assume string
		items = &Schema{Type: TypeString}
	}

	return &Schema{
		Type:          typ,
		Items:         items,
		MultipleOf:    RemovePointer(merged.MultipleOf),
		Maximum:       RemovePointer(merged.Max),
		Minimum:       RemovePointer(merged.Min),
		MaxLength:     int64(RemovePointer(merged.MaxLength)),
		MinLength:     int64(merged.MinLength),
		Pattern:       merged.Pattern,
		Format:        merged.Format,
		MaxItems:      int64(RemovePointer(merged.MaxItems)),
		MinItems:      int64(merged.MinItems),
		MaxProperties: int64(RemovePointer(merged.MaxProps)),
		MinProperties: int64(merged.MinProps),
		Required:      merged.Required,
		Enum:          merged.Enum,
		Properties:    properties,
		Not:           not,
		Default:       merged.Default,
		Nullable:      merged.Nullable,
		ReadOnly:      merged.ReadOnly,
		WriteOnly:     merged.WriteOnly,
		Example:       merged.Example,
		Deprecated:    merged.Deprecated,
	}
}

// mergeKinSubSchemas merges allOf, anyOf, oneOf and not into a single schema
func mergeKinSubSchemas(schema *openapi3.Schema) (*openapi3.Schema, string) {
	allOf := schema.AllOf
	anyOf := schema.AnyOf
	oneOf := schema.OneOf
	not := schema.Not

	// base case: schema is flat
	if len(allOf) == 0 && len(anyOf) == 0 && len(oneOf) == 0 && not == nil {
		if schema != nil && len(schema.Type) == 0 {
			schema.Type = TypeObject
		}
		return schema, ""
	}

	// reset
	schema.AllOf = nil
	schema.AnyOf = nil
	schema.OneOf = nil
	schema.Not = nil

	properties := schema.Properties
	if len(properties) == 0 {
		properties = make(map[string]*openapi3.SchemaRef)
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
		pickKinSchemaProxy(anyOf),
		pickKinSchemaProxy(oneOf),
	)

	subRef := ""
	for _, schemaRef := range allOf {
		if schemaRef == nil {
			continue
		}
		subSchema := schemaRef.Value

		if subRef == "" && schemaRef.Ref != "" {
			subRef = schemaRef.Ref
		}

		if impliedType == "" {
			if len(subSchema.Type) > 0 {
				impliedType = subSchema.Type
			}
			if impliedType == "" && subSchema.Items != nil && subSchema.Items.Value != nil {
				impliedType = TypeArray
			}
			if impliedType == "" {
				impliedType = TypeObject
			}
		}

		if impliedType == TypeObject {
			for propertyName, property := range subSchema.Properties {
				if subRef == "" {
					subRef = property.Ref
				}
				properties[propertyName] = property
			}
		}

		if impliedType == TypeArray && subSchema.Items != nil && subSchema.Items.Value != nil {
			if subRef == "" {
				subRef = subSchema.Items.Ref
			}
			schema.Items = subSchema.Items
		}

		// gather fom the sub
		schema.AllOf = append(schema.AllOf, subSchema.AllOf...)
		schema.AnyOf = append(schema.AnyOf, subSchema.AnyOf...)
		schema.OneOf = append(schema.OneOf, subSchema.OneOf...)
		schema.Required = append(schema.Required, subSchema.Required...)

		required = append(required, subSchema.Required...)
	}

	// make required unique
	required = SliceUnique(required)

	if not != nil {
		resultNot, _ := mergeKinSubSchemas(not.Value)
		if resultNot != nil {
			// not is always an object
			resultNot.Type = TypeObject
		}
		schema.Not = openapi3.NewSchemaRef("", resultNot)
	}

	schema.Type = impliedType
	schema.Properties = properties
	schema.Required = required

	if len(schema.AllOf) > 0 {
		return mergeKinSubSchemas(schema)
	}

	return schema, subRef
}

// pickKinSchemaProxy returns the first non-nil schema proxy with reference
// or the first non-nil schema proxy if none of them have reference.
func pickKinSchemaProxy(items []*openapi3.SchemaRef) *openapi3.SchemaRef {
	if len(items) == 0 {
		return nil
	}

	var fstNonEmpty *openapi3.SchemaRef

	for _, item := range items {
		if item == nil {
			continue
		}

		if fstNonEmpty == nil {
			fstNonEmpty = item
		}

		// prefer reference
		if item.Ref != "" {
			return item
		}
	}

	return fstNonEmpty
}

func getKinAdditionalProperties(source openapi3.AdditionalProperties) *openapi3.Schema {
	schemaRef := source.Schema
	if schemaRef == nil || schemaRef.Value == nil {
		has := RemovePointer(source.Has)
		if !has {
			return nil
		}
		// case when additionalProperties is true
		return &openapi3.Schema{
			Type: TypeString,
		}
	}

	// we have schema
	return schemaRef.Value
}
