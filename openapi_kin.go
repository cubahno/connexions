package connexions

import (
	"github.com/getkin/kin-openapi/openapi3"
	"net/http"
)

type KinDocument struct {
	*openapi3.T
	ParseConfig *ParseConfig
}

type KinOperation struct {
	*openapi3.Operation
	Operationer
	ParseConfig *ParseConfig
}

type KinResponse struct {
	*openapi3.Response
	ParseConfig *ParseConfig
}

func NewKinDocumentFromFile(filePath string, parseConfig *ParseConfig) (Document, error) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(filePath)
	if err != nil {
		return nil, err
	}
	return &KinDocument{
		T:           doc,
		ParseConfig: parseConfig,
	}, err
}

func (d *KinDocument) GetVersion() string {
	return d.OpenAPI
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
	return &KinOperation{
		Operation:   op,
		ParseConfig: d.ParseConfig,
	}
}

func (op *KinOperation) GetParameters() OpenAPIParameters {
	var res []*OpenAPIParameter
	for _, param := range op.Parameters {
		p := param.Value
		if p == nil {
			continue
		}

		var schema *Schema
		if p.Schema != nil {
			schema = NewSchemaFromKin(p.Schema.Value, op.ParseConfig)
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
			return NewSchemaFromKin(contentTypes[contentType].Schema.Value, op.ParseConfig), contentType
		}
	}

	// Get first defined
	for contentType, mediaType := range contentTypes {
		return NewSchemaFromKin(mediaType.Schema.Value, op.ParseConfig), contentType
	}

	return nil, ""
}

func (op *KinOperation) GetResponse() (OpenAPIResponse, int) {
	available := op.Responses

	var responseRef *openapi3.ResponseRef
	for _, code := range []int{http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent} {
		responseRef = available.Get(code)
		if responseRef != nil {
			return &KinResponse{
				Response:    responseRef.Value,
				ParseConfig: op.ParseConfig,
			}, code
		}
	}

	// Get first defined
	for codeName, respRef := range available {
		if codeName == "default" {
			continue
		}
		return &KinResponse{
			Response:    respRef.Value,
			ParseConfig: op.ParseConfig,
		}, TransformHTTPCode(codeName)
	}

	return &KinResponse{
		Response:    available.Default().Value,
		ParseConfig: op.ParseConfig,
	}, 200
}

func (r *KinResponse) GetContent() (*Schema, string) {
	types := r.Content
	if len(types) == 0 {
		return nil, ""
	}

	prioTypes := []string{"application/json", "text/plain", "text/html"}
	for _, contentType := range prioTypes {
		if _, ok := types[contentType]; ok {
			return NewSchemaFromKin(types[contentType].Schema.Value, r.ParseConfig), contentType
		}
	}

	for contentType, mediaType := range types {
		return NewSchemaFromKin(mediaType.Schema.Value, r.ParseConfig), contentType
	}

	return nil, ""
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
			schema = NewSchemaFromKin(p.Schema.Value, r.ParseConfig)
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

func NewSchemaFromKin(s *openapi3.Schema, parseConfig *ParseConfig) *Schema {
	return newSchemaFromKin(s, nil)
}

func newSchemaFromKin(s *openapi3.Schema, visited map[string]bool) *Schema {
	if s == nil {
		return nil
	}

	s = MergeKinSubSchemas(s)

	if len(visited) == 0 {
		visited = make(map[string]bool)
	}

	var items *Schema
	if s.Items != nil && s.Items.Value != nil {
		if s.Items.Ref != "" {
			if visited[s.Items.Ref] {
				return nil
			}

			visited[s.Items.Ref] = true
		}
		items = newSchemaFromKin(s.Items.Value, visited)
	}

	var properties map[string]*Schema
	if len(s.Properties) > 0 {
		properties = make(map[string]*Schema)
		for name, ref := range s.Properties {
			t := visited
			if ref.Ref != "" && visited[ref.Ref] {
				continue
			}

			if ref.Ref != "" {
				visited[ref.Ref] = true
			}

			properties[name] = newSchemaFromKin(ref.Value, t)
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
	not := schema.Not

	if len(schema.Properties) == 0 {
		schema.Properties = make(map[string]*openapi3.SchemaRef)
	}

	// reset
	schema.AllOf = make([]*openapi3.SchemaRef, 0)
	schema.Not = nil

	for _, schemaRef := range allOf {
		sub := schemaRef.Value
		if sub == nil {
			continue
		}

		for propertyName, property := range sub.Properties {
			schema.Properties[propertyName] = property
		}

		// gather fom the sub
		schema.AllOf = append(schema.AllOf, sub.AllOf...)
		schema.AnyOf = append(schema.AnyOf, sub.AnyOf...)
		schema.OneOf = append(schema.OneOf, sub.OneOf...)
		schema.Required = append(schema.Required, sub.Required...)
	}

	// pick first from each if present
	var either []*openapi3.SchemaRef
	if len(schema.AnyOf) > 0 {
		either = append(either, schema.AnyOf[0])
	}
	if len(schema.OneOf) > 0 {
		either = append(either, schema.OneOf[0])
	}

	// reset
	schema.AnyOf = make([]*openapi3.SchemaRef, 0)
	schema.OneOf = make([]*openapi3.SchemaRef, 0)

	for _, schemaRef := range either {
		if schemaRef == nil {
			continue
		}

		sub := schemaRef.Value
		if sub == nil {
			continue
		}

		for propertyName, property := range sub.Properties {
			schema.Properties[propertyName] = property
		}

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

	if len(schema.AllOf) > 0 {
		return MergeKinSubSchemas(schema)
	}

	if schema.Type == "" {
		schema.Type = "object"
	}

	return schema
}
