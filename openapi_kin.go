package connexions

import (
	"github.com/getkin/kin-openapi/openapi3"
	"net/http"
)

type KinDocument struct {
	*openapi3.T
}

type KinOperation struct {
	*openapi3.Operation
	Operationer
}

type KinResponse struct {
	*openapi3.Response
}

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

func (d *KinDocument) GetVersion() string {
	if d.Info == nil {
		return ""
	}
	return d.Info.Version
}

func (d *KinDocument) GetResources() map[string][]string {
	res := make(map[string][]string)
	for resName, pathItem := range d.Paths {
		res[resName] = make([]string, 0)
		for method, _ := range pathItem.Operations() {
			res[resName] = append(res[resName], method)
		}
	}
	return res
}

func (d *KinDocument) FindOperation(resourceName, method string) Operationer {
	path := d.Paths.Find(resourceName)
	if path == nil {
		return nil
	}
	op := path.GetOperation(method)
	if op == nil {
		return nil
	}
	return &KinOperation{Operation: op}
}

func (o *KinOperation) GetParameters() OpenAPIParameters {
	var res []*OpenAPIParameter
	for _, param := range o.Parameters {
		p := param.Value
		if p == nil {
			continue
		}

		var schema *Schema
		if p.Schema != nil {
			schema = NewSchemaFromKin(p.Schema.Value, nil)
		}

		res = append(res, &OpenAPIParameter{
			Name:     p.Name,
			In:       p.In,
			Required: p.Required,
			Schema:   schema,
		})
	}
	return res
}

func (o *KinOperation) GetRequestBody() (*Schema, string) {
	if o.RequestBody == nil {
		return nil, ""
	}

	schema := o.RequestBody.Value
	contentTypes := schema.Content
	if len(contentTypes) == 0 {
		return nil, ""
	}

	typesOrder := []string{"application/json", "multipart/form-data", "application/x-www-form-urlencoded"}
	for _, contentType := range typesOrder {
		if _, ok := contentTypes[contentType]; ok {
			return NewSchemaFromKin(contentTypes[contentType].Schema.Value, nil), contentType
		}
	}

	// Get first defined
	for contentType, mediaType := range contentTypes {
		return NewSchemaFromKin(mediaType.Schema.Value, nil), contentType
	}

	return nil, ""
}

func (o *KinOperation) GetResponse() (OpenAPIResponse, int) {
	available := o.Responses

	var responseRef *openapi3.ResponseRef
	for _, code := range []int{http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent} {
		responseRef = available.Get(code)
		if responseRef != nil {
			return &KinResponse{responseRef.Value}, code
		}
	}

	// Get first defined
	for codeName, respRef := range available {
		if codeName == "default" {
			continue
		}
		return &KinResponse{respRef.Value}, TransformHTTPCode(codeName)
	}

	return &KinResponse{available.Default().Value}, 200
}

func (r *KinResponse) GetContent() (string, *Schema) {
	types := r.Content
	if len(types) == 0 {
		return "", nil
	}

	prioTypes := []string{"application/json", "text/plain", "text/html"}
	for _, contentType := range prioTypes {
		if _, ok := types[contentType]; ok {
			return contentType, NewSchemaFromKin(types[contentType].Schema.Value, nil)
		}
	}

	for contentType, mediaType := range types {
		return contentType, NewSchemaFromKin(mediaType.Schema.Value, nil)
	}

	return "", nil
}

func (r *KinResponse) GetHeaders() OpenAPIHeaders {
	res := make(OpenAPIHeaders)
	for name, header := range r.Headers {
		ref := header.Value
		if ref == nil {
			continue
		}

		p := ref.Parameter
		var schema *Schema
		if p.Schema != nil && p.Schema.Value != nil {
			schema = NewSchemaFromKin(p.Schema.Value, nil)
		}

		res[name] = &OpenAPIParameter{
			Name:     p.Name,
			In:       p.In,
			Required: p.Required,
			Schema:   schema,
		}
	}
	return res
}

