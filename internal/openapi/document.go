package openapi

import (
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/cubahno/connexions/internal/config"
	"github.com/cubahno/connexions/internal/types"
	"github.com/getkin/kin-openapi/openapi3"
)

// KinDocument is a wrapper around openapi3.T
// Implements Document interface
type KinDocument struct {
	*openapi3.T
}

// KinOperation is a wrapper around openapi3.Operation
type KinOperation struct {
	*openapi3.Operation
	parseConfig *config.ParseConfig
	mu          sync.Mutex
}

// NewDocumentFromFile creates a new Document from a file path
func NewDocumentFromFile(filePath string) (*KinDocument, error) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(filePath)
	if err != nil {
		return nil, err
	}
	return &KinDocument{
		T: doc,
	}, err
}

// GetVersion returns the version of the document
func (d *KinDocument) GetVersion() string {
	return d.OpenAPI
}

// GetResources returns a map of resource names and their methods.
func (d *KinDocument) GetResources() map[string][]string {
	res := make(map[string][]string)
	for resName, pathItem := range d.Paths.Map() {
		res[resName] = make([]string, 0)
		for method := range pathItem.Operations() {
			res[resName] = append(res[resName], method)
		}
	}
	return res
}

func (d *KinDocument) GetSecurity() SecurityComponents {
	schemes := d.Components
	if schemes == nil {
		return nil
	}

	securitySchemes := schemes.SecuritySchemes
	if securitySchemes == nil {
		return nil
	}

	res := make(SecurityComponents)
	for name, schemeRef := range securitySchemes {
		if v := schemeRef.Value; v != nil {
			// the other allowed value is "cookie"
			in := AuthLocationHeader
			switch v.In {
			case "header":
				in = AuthLocationHeader
			case "query":
				in = AuthLocationQuery
			}

			var typ AuthType
			switch v.Type {
			case "http":
				typ = AuthTypeHTTP
			case "apiKey":
				typ = AuthTypeApiKey
			default:
				continue
			}

			var scheme AuthScheme
			switch v.Scheme {
			case "bearer":
				scheme = AuthSchemeBearer
			case "basic":
				scheme = AuthSchemeBasic
			}

			res[name] = &SecurityComponent{
				Type:   typ,
				Scheme: scheme,
				In:     in,
				Name:   v.Name,
			}
		} else if ref := schemeRef.Ref; ref != "" {
			log.Printf("Security scheme reference %s resolve is not supported yet", ref)
			continue
		}
	}
	return res
}

