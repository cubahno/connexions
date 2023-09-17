package connexions

import (
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/resolver"
	"log"
	"os"
	"strings"
)

func NewLibOpenAPIDocumentFromFile(filePath string) (Document, error) {
	src, _ := os.ReadFile(filePath)

	lib, err := libopenapi.NewDocument(src)
	if err != nil {
		return nil, err
	}

	if strings.HasPrefix(lib.GetVersion(), "2.") {
		model, errs := lib.BuildV2Model()
		if len(errs) > 0 {
			for i := range errs {
				if circError, ok := errs[i].(*resolver.ResolvingError); ok {
					if circError.CircularReference == nil {
						break
					}
					log.Printf("Message: %s\n--> Loop starts line %d | Polymorphic? %v\n\n",
						circError.Error(),
						circError.CircularReference.LoopPoint.Node.Line,
						circError.CircularReference.IsPolymorphicResult)
					return &LibV2Document{
						DocumentModel: model,
					}, nil
				}
			}
			return nil, errs[0]
		}
		return &LibV2Document{
			DocumentModel: model,
		}, nil
	}

	model, errs := lib.BuildV3Model()
	if len(errs) > 0 {
		for i := range errs {
			if circError, ok := errs[i].(*resolver.ResolvingError); ok {
				log.Printf("Message: %s\n", circError)
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

	var not *Schema
	if schema.Not != nil {
		not = newSchemaFromLibOpenAPI(schema.Not.Schema(), parseConfig, refPath, namePath)
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
		Not:           not,
		Default:       schema.Default,
		Nullable:      RemovePointer(schema.Nullable),
		ReadOnly:      schema.ReadOnly,
		WriteOnly:     schema.WriteOnly,
		Example:       schema.Example,
		Deprecated:    RemovePointer(schema.Deprecated),
	}
}

func mergeLibOpenAPISubSchemas(schema *base.Schema) (*base.Schema, string) {
	allOf := schema.AllOf
	anyOf := schema.AnyOf
	oneOf := schema.OneOf
	not := schema.Not

	// base case: schema is flat
	if len(allOf) == 0 && len(anyOf) == 0 && len(oneOf) == 0 && not == nil {
		if schema != nil && len(schema.Type) == 0 {
			schema.Type = []string{TypeObject}
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
		pickLibOpenAPISchemaProxy(anyOf),
		pickLibOpenAPISchemaProxy(oneOf),
	)

	subRef := ""
	for _, schemaRef := range allOf {
		if schemaRef == nil {
			continue
		}
		subSchema := schemaRef.Schema()

		if subRef == "" && schemaRef.IsReference() {
			subRef = schemaRef.GetReference()
		}

		if impliedType == "" {
			if len(subSchema.Type) > 0 {
				impliedType = subSchema.Type[0]
			}
			if impliedType == "" && subSchema.Items != nil && subSchema.Items.IsA() && len(subSchema.Items.A.Schema().Properties) > 0 {
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

		if impliedType == TypeArray && subSchema.Items != nil && subSchema.Items.IsA() {
			if subRef == "" {
				subRef = subSchema.Items.A.GetReference()
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
		resultNot, _ := mergeLibOpenAPISubSchemas(not.Schema())
		if resultNot != nil {
			// not is always an object
			resultNot.Type = []string{TypeObject}
		}
		schema.Not = base.CreateSchemaProxy(resultNot)
	}

	schema.Type = []string{impliedType}
	schema.Properties = properties
	schema.Required = required

	if len(schema.AllOf) > 0 {
		return mergeLibOpenAPISubSchemas(schema)
	}

	return schema, subRef
}

// pickLibOpenAPISchemaProxy returns the first non-nil schema proxy with reference
// or the first non-nil schema proxy if none of them have reference.
func pickLibOpenAPISchemaProxy(items []*base.SchemaProxy) *base.SchemaProxy {
	if len(items) == 0 {
		return nil
	}

	var fstNonEmpty *base.SchemaProxy

	for _, item := range items {
		if item == nil {
			continue
		}

		if fstNonEmpty == nil {
			fstNonEmpty = item
		}

		// prefer reference
		if item.GetReference() != "" {
			return item
		}
	}

	return fstNonEmpty
}
