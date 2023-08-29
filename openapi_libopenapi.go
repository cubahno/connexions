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
	"strings"
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
}

type LibV3Response struct {
	*v3high.Response
	ParseConfig *ParseConfig
}

func NewLibOpenAPIDocumentFromFile(filePath string, parseConfig *ParseConfig) (Document, error) {
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
					ParseConfig:   parseConfig,
				}, nil
			}
		}
		return nil, errs[0]
	}

	return &LibV3Document{
		DocumentModel: model,
		ParseConfig:   parseConfig,
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

func (d *LibV3Document) FindOperation(resourceName, method string) Operationer {
	path, ok := d.Model.Paths.PathItems[resourceName]
	if !ok {
		return nil
	}

	for m, op := range path.GetOperations() {
		if strings.ToUpper(m) == strings.ToUpper(method) {
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

	return params
}

func (op *LibV3Operation) GetResponse() (OpenAPIResponse, int) {
	available := op.Responses.Codes

	var responseRef *v3high.Response
	for _, code := range []int{http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent} {
		responseRef = available[fmt.Sprintf("%v", code)]
		if responseRef != nil {
			return &LibV3Response{
				Response:    responseRef,
				ParseConfig: op.ParseConfig,
			}, code
		}
	}

	// Get first defined
	for codeName, respRef := range available {
		if codeName == "default" {
			continue
		}
		return &LibV3Response{
			Response:    respRef,
			ParseConfig: op.ParseConfig,
		}, TransformHTTPCode(codeName)
	}

	return &LibV3Response{
		Response:    op.Responses.Default,
		ParseConfig: op.ParseConfig,
	}, 200
}

func (op *LibV3Operation) GetRequestBody() (*Schema, string) {
	if op.RequestBody == nil {
		return nil, ""
	}

	contentTypes := op.RequestBody.Content
	if len(contentTypes) == 0 {
		return nil, ""
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

func (r *LibV3Response) GetContent() (*Schema, string) {
	types := r.Content
	if len(types) == 0 {
		return nil, ""
	}

	prioTypes := []string{"application/json", "text/plain", "text/html"}
	for _, contentType := range prioTypes {
		if _, ok := types[contentType]; ok {
			px := types[contentType].Schema
			return NewSchemaFromLibOpenAPI(px.Schema(), r.ParseConfig), contentType
		}
	}

	for contentType, mediaType := range types {
		px := mediaType.Schema
		return NewSchemaFromLibOpenAPI(px.Schema(), r.ParseConfig), contentType
	}

	return nil, ""
}

func (r *LibV3Response) GetHeaders() OpenAPIHeaders {
	res := make(OpenAPIHeaders)
	for name, header := range r.Headers {
		if header == nil {
			continue
		}

		var schema *Schema
		if header.Schema != nil {
			px := header.Schema
			schema = NewSchemaFromLibOpenAPI(px.Schema(), r.ParseConfig)
		}

		res[name] = &OpenAPIParameter{
			Name:     name,
			In:       ParameterInHeader,
			Required: header.Required,
			Schema:   schema,
		}
	}
	return res
}

func NewSchemaFromLibOpenAPI(schema *base.Schema, parseConfig *ParseConfig) *Schema {
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

	if parseConfig != nil && parseConfig.MaxLevels > 0 && len(namePath) > parseConfig.MaxLevels {
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
	}

	// exclude properties from `not`
	if not != nil {
		notSchema := not.Schema()
		deletes := map[string]bool{}
		for propertyName, _ := range notSchema.Properties {
			delete(properties, propertyName)
			deletes[propertyName] = true
		}
	}

	schema.Properties = properties
	schema.Type = []string{impliedType}

	return schema, subRef
}
