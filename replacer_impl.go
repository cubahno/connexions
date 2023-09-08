package connexions

import (
	"fmt"
	"log"
	"strings"
	"time"
)

type Replacer func(ctx *ReplaceContext) any
type Any interface {
	string | int | bool | float64 | any
}

// NULL is used to force resolve to nil
const (
	NULL = "__null__"
)

func HasCorrectSchemaType(ctx *ReplaceContext, value any) bool {
	schema, ok := ctx.Schema.(*Schema)
	if !ok {
		// TODO(igor): check how to handle with other content schemas
		return true
	}

	if schema == nil {
		return true
	}

	return IsCorrectlyReplacedType(value, schema.Type)
}

func ReplaceInHeaders(ctx *ReplaceContext) any {
	if !ctx.State.IsHeader {
		return nil
	}
	return replaceInArea(ctx, "header")
}

func ReplaceInPath(ctx *ReplaceContext) any {
	if !ctx.State.IsPathParam {
		return nil
	}
	return replaceInArea(ctx, "path")
}

func replaceInArea(ctx *ReplaceContext, area string) any {
	ctxAreaPrefix := ctx.Resource.ContextAreaPrefix
	if ctxAreaPrefix == "" {
		return nil
	}

	snakedNamePath := []string{ToSnakeCase(ctx.State.NamePath[0])}

	for _, data := range ctx.Resource.ContextData {
		replacements, ok := data[fmt.Sprintf("%s%s", ctxAreaPrefix, area)]
		if !ok {
			continue
		}

		if res := ReplaceValueWithContext(snakedNamePath, replacements); res != nil {
			return res
		}
	}

	return nil
}

func ReplaceFromContext(ctx *ReplaceContext) any {
	var snakedNamePath []string
	// context data is stored in snake case
	for _, name := range ctx.State.NamePath {
		snakedNamePath = append(snakedNamePath, ToSnakeCase(name))
	}

	for _, data := range ctx.Resource.ContextData {
		if res := ReplaceValueWithContext(snakedNamePath, data); res != nil {
			return res
		}
	}

	return nil
}

func ReplaceValueWithContext(path []string, contextData any) interface{} {
	switch valueType := contextData.(type) {
	case map[string]string:
		return replaceValueWithMapContext[string](path, valueType)
	case map[string]int:
		return replaceValueWithMapContext[int](path, valueType)
	case map[string]bool:
		return replaceValueWithMapContext[bool](path, valueType)
	case map[string]float64:
		return replaceValueWithMapContext[float64](path, valueType)
	case map[string]any:
		return replaceValueWithMapContext[any](path, valueType)

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

func replaceValueWithMapContext[T Any](path []string, contextData map[string]T) any {
	if len(path) == 0 {
		return nil
	}

	fieldName := path[len(path)-1]

	// Expect direct match first.
	if value, exists := contextData[fieldName]; exists {
		return ReplaceValueWithContext(path[1:], value)
	}

	// Shrink the context data to the last element of the path.
	if len(path) > 1 {
		fst := path[0]
		if value, exists := contextData[fst]; exists {
			return ReplaceValueWithContext(path[1:], value)
		}
	}

	// Field doesn't exist in the context as-is.
	// But the context field might be a regex pattern.
	for key, keyValue := range contextData {
		if MaybeRegexPattern(key) && ValidateStringWithPattern(fieldName, key) {
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
	case "date-time", "datetime":
		return ctx.Faker.Time().Time(time.Now()).Format("2006-01-02T15:04:05.000Z")
	case "email":
		return ctx.Faker.Internet().Email()
	case "uuid":
		return ctx.Faker.UUID().V4()
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
func ApplySchemaConstraints(openAPISchema any, res any) any {
	if openAPISchema == nil {
		return res
	}

	schema, ok := openAPISchema.(*Schema)
	if !ok {
		return res
	}
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

	if maxLength > 0 && int64(len(value)) > maxLength {
		return value[:maxLength]
	}

	return value
}

func applySchemaNumberConstraints(schema *Schema, value float64) float64 {
	if schema == nil {
		return value
	}

	reqMin := schema.Minimum
	reqMax := schema.Maximum
	multOf := schema.MultipleOf

	if multOf != 0 {
		value = float64(int(value/multOf)) * multOf
	}

	if reqMin != 0 && value < reqMin {
		value = reqMin
	}

	if reqMax != 0 && value > reqMax {
		value = reqMax
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
