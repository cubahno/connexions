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
			schema = NewSchemaFromLibOpenAPI(px.Schema(), nil)
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
			return NewSchemaFromLibOpenAPI(px.Schema(), nil), contentType
		}
	}

	// Get first defined
	for contentType, mediaType := range contentTypes {
		px := mediaType.Schema
		return NewSchemaFromLibOpenAPI(px.Schema(), nil), contentType
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
			return NewSchemaFromLibOpenAPI(px.Schema(), nil), contentType
		}
	}

	for contentType, mediaType := range types {
		px := mediaType.Schema
		return NewSchemaFromLibOpenAPI(px.Schema(), nil), contentType
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
			schema = NewSchemaFromLibOpenAPI(px.Schema(), nil)
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

func NewSchemaFromLibOpenAPI(s *base.Schema, path []string) *Schema {
	if s == nil {
		return nil
	}

	if len(path) == 0 {
		path = make([]string, 0)
	}

	//s = MergeLibOpenAPISubSchemas(s, path)

	var items *SchemaWithReference
	if s.Items != nil && s.Items.IsA() {
		libItems := s.Items.A

		ref := libItems.GetReference()
		if ref != "" {
			for _, pathItem := range path {
				if pathItem == ref {
					println("circular reference from array:", ref)
					return nil
				}
			}
			path = append(path, ref)
		}

		sub := NewSchemaFromLibOpenAPI(libItems.Schema(), path)
		if sub == nil {
			// return nil
		}
		items = &SchemaWithReference{
			Schema:    sub,
			Reference: ref,
		}
	}

	var properties map[string]*SchemaWithReference
	if len(s.Properties) > 0 {
		properties = make(map[string]*SchemaWithReference)
		for propName, sProxy := range s.Properties {
			t := path
			ref := sProxy.GetReference()
			if ref != "" {
				for _, pathItem := range path {
					if pathItem == ref {
						println("circular reference from object:", ref)
						return nil
					}
				}
				t = append(t, ref)
			}

			sub := NewSchemaFromLibOpenAPI(sProxy.Schema(), t)
			if sub == nil {
				continue
			}
			properties[propName] = &SchemaWithReference{
				Schema:    sub,
				Reference: ref,
			}
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

func MergeLibOpenAPISubSchemas2(schema *base.Schema, path []string) *base.Schema {
	if len(path) == 0 {
		path = make([]string, 0)
	}

	// detect circular references
	visited := make(map[string]bool)
	for _, item := range path {
		if _, ok := visited[item]; ok {
			return nil
		}
		visited[item] = true
	}

	allOf := schema.AllOf
	anyOf := schema.AnyOf
	oneOf := schema.OneOf
	not := schema.Not

	for propertyName, property := range schema.Properties {
		resolved := MergeLibOpenAPISubSchemas2(property.Schema(), append(path, propertyName))
		if resolved == nil {
			delete(schema.Properties, propertyName)
			continue
		}

		objProperties := make(map[string]*base.SchemaProxy)
		if len(resolved.Properties) > 0 {
			for resName, resProperty := range resolved.Properties {
				objProperties[resName] = resProperty
			}
			schema.Properties[propertyName] = base.CreateSchemaProxy(&base.Schema{
				Type:       []string{"object"},
				Properties: objProperties,
			})
		}
	}

	if schema.Items != nil && schema.Items.IsA() {
		items := schema.Items.A
		itemsSchema := items.Schema()

		// set initial properties
		itemProperties := make(map[string]*base.SchemaProxy)
		for currentName, currentProperty := range itemsSchema.Properties {

			resolved := MergeLibOpenAPISubSchemas2(currentProperty.Schema(), append(path, currentName))
			if resolved == nil {
				delete(itemsSchema.Properties, currentName)
				continue
			}
			resProperties := make(map[string]*base.SchemaProxy)
			for resName, resProperty := range resolved.Properties {
				resProperties[resName] = resProperty
			}

			itemProperties[currentName] = base.CreateSchemaProxy(&base.Schema{
				Type:       currentProperty.Schema().Type,
				Properties: resProperties,
				Items:      resolved.Items,
			})
		}

		resolved := MergeLibOpenAPISubSchemas2(itemsSchema, path)
		for resName, resProperty := range resolved.Properties {
			itemProperties[resName] = resProperty
		}
		schema.Items = &base.DynamicValue[*base.SchemaProxy, bool]{
			A: base.CreateSchemaProxy(&base.Schema{
				Type:       []string{"object"},
				Properties: itemProperties,
			}),
		}
	}

	// schema is flat
	if len(allOf) == 0 && len(anyOf) == 0 && len(oneOf) == 0 && not == nil {
		return schema
	}

	// can contain only references or schema objects,
	// not primitive types like strings, numbers, or booleans
	properties := make(map[string]*base.SchemaProxy)
	var items *base.DynamicValue[*base.SchemaProxy, bool]
	var typ []string

	if len(allOf) > 0 {
		typ = []string{"object"}
	}

	for _, schemaRef := range allOf {
		// take each item and resolve it
		sub := schemaRef.Schema()
		if sub == nil {
			continue
		}

		for propertyName, property := range sub.Properties {
			resolved := MergeLibOpenAPISubSchemas2(property.Schema(), append(path, propertyName))
			if resolved == nil {
				continue
			}
			if resolved == nil {
				// delete(schema.Properties, propertyName)
				continue
			}

			objProperties := make(map[string]*base.SchemaProxy)
			if len(resolved.Properties) > 0 {
				for resName, resProperty := range resolved.Properties {
					objProperties[resName] = resProperty
				}
			}
			properties[propertyName] = base.CreateSchemaProxy(&base.Schema{
				Type:       resolved.Type,
				Properties: objProperties,
			})
		}

		// resolved := MergeLibOpenAPISubSchemas(sub, nil)
		// if resolved == nil {
		// 	continue
		// }

		// // reference can only be an object in allOf
		// // we're composing new properties
		// for propertyName, property := range resolved.Properties {
		// 	properties[propertyName] = property
		// }

		// if sub.Items != nil && sub.Items.IsA() {
		// 	subPx := sub.Items.A
		// 	ref := subPx.GetReference()
		// 	subSchema := subPx.Schema()
		// 	schema.Items
		// }

		// gather from the sub
		// schema.AllOf = append(schema.AllOf, sub.AllOf...)
		// schema.AnyOf = append(schema.AnyOf, sub.AnyOf...)
		// schema.OneOf = append(schema.OneOf, sub.OneOf...)
		// schema.Required = append(schema.Required, sub.Required...)
	}

	// pick one from each
	// either := []*base.SchemaProxy{
	// 	PicklLibOpenAPISchemaProxy(schema.AnyOf),
	// 	PicklLibOpenAPISchemaProxy(schema.OneOf),
	// }
	// reset
	//schema.AnyOf = make([]*base.SchemaProxy, 0)
	//schema.OneOf = make([]*base.SchemaProxy, 0)

	// for _, schemaRef := range either {
	// 	if schemaRef == nil {
	// 		continue
	// 	}
	//
	// 	sub := schemaRef.Schema()
	// 	if sub == nil {
	// 		continue
	// 	}
	//
	// 	// ...
	// 	for propertyName, property := range sub.Properties {
	// 		schema.Properties[propertyName] = property
	// 	}
	//
	// 	schema.Required = append(schema.Required, sub.Required...)
	// }
	//
	// // exclude properties from `not`
	// if not != nil {
	// 	notSchema := not.Schema()
	// 	deletes := map[string]bool{}
	// 	for propertyName, _ := range notSchema.Properties {
	// 		delete(schema.Properties, propertyName)
	// 		deletes[propertyName] = true
	// 	}
	//
	// 	// remove from required properties
	// 	for i, propertyName := range schema.Required {
	// 		if _, ok := deletes[propertyName]; ok {
	// 			schema.Required = append(schema.Required[:i], schema.Required[i+1:]...)
	// 		}
	// 	}
	// }

	// if len(schema.AllOf) > 0 {
	// 	return MergeLibOpenAPISubSchemas(schema, path)
	// }

	return &base.Schema{
		Type:       typ,
		Properties: properties,
		Items: items,
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

func NormalizeLibOpenAPISchema(schema *base.Schema, path []string) *Schema {
	return nil
}

func mergeLibOpenAPISubSchemas(schemaProxy *base.SchemaProxy, path []string) *base.SchemaProxy {
	if len(path) == 0 {
		path = make([]string, 0)
	}

	if isPathRepeated(path) {
		return nil
	}

	schema := schemaProxy.Schema()

	allOf := schema.AllOf
	anyOf := schema.AnyOf
	oneOf := schema.OneOf
	not := schema.Not

	// base case: schema is flat
	if len(allOf) == 0 && len(anyOf) == 0 && len(oneOf) == 0 && not == nil {
		return schemaProxy
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

	for _, schemaRef := range allOf {
		if schemaRef == nil {
			continue
		}
		// it might have it deep-nested. resolve 'em all here.
		// sub := mergeLibOpenAPISubSchemas(schemaRef, path)
		// if sub == nil {
		// 	continue
		// }
		for propertyName, property := range schemaRef.Schema().Properties {
			properties[propertyName] = mergeLibOpenAPISubSchemas(property, append(path, propertyName))
		}
	}

	schema.Properties = properties
	schema.Type = []string{TypeObject}

	return base.CreateSchemaProxy(schema)
}

func collectLibObjects(schemaProxy *base.SchemaProxy, path []string) *base.SchemaProxy {
	if schemaProxy == nil || isPathRepeated(path) {
		return nil
	}

	schemaProxy = mergeLibOpenAPISubSchemas(schemaProxy, path)

	schema := schemaProxy.Schema()
	typ := ""
	if len(schema.Type) > 0 {
		typ = schema.Type[0]
	}

	if typ == TypeArray {
		return collectLibArrays(schemaProxy, path)
	}

	if typ != TypeObject {
		merged := mergeLibOpenAPISubSchemas(schemaProxy, path)
		return merged
	}

	properties := make(map[string]*base.SchemaProxy)
	for name, property := range schema.Properties {
		p := path
		ref := property.GetReference()
		if ref != "" {
			p = append(p, ref)
		}
		rv := collectLibObjects(property, p)
		if rv == nil {
			continue
		}
		// flat allOf, ...
		flatted := mergeLibOpenAPISubSchemas(rv, p)
		properties[name] = flatted
	}

	schema.Properties = properties

	return base.CreateSchemaProxy(schema)
}

func collectLibArrays(schemaProxy *base.SchemaProxy, path []string) *base.SchemaProxy {
	if schemaProxy == nil || isPathRepeated(path) {
		return nil
	}

	schema := schemaProxy.Schema()
	ref := schemaProxy.GetReference()
	// ref := ""
	if ref != "" {
		path = append(path, schemaProxy.GetReference())
	}

	typ := ""
	if len(schema.Type) > 0 {
		typ = schema.Type[0]
	}

	if typ == TypeObject {
		return collectLibObjects(schemaProxy, path)
	}

	if typ != TypeArray {
		merged := mergeLibOpenAPISubSchemas(schemaProxy, path)
		return merged
	}

	if schema.Items != nil && !schema.Items.IsA() {
		return nil
	}

	items := schema.Items.A
	rv := collectLibArrays(items, path)
	if rv == nil {
		return nil
	}

	rv2 := mergeLibOpenAPISubSchemas(rv, path)


	schema.Items.A = rv2

	// set initial properties
	// itemProperties := make(map[string]*base.SchemaProxy)
	// for currentName, currentProperty := range itemsSchema.Properties {
	// 	resolved := collectLibArrays(currentProperty)
	// 	itemProperties[currentName] = resolved
	// }



	// return base.CreateSchemaProxy(&base.Schema{
	// 	Type:       []string{TypeArray},
	// 	Items: &base.DynamicValue[*base.SchemaProxy, bool]{
	// 		A: rr,
	// 	},
	// })
	return base.CreateSchemaProxy(schema)
}

func isPathRepeated(path []string) bool {
	// detect circular references
	visited := make(map[string]bool)
	for _, item := range path {
		if _, ok := visited[item]; ok {
			return true
		}
		visited[item] = true
	}
	return false
}