func NewSchemaFromKin(s *openapi3.Schema, visited map[string]bool) *Schema {
	if s == nil {
		return nil
	}

	s = MergeKinSubSchemas(s)

	if len(visited) == 0 {
		visited = make(map[string]bool)
	}

	var items *SchemaWithReference
	if s.Items != nil && s.Items.Value != nil {
		if s.Items.Ref != "" {
			if visited[s.Items.Ref] {
				return nil
			}

			visited[s.Items.Ref] = true
		}
		items = &SchemaWithReference{
			Schema:    NewSchemaFromKin(s.Items.Value, visited),
			Reference: s.Items.Ref,
		}
	}

	var properties map[string]*SchemaWithReference
	if len(s.Properties) > 0 {
		properties = make(map[string]*SchemaWithReference)
		for name, ref := range s.Properties {
			t := visited
			if ref.Ref != "" && visited[ref.Ref] {
				continue
			}

			if ref.Ref != "" {
				visited[ref.Ref] = true
			}

			properties[name] = &SchemaWithReference{
				Schema:    NewSchemaFromKin(ref.Value, t),
				Reference: ref.Ref,
			}
		}
	}

	return &Schema{
		Type:          s.Type,
		Items:         items,
		MultipleOf:    RemovePointer(s.MultipleOf),
		Maximum:       RemovePointer(s.Max),
		Minimum:       RemovePointer(s.Min),
		MaxLength:     int64(RemovePointer(s.MaxLength)),
		MinLength:     int64(s.MinLength),
		Pattern:       s.Pattern,
		Format:        s.Format,
		MaxItems:      int64(RemovePointer(s.MaxItems)),
		MinItems:      int64(s.MinItems),
		MaxProperties: int64(RemovePointer(s.MaxProps)),
		MinProperties: int64(s.MinProps),
		Required:      s.Required,
		Enum:          s.Enum,
		Properties:    properties,
		Default:       s.Default,
		Nullable:      s.Nullable,
		ReadOnly:      s.ReadOnly,
		WriteOnly:     s.WriteOnly,
		Example:       s.Example,
		Deprecated:    s.Deprecated,
	}
}

func MergeKinSubSchemas(schema *openapi3.Schema) *openapi3.Schema {
	allOf := schema.AllOf
	anyOf := schema.AnyOf
	oneOf := schema.OneOf
	not := schema.Not

	if len(schema.Properties) == 0 {
		schema.Properties = make(map[string]*openapi3.SchemaRef)
	}

	schema.AllOf = make([]*openapi3.SchemaRef, 0)
	schema.AnyOf = make([]*openapi3.SchemaRef, 0)
	schema.OneOf = make([]*openapi3.SchemaRef, 0)
	schema.Not = nil

	for _, schemaRef := range allOf {
		sub := schemaRef.Value
		if sub == nil {
			continue
		}

		for propertyName, property := range sub.Properties {
			schema.Properties[propertyName] = property
		}

		schema.AllOf = append(schema.AllOf, sub.AllOf...)
		schema.AnyOf = append(schema.AnyOf, sub.AnyOf...)
		schema.OneOf = append(schema.OneOf, sub.OneOf...)
		schema.Required = append(schema.Required, sub.Required...)
	}

	// pick first from each if present
	schemaRefs := [][]*openapi3.SchemaRef{anyOf, oneOf}
	for _, schemaRef := range schemaRefs {
		if len(schemaRef) == 0 {
			continue
		}

		sub := schemaRef[0].Value
		if sub == nil {
			continue
		}

		for propertyName, property := range sub.Properties {
			schema.Properties[propertyName] = property
		}

		schema.AllOf = append(schema.AllOf, sub.AllOf...)
		schema.AnyOf = append(schema.AnyOf, sub.AnyOf...)
		schema.OneOf = append(schema.OneOf, sub.OneOf...)
		schema.Required = append(schema.Required, sub.Required...)
	}

	// exclude properties from `not`
	if not != nil && not.Value != nil {
		notSchema := not.Value
		deletes := map[string]bool{}
		for propertyName, _ := range notSchema.Properties {
			delete(schema.Properties, propertyName)
			deletes[propertyName] = true
		}

		// remove from required properties
		for i, propertyName := range schema.Required {
			if _, ok := deletes[propertyName]; ok {
				schema.Required = append(schema.Required[:i], schema.Required[i+1:]...)
			}
		}
	}

	if len(schema.AllOf) > 0 || len(schema.AnyOf) > 0 || len(schema.OneOf) > 0 {
		return MergeKinSubSchemas(schema)
	}

	if schema.Type == "" {
		schema.Type = "object"
	}

	return schema
}
