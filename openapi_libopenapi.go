package connexions

import (
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	"log"
	"os"
	"strconv"
	"strings"
)

// NewLibOpenAPIDocumentFromFile creates a new Document from a file path.
// It uses libopenapi to parse the file and then builds a model.
// Circular references are handled by logging the error and returning Document without errors.
func NewLibOpenAPIDocumentFromFile(filePath string) (Document, error) {
	src, _ := os.ReadFile(filePath)

	lib, err := libopenapi.NewDocument(src)
	if err != nil {
		return nil, err
	}

	if strings.HasPrefix(lib.GetVersion(), "2.") {
		model, errs := lib.BuildV2Model()
		if len(errs) > 0 {
			if model == nil {
				return nil, errs[0]
			}

			for err := range errs {
				log.Printf("Ignored error in %s: %v\n", filePath, err)
			}

			// if models is there we can ignore the errors
			return &LibV2Document{
				DocumentModel: model,
			}, nil
		}
		return &LibV2Document{
			DocumentModel: model,
		}, nil
	}

	model, errs := lib.BuildV3Model()
	if len(errs) > 0 {
		if model == nil {
			return nil, errs[0]
		}
		for err := range errs {
			log.Printf("Ignored error in %s: %v\n", filePath, err)
		}

		// if models is there we can ignore the errors
		return &LibV3Document{
			DocumentModel: model,
		}, nil
	}

	return &LibV3Document{
		DocumentModel: model,
	}, nil
}

// NewSchemaFromLibOpenAPI creates a new Schema from a libopenapi Schema.
func NewSchemaFromLibOpenAPI(schema *base.Schema, parseConfig *ParseConfig) *Schema {
	if parseConfig == nil {
		parseConfig = NewParseConfig()
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

	if GetSliceMaxRepetitionNumber(refPath) > parseConfig.MaxRecursionLevels {
		return nil
	}

	merged, mergedReference := mergeLibOpenAPISubSchemas(schema)

	typ := ""
	if len(merged.Type) > 0 {
		typ = merged.Type[0]
	}
	typ = FixSchemaTypeTypos(typ)

	var items *Schema
	if merged.Items != nil && merged.Items.IsA() {
		libItems := merged.Items.A
		sub := libItems.Schema()
		ref := libItems.GetReference()

		// detect circular reference early
		if parseConfig.MaxRecursionLevels == 0 && SliceContains(refPath, ref) {
			return nil
		}

		items = newSchemaFromLibOpenAPI(sub,
			parseConfig,
			AppendSliceFirstNonEmpty(refPath, ref, mergedReference),
			namePath)
	}

	var properties map[string]*Schema
	if len(merged.Properties) > 0 {
		properties = make(map[string]*Schema)
		for propName, sProxy := range merged.Properties {
			if parseConfig.OnlyRequired && !SliceContains(merged.Required, propName) {
				continue
			}
			properties[propName] = newSchemaFromLibOpenAPI(sProxy.Schema(),
				parseConfig,
				AppendSliceFirstNonEmpty(refPath, sProxy.GetReference(), mergedReference),
				append(namePath, propName))
		}
	}

	// add additional properties
	additionalProps := getLibAdditionalProperties(merged.AdditionalProperties)
	if additionalProps != nil {
		if properties == nil {
			properties = make(map[string]*Schema)
		}

		// TODO(cubahno): find out if this the correct property, or one from AdditionalProperties should be used
		minProperties := RemovePointer(merged.MinProperties)

		// TODO(cubahno): move to config
		additionalNum := 3
		if minProperties > 0 {
			additionalNum = int(minProperties)
		}

		additionalPrefix := "extra-"

		for i := 0; i < additionalNum; i++ {
			propName := additionalPrefix + strconv.Itoa(i+1)
			propSchema := newSchemaFromLibOpenAPI(
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
		not = newSchemaFromLibOpenAPI(merged.Not.Schema(), parseConfig, refPath, namePath)
	}

	if typ == TypeArray && items == nil {
		items = &Schema{Type: TypeString}
	}

	return &Schema{
		Type:          typ,
		Items:         items,
		MultipleOf:    RemovePointer(merged.MultipleOf),
		Maximum:       RemovePointer(merged.Maximum),
		Minimum:       RemovePointer(merged.Minimum),
		MaxLength:     RemovePointer(merged.MaxLength),
		MinLength:     RemovePointer(merged.MinLength),
		Pattern:       merged.Pattern,
		Format:        merged.Format,
		MaxItems:      RemovePointer(merged.MaxItems),
		MinItems:      RemovePointer(merged.MinItems),
		MaxProperties: RemovePointer(merged.MaxProperties),
		MinProperties: RemovePointer(merged.MinProperties),
		Required:      merged.Required,
		Enum:          merged.Enum,
		Properties:    properties,
		Not:           not,
		Default:       merged.Default,
		Nullable:      RemovePointer(merged.Nullable),
		ReadOnly:      merged.ReadOnly,
		WriteOnly:     merged.WriteOnly,
		Example:       merged.Example,
		Deprecated:    RemovePointer(merged.Deprecated),
	}
}

// mergeLibOpenAPISubSchemas merges allOf, anyOf, oneOf and not into a single schema.
func mergeLibOpenAPISubSchemas(schema *base.Schema) (*base.Schema, string) {
	allOf := schema.AllOf
	anyOf := schema.AnyOf
	oneOf := schema.OneOf
	not := schema.Not

	// base case: schema is flat
	if len(allOf) == 0 && len(anyOf) == 0 && len(oneOf) == 0 && not == nil {
		if schema != nil && len(schema.Type) == 0 {
			typ := TypeObject
			if len(schema.Enum) > 0 {
				enumType := GetOpenAPITypeFromValue(schema.Enum[0])
				if enumType != "" {
					typ = enumType
				}
			}
			schema.Type = []string{typ}
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

// getLibAdditionalProperties returns the additionalProperties of a libopenapi Schema.
func getLibAdditionalProperties(source any) *base.Schema {
	if source == nil {
		return nil
	}

	switch v := source.(type) {
	case bool:
		if !v {
			return nil
		}
		// default dictionary
		return &base.Schema{Type: []string{TypeString}}

	case *base.SchemaProxy:
		return v.Schema()
	}

	return nil
}
