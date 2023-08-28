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
}

type LibV3Operation struct {
	*v3high.Operation
}

type LibV3Response struct {
	*v3high.Response
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
				return &LibV3Document{model}, nil
			}
		}
		return nil, errs[0]
	}

	return &LibV3Document{model}, nil
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
			return &LibV3Operation{op}
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
			schema = NewSchemaFromLibOpenAPI(NormalizeLibOpenAPISchema(px.Schema(), nil))
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
			return &LibV3Response{responseRef}, code
		}
	}

	// Get first defined
	for codeName, respRef := range available {
		if codeName == "default" {
			continue
		}
		return &LibV3Response{respRef}, TransformHTTPCode(codeName)
	}

	return &LibV3Response{op.Responses.Default}, 200
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
			return NewSchemaFromLibOpenAPI(NormalizeLibOpenAPISchema(px.Schema(), nil)), contentType
		}
	}

	// Get first defined
	for contentType, mediaType := range contentTypes {
		px := mediaType.Schema
		return NewSchemaFromLibOpenAPI(NormalizeLibOpenAPISchema(px.Schema(), nil)), contentType
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
			return NewSchemaFromLibOpenAPI(NormalizeLibOpenAPISchema(px.Schema(), nil)), contentType
		}
	}

	for contentType, mediaType := range types {
		px := mediaType.Schema
		return NewSchemaFromLibOpenAPI(NormalizeLibOpenAPISchema(px.Schema(), nil)), contentType
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
			schema = NewSchemaFromLibOpenAPI(NormalizeLibOpenAPISchema(px.Schema(), nil))
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

func NewSchemaFromLibOpenAPI(s *base.Schema) *Schema {
	if s == nil {
		return nil
	}

	var items *Schema
	if s.Items != nil && s.Items.IsA() {
		libItems := s.Items.A
		items = NewSchemaFromLibOpenAPI(libItems.Schema())
	}

	var properties map[string]*Schema
	if len(s.Properties) > 0 {
		properties = make(map[string]*Schema)
		for propName, sProxy := range s.Properties {
			properties[propName] = NewSchemaFromLibOpenAPI(sProxy.Schema())
		}
	}

	typ := ""
	if len(s.Type) > 0 {
		typ = s.Type[0]
	}

	return &Schema{
		Type:          typ,
		Items:         items,
		MultipleOf:    RemovePointer(s.MultipleOf),
		Maximum:       RemovePointer(s.Maximum),
		Minimum:       RemovePointer(s.Minimum),
		MaxLength:     RemovePointer(s.MaxLength),
		MinLength:     RemovePointer(s.MinLength),
		Pattern:       s.Pattern,
		Format:        s.Format,
		MaxItems:      RemovePointer(s.MaxItems),
		MinItems:      RemovePointer(s.MinItems),
		MaxProperties: RemovePointer(s.MaxProperties),
		MinProperties: RemovePointer(s.MinProperties),
		Required:      s.Required,
		Enum:          s.Enum,
		Properties:    properties,
		Default:       s.Default,
		Nullable:      RemovePointer(s.Nullable),
		ReadOnly:      s.ReadOnly,
		WriteOnly:     s.WriteOnly,
		Example:       s.Example,
		Deprecated:    RemovePointer(s.Deprecated),
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

func NormalizeLibOpenAPISchema(schema *base.Schema, path []string) *base.Schema {
	if schema == nil || isPathRepeated(path) {
		return &base.Schema{Title: "circular-reference"}
	}
	if len(path) == 0 {
		path = make([]string, 0)
	}

	schema = mergeLibOpenAPISubSchemas(schema)

	typ := ""
	if len(schema.Type) > 0 {
		typ = schema.Type[0]
	}
	if typ == "" {
		typ = TypeObject
	}

	if typ != TypeArray && typ != TypeObject {
		return schema
	}

	properties := make(map[string]*base.SchemaProxy)
	for name, property := range schema.Properties {
		p := path
		if property == nil {
			continue
		}
		propSchema := property.Schema()

		// TODO(igor): fix in libopenapi or find a way to get unique reference
		// needed  to avoid circular references
		propRef := name
		if property.IsReference() {
			propRef = property.GetReference()
		}

		if propRef != "" {
			p = append(p, propRef)
		}
		rv := NormalizeLibOpenAPISchema(propSchema, p)
		if rv == nil || rv.Title == "circular-reference" {
			continue
		}

		sp := base.CreateSchemaProxy(rv)
		properties[name] = sp
	}
	if typ == TypeObject {
		schema.Properties = properties
	}

	if schema.Items != nil && schema.Items.IsA() {
		items := schema.Items.A
		rv := NormalizeLibOpenAPISchema(items.Schema(), path)
		if rv == nil || rv.Title == "circular-reference" {
			return nil
		}
		schema.Items.A = base.CreateSchemaProxy(rv)
		schema.Type = []string{TypeArray}
	}

	return schema
}

func mergeLibOpenAPISubSchemas(schema *base.Schema) *base.Schema {
	allOf := schema.AllOf
	anyOf := schema.AnyOf
	oneOf := schema.OneOf
	not := schema.Not

	// base case: schema is flat
	if len(allOf) == 0 && len(anyOf) == 0 && len(oneOf) == 0 && not == nil {
		return schema
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

	for _, schemaRef := range allOf {
		if schemaRef == nil {
			continue
		}

		subSchema := schemaRef.Schema()
		if subSchema == nil || subSchema.Title == "circular-reference" {
			continue
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
				properties[propertyName] = property
			}
		}

		if impliedType == TypeArray && subSchema.Items.IsA() {
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

	return schema
}

func isPathRepeated[T comparable](path []T) bool {
	// detect circular references
	visited := make(map[T]bool)
	for _, item := range path {
		if _, ok := visited[item]; ok {
			return true
		}
		visited[item] = true
	}
	return false
}
