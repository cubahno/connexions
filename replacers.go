package xs

import (
	"log"
	"strings"
	"time"
)

type Replacer func(ctx *ReplaceContext) any
type Any interface {
	string | int | bool | float64 | any
}

// NULL is used to force resolve to None
const (
	NULL   = "__null__"
	NONAME = "__noname__"
)

func HasCorrectSchemaType(ctx *ReplaceContext, value any) bool {
	schema, ok := ctx.Schema.(*Schema)
	if !ok {
		// TODO(igor): check how to handle with other content schemas
		return true
	}
	return IsCorrectlyReplacedType(value, schema.Type)
}

func ReplaceInHeaders(ctx *ReplaceContext) any {
	if !ctx.State.IsHeader {
		return nil
	}
	// TODO(igor): implement it
	return nil
}

func ReplaceFromContext(ctx *ReplaceContext) any {
	for _, replacements := range ctx.Resource.ContextData {
		if res := ReplaceValueWithContext(ctx.State.NamePath, replacements); res != nil {
			return res
		}
	}

	return nil
}

func ReplaceValueWithContext(path []string, contextData any) interface{} {
	switch valueType := contextData.(type) {
	case map[string]string:
		return ReplaceValueWithMapContext[string](path, valueType)
	case map[string]int:
		return ReplaceValueWithMapContext[int](path, valueType)
	case map[string]bool:
		return ReplaceValueWithMapContext[bool](path, valueType)
	case map[string]float64:
		return ReplaceValueWithMapContext[float64](path, valueType)
	case map[string]any:
		return ReplaceValueWithMapContext[any](path, valueType)

	// base cases below:
	case FakeFunc:
		return valueType().Get()
	case string, int, bool, float64:
		return valueType
	case []string:
		return GetRandomSliceValue(valueType)
	case []int:
		return GetRandomSliceValue(valueType)
	case []bool:
		return GetRandomSliceValue(valueType)
	case []float64:
		return GetRandomSliceValue(valueType)
	case []any:
		return GetRandomSliceValue[any](valueType)
	default:
		return nil // Invalid path
	}
}

func ReplaceValueWithMapContext[T Any](path []string, contextData map[string]T) any {
	if len(path) == 0 {
		return nil
	}

	fieldName := path[0]

	if value, exists := contextData[fieldName]; exists {
		return ReplaceValueWithContext(path[1:], value)
	}

	// Field doesn't exist in the context
	// Try prefixes
	for key, keyValue := range contextData {
		if strings.HasPrefix(key, fieldName) {
			return ReplaceValueWithContext(path[1:], keyValue)
		}
	}

	return nil
}

func ReplaceFromSchemaFormat(ctx *ReplaceContext) any {
	schema, ok := ctx.Schema.(*Schema)
	if !ok {
		return nil
	}

	switch schema.Format {
	case "date":
		return ctx.Faker.Time().Time(time.Now()).Format("2006-01-02")
	case "date-time":
		return ctx.Faker.Time().Time(time.Now()).Format("2006-01-02T15:04:05.000Z")
	case "email":
		return ctx.Faker.Internet().Email()
	case "uuid":
		return ctx.Faker.UUID()
	case "password":
		return ctx.Faker.Internet().Password()
	case "hostname":
		return ctx.Faker.Internet().Domain()
	case "uri", "url":
		return ctx.Faker.Internet().URL()
	}
	return nil
}

func ReplaceFromSchemaPrimitive(ctx *ReplaceContext) any {
	schema, ok := ctx.Schema.(*Schema)
	if !ok {
		return nil
	}
	faker := ctx.Faker

	switch schema.Type {
	case TypeString:
		return faker.Lorem().Word()
	case TypeInteger, TypeNumber:
		return faker.UInt32()
	case TypeBoolean:
		return faker.Bool()
	case TypeObject:
		// empty object with no response
		return map[string]any{}
	}
	return nil
}

func ReplaceFromSchemaExample(ctx *ReplaceContext) any {
	schema, ok := ctx.Schema.(*Schema)
	if !ok {
		return nil
	}
	return schema.Example
}

// ApplySchemaConstraints applies schema constraints to the value.
// It converts the input value to match the corresponding OpenAPI type specified in the schema.
func ApplySchemaConstraints(schema *Schema, res any) any {
	if schema == nil {
		return res
	}

	switch schema.Type {
	case TypeString:
		return applySchemaStringConstraints(schema, res.(string))
	case TypeInteger:
		floatValue, err := ToFloat64(res)
		if err != nil {
			log.Printf("Failed to convert %v to float64: %v", res, err)
			return nil
		}
		return int64(applySchemaNumberConstraints(schema, floatValue))
	case TypeNumber:
		floatValue, _ := ToFloat64(res)
		return applySchemaNumberConstraints(schema, floatValue)
	}
	return res
}

func applySchemaStringConstraints(schema *Schema, value string) any {
	if schema == nil {
		return value
	}

	minLength := schema.MinLength
	maxLength := schema.MaxLength
	pattern := schema.Pattern

	if pattern != "" && !ValidateStringWithPattern(value, pattern) {
		return nil
	}

	expectedEnums := make(map[string]bool)
	// remove random nulls from enum values
	for _, v := range schema.Enum {
		if v != nil {
			expectedEnums[v.(string)] = true
		}
	}

	if len(expectedEnums) > 0 && !expectedEnums[value] {
		return GetRandomKeyFromMap(expectedEnums)
	}

	if minLength > 0 && len(value) < int(minLength) {
		return value + strings.Repeat("-", int(minLength)-len(value))
	}

	if maxLength != nil && len(value) > int(*maxLength) {
		return value[:int(*maxLength)]
	}

	return value
}

func applySchemaNumberConstraints(schema *Schema, value float64) float64 {
	if schema == nil {
		return value
	}

	reqMin := schema.Min
	reqMax := schema.Max
	multOf := schema.MultipleOf

	if multOf != nil {
		value = float64(int(value / *multOf)) * *multOf
	}

	if reqMin != nil && value < *reqMin {
		value = *reqMin
	}

	if reqMax != nil && value > *reqMax {
		value = *reqMax
	}

	return value
}

func ReplaceFallback(ctx *ReplaceContext) any {
	schema, ok := ctx.Schema.(*Schema)
	if !ok {
		return nil
	}
	return schema.Default
}