// FindOperation finds an operation by resource and method.
func (d *KinDocument) FindOperation(options *OperationDescription) Operation {
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

func (op *KinOperation) Unwrap() Operation {
	return op
}

func (op *KinOperation) GetRequest(securityComponents SecurityComponents) *Request {
	params := op.getParameters(securityComponents)
	content, contentType := op.getRequestBody()

	return &Request{
		Parameters: params,
		Body: &RequestBody{
			Schema: content,
			Type:   contentType,
		},
	}
}

// getParameters returns the operation parameters
func (op *KinOperation) getParameters(securityComponents SecurityComponents) Parameters {
	if securityComponents == nil {
		securityComponents = make(SecurityComponents)
	}

	var params []*Parameter
	for _, param := range op.Parameters {
		p := param.Value
		if p == nil {
			continue
		}

		var schema *types.Schema
		if p.Schema != nil {
			schema = NewSchemaFromKin(p.Schema.Value, op.parseConfig)
		}

		params = append(params, &Parameter{
			Name:     p.Name,
			In:       p.In,
			Required: p.Required,
			Schema:   schema,
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
		case AuthTypeHTTP:
			switch sec.Scheme {
			case AuthSchemeBearer:
				format = "bearer"
			case AuthSchemeBasic:
				format = "basic"
			default:
				continue

			}
			params = append(params, &Parameter{
				Name:     "authorization",
				In:       "header",
				Required: true,
				Schema: &types.Schema{
					Type:   types.TypeString,
					Format: format,
				},
			})
		case AuthTypeApiKey:
			params = append(params, &Parameter{
				Name:     sec.Name,
				In:       string(sec.In),
				Required: true,
				Schema: &types.Schema{
					Type: types.TypeString,
				},
			})
		}
	}

	sort.Slice(params, func(i, j int) bool {
		return params[i].Name < params[j].Name
	})

	return params
}

// GetRequestBody returns the operation request body
func (op *KinOperation) getRequestBody() (*types.Schema, string) {
	if op.RequestBody == nil {
		return nil, ""
	}

	schema := op.RequestBody.Value
	contentTypes := schema.Content
	if len(contentTypes) == 0 {
		return nil, ""
	}

	typesOrder := []string{
		"application/json",
		"multipart/form-data",
		"application/x-www-form-urlencoded",
		"application/octet-stream",
	}
	for _, contentType := range typesOrder {
		if _, ok := contentTypes[contentType]; ok {
			return NewSchemaFromKin(contentTypes[contentType].Schema.Value, op.parseConfig), contentType
		}
	}

	// Get first defined
	var content *types.Schema
	var contentType string

	for contentTyp, mediaType := range contentTypes {
		content = NewSchemaFromKin(mediaType.Schema.Value, op.parseConfig)
		contentType = contentTyp
		break
	}

	return content, contentType
}

func (op *KinOperation) getSecurity() []string {
	securityReqs := op.Security
	res := make([]string, 0)
	if securityReqs == nil {
		return res
	}

	for _, securityReq := range *securityReqs {
		for secName, _ := range securityReq {
			res = append(res, secName)
		}
	}
	return res
}

// GetResponse returns the operation response
func (op *KinOperation) GetResponse() *Response {
	available := op.Responses

	var libResponse *openapi3.Response
	statusCode := 200

	for _, code := range []int{http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent} {
		if codeResp := available.Value(strconv.Itoa(code)); codeResp != nil {
			libResponse = codeResp.Value
			statusCode = code
			break
		}
	}

	// Get first defined
	if libResponse == nil {
		for codeName, respRef := range available.Map() {
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
		return &Response{
			StatusCode: statusCode,
		}
	}

	libContent, contentType := op.getContent(libResponse.Content)

	headers := make(Headers)
	for name, header := range libResponse.Headers {
		ref := header.Value
		if ref == nil {
			continue
		}
		p := ref.Parameter
		var schema *types.Schema
		if p.Schema != nil && p.Schema.Value != nil {
			schema = NewSchemaFromKin(p.Schema.Value, op.parseConfig)
		}
		name = strings.ToLower(name)
		headers[name] = &Parameter{
			Name:     name,
			In:       ParameterInHeader,
			Required: p.Required,
			Schema:   schema,
		}
	}

	if len(headers) == 0 {
		headers = nil
	}

	return &Response{
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
		if _, ok := types[contentType]; !ok {
			continue
		}
		schemaRef := types[contentType].Schema
		if schemaRef == nil {
			return nil, contentType
		}

		return schemaRef.Value, contentType
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
func (op *KinOperation) WithParseConfig(config *config.ParseConfig) Operation {
	op.mu.Lock()
	defer op.mu.Unlock()

	op.parseConfig = config
	return op
}

// NewSchemaFromKin creates a new Scheme from a Kin schema
func NewSchemaFromKin(schema *openapi3.Schema, parseConfig *config.ParseConfig) *types.Schema {
	if parseConfig == nil {
		parseConfig = config.NewParseConfig()
	}
	return newSchemaFromKin(schema, parseConfig, nil, nil)
}

func newSchemaFromKin(schema *openapi3.Schema, parseConfig *config.ParseConfig, refPath []string, namePath []string) *types.Schema {
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

	if types.GetSliceMaxRepetitionNumber(refPath) > parseConfig.MaxRecursionLevels {
		return nil
	}

	typ := ""
	for _, t := range merged.Type.Slice() {
		typ = FixSchemaTypeTypos(t)
		if typ != "" {
			break
		}
	}

	var items *types.Schema
	if merged.Items != nil && merged.Items.Value != nil {
		kinItems := merged.Items
		sub := kinItems.Value
		ref := kinItems.Ref

		// detect circular reference early
		if parseConfig.MaxRecursionLevels == 0 && types.SliceContains(refPath, ref) {
			return nil
		}

		items = newSchemaFromKin(sub,
			parseConfig,
			types.AppendSliceFirstNonEmpty(refPath, merged.Items.Ref, mergedReference),
			namePath)
	}

	var properties map[string]*types.Schema
	if len(merged.Properties) > 0 {
		properties = make(map[string]*types.Schema)
		for propName, ref := range merged.Properties {
			if parseConfig.OnlyRequired && !types.SliceContains(merged.Required, propName) {
				continue
			}
			properties[propName] = newSchemaFromKin(ref.Value,
				parseConfig,
				types.AppendSliceFirstNonEmpty(refPath, ref.Ref, mergedReference),
				types.AppendSliceFirstNonEmpty(namePath, propName))
		}
	}

	// add additional properties
	additionalProps := getKinAdditionalProperties(merged.AdditionalProperties)
	if additionalProps != nil {
		if properties == nil {
			properties = make(map[string]*types.Schema)
		}

		// TODO(cubahno): find out if this the correct property, or one from AdditionalProperties should be used
		minProperties := int64(merged.MinProps)

		// TODO(cubahno): move to config
		additionalNum := 3
		if minProperties > 0 {
			additionalNum = int(minProperties)
		}

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

	var not *types.Schema
	if merged.Not != nil {
		not = newSchemaFromKin(merged.Not.Value, parseConfig, refPath, namePath)
	}

	if merged.Type.Is(types.TypeArray) && items == nil {
		// if no items specified means they could be anything, so let's assume string
		items = &types.Schema{Type: types.TypeString}
	}

	return &types.Schema{
		Type:          typ,
		Items:         items,
		MultipleOf:    types.RemovePointer(merged.MultipleOf),
		Maximum:       types.RemovePointer(merged.Max),
		Minimum:       types.RemovePointer(merged.Min),
		MaxLength:     int64(types.RemovePointer(merged.MaxLength)),
		MinLength:     int64(merged.MinLength),
		Pattern:       merged.Pattern,
		Format:        merged.Format,
		MaxItems:      int64(types.RemovePointer(merged.MaxItems)),
		MinItems:      int64(merged.MinItems),
		MaxProperties: int64(types.RemovePointer(merged.MaxProps)),
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
		if schema != nil && len(schema.Type.Slice()) == 0 {
			typ := types.TypeObject
			if len(schema.Enum) > 0 {
				enumType := GetOpenAPITypeFromValue(schema.Enum[0])
				if enumType != "" {
					typ = enumType
				}
			}
			schema.Type = &openapi3.Types{typ}
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
		impliedType = types.TypeObject
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
			typs := subSchema.Type.Slice()
			if len(typs) > 0 {
				impliedType = typs[0]
			}
			if impliedType == "" && subSchema.Items != nil && subSchema.Items.Value != nil {
				impliedType = types.TypeArray
			}
			if impliedType == "" {
				impliedType = types.TypeObject
			}
		}

		if impliedType == types.TypeObject {
			for propertyName, property := range subSchema.Properties {
				if subRef == "" {
					subRef = property.Ref
				}
				properties[propertyName] = property
			}
		}

		if impliedType == types.TypeArray && subSchema.Items != nil && subSchema.Items.Value != nil {
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
	required = types.SliceUnique(required)

	if not != nil {
		resultNot, _ := mergeKinSubSchemas(not.Value)
		if resultNot != nil {
			// not is always an object
			resultNot.Type = &openapi3.Types{types.TypeObject}
		}
		schema.Not = openapi3.NewSchemaRef("", resultNot)
	}

	schema.Type = &openapi3.Types{impliedType}
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
		has := types.RemovePointer(source.Has)
		if !has {
			return nil
		}
		// case when additionalProperties is true
		return &openapi3.Schema{
			Type: &openapi3.Types{types.TypeString},
		}
	}

	// we have schema
	return schemaRef.Value
}
