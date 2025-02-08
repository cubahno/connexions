package internal

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/cubahno/connexions/internal/types"
	"github.com/lucasjones/reggen"
)

// Replacer is a function that returns a value to replace the original value with.
// Replacer functions are predefined and set in the correct order to be executed.
type Replacer func(ctx *ReplaceContext) any

// Any is a type that can be used to represent any type in generics.
type Any interface {
	string | int | bool | float64 | any
}

// NULL is used to force resolve to nil
const (
	NULL = "__null__"
)

// HasCorrectSchemaValue checks if the value is of the correct type and format.
func HasCorrectSchemaValue(ctx *ReplaceContext, value any) bool {
	// TODO: check how to handle other content schemas
	if ctx.Schema == nil {
		return true
	}
	schema, ok := ctx.Schema.(*Schema)
	if !ok || schema == nil {
		return true
	}

	if !IsCorrectlyReplacedType(value, schema.Type) {
		return false
	}

	reqFormat := schema.Format
	if reqFormat == "" {
		return true
	}

	switch reqFormat {
	case "int32":
		_, ok = types.ToInt32(value)
		return ok
	case "int64":
		_, ok = types.ToInt64(value)
		return ok
	case "date":
		v, err := time.Parse("2006-01-02", value.(string))
		return err == nil && !v.IsZero()
	case "date-time", "datetime":
		v, err := time.Parse("2006-01-02T15:04:05.000Z", value.(string))
		return err == nil && !v.IsZero()
	default:
		return true
	}
}

// ReplaceInHeaders is a replacer that replaces values only in headers.
func ReplaceInHeaders(ctx *ReplaceContext) any {
	if !ctx.State.IsHeader {
		return nil
	}
	v := replaceInArea(ctx, "header")

	schema, ok := ctx.Schema.(*Schema)
	if !ok {
		return v
	}

	name := ctx.State.NamePath[0]
	format := schema.Format

	if name == "authorization" {
		switch format {
		case "basic":
			if v == nil {
				v = fmt.Sprintf("%s:%s", ctx.Faker.Internet().User(), ctx.Faker.Internet().Password())
			}
			return "Basic " + types.Base64Encode(v.(string))
		case "bearer":
			if v == nil {
				v = ctx.Faker.Internet().Password()
			}
			return "Bearer " + v.(string)
		}
	}

	return v
}

// ReplaceInPath is a replacer that replaces values only in path parameters.
func ReplaceInPath(ctx *ReplaceContext) any {
	if !ctx.State.IsPathParam {
		return nil
	}
	return replaceInArea(ctx, "path")
}

func replaceInArea(ctx *ReplaceContext, area string) any {
	ctxAreaPrefix := ctx.AreaPrefix
	if ctxAreaPrefix == "" {
		return nil
	}

	snakedNamePath := []string{types.ToSnakeCase(ctx.State.NamePath[0])}

	for _, data := range ctx.Data {
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

// ReplaceFromContext is a replacer that replaces values from the context.
func ReplaceFromContext(ctx *ReplaceContext) any {
	var snakedNamePath []string
	// context data is stored in snake case
	for _, name := range ctx.State.NamePath {
		snakedNamePath = append(snakedNamePath, types.ToSnakeCase(name))
	}

	for _, data := range ctx.Data {
		if res := ReplaceValueWithContext(snakedNamePath, data); res != nil {
			return CastToSchemaFormat(ctx, res)
		}
	}

	return nil
}

// CastToSchemaFormat casts the value to the schema format if possible.
// If the schema format is not specified, the value is returned as-is.
func CastToSchemaFormat(ctx *ReplaceContext, value any) any {
	schema, ok := ctx.Schema.(*Schema)
	if !ok || schema == nil {
		return value
	}

	switch schema.Format {
	case "int32":
		if v, ok := types.ToInt32(value); ok {
			return v
		}
		return value
	case "int64":
		if v, ok := types.ToInt64(value); ok {
			return v
		}
		return value
	default:
		return value
	}
}

// ReplaceValueWithContext is a replacer that replaces values from the context.
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
		return types.GetRandomSliceValue(valueType)
	case []int:
		return types.GetRandomSliceValue(valueType)
	case []bool:
		return types.GetRandomSliceValue(valueType)
	case []float64:
		return types.GetRandomSliceValue(valueType)
	case []any:
		return types.GetRandomSliceValue[any](valueType)
	default:
		return nil // unmapped type
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
		if types.MaybeRegexPattern(key) && types.ValidateStringWithPattern(fieldName, key) {
			return ReplaceValueWithContext(path[1:], keyValue)
		}
	}

	return nil
}

// ReplaceFromSchemaFormat is a replacer that replaces values from the schema format.
func ReplaceFromSchemaFormat(ctx *ReplaceContext) any {
	schema, ok := ctx.Schema.(*Schema)
	if !ok || schema == nil {
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
	case "int32":
		return int32(math.Abs(float64(ctx.Faker.Int32())))
	case "int64":
		return int64(math.Abs(float64(ctx.Faker.Int64())))
	case "ipv4":
		return ctx.Faker.Internet().Ipv4()
	case "ipv6":
		return ctx.Faker.Internet().Ipv6()
	}
	return nil
}

// ReplaceFromSchemaPrimitive is a replacer that replaces values from the schema primitive.
func ReplaceFromSchemaPrimitive(ctx *ReplaceContext) any {
	schema, ok := ctx.Schema.(*Schema)
	if !ok || schema == nil {
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

// ReplaceFromSchemaExample is a replacer that replaces values from the schema example.
func ReplaceFromSchemaExample(ctx *ReplaceContext) any {
	schema, ok := ctx.Schema.(*Schema)
	if !ok || schema == nil {
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
	if !ok || schema == nil {
		return res
	}

	switch schema.Type {
	case TypeBoolean:
		if len(schema.Enum) > 0 {
			return types.GetRandomSliceValue(schema.Enum)
		}
	case TypeString:
		return applySchemaStringConstraints(schema, res.(string))
	case TypeInteger:
		floatValue, err := types.ToFloat64(res)
		if err != nil {
			log.Printf("Failed to convert %v to float64: %v", res, err)
			return nil
		}
		return int64(applySchemaNumberConstraints(schema, floatValue))
	case TypeNumber:
		floatValue, err := types.ToFloat64(res)
		if err != nil {
			log.Printf("Failed to convert %v to float64: %v", res, err)
			return nil
		}
		return applySchemaNumberConstraints(schema, floatValue)
	}
	return res
}

// applySchemaStringConstraints applies string constraints to the value.
// in case of invalid value, the function tries to correct it.
func applySchemaStringConstraints(schema *Schema, value string) any {
	if schema == nil {
		return value
	}

	minLength := schema.MinLength
	maxLength := schema.MaxLength
	pattern := schema.Pattern

	expectedEnums := make(map[string]bool)
	// remove random nulls from enum values
	for _, v := range schema.Enum {
		if v != nil {
			// values can be numbers in the schema too, make sure we get strings here
			// otherwise we'll get a panic if validation is on.
			expectedEnums[fmt.Sprintf("%v", v)] = true
		}
	}

	if len(expectedEnums) > 0 && !expectedEnums[value] {
		return types.GetRandomKeyFromMap(expectedEnums)
	}

	if pattern != "" && !types.ValidateStringWithPattern(value, pattern) {
		// safest way here is to return the example if it exists
		if schema.Example != nil {
			return schema.Example
		}
		value = createStringFromPattern(pattern)

		// regex will be cached
		if !types.ValidateStringWithPattern(value, pattern) {
			return nil
		}

		return value
	}

	if minLength > 0 && len(value) < int(minLength) {
		value += strings.Repeat("-", int(minLength)-len(value))
	}

	if maxLength > 0 && int64(len(value)) > maxLength {
		value = value[:maxLength]
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

	expectedEnums := make(map[string]bool)
	// remove random nulls from enum values
	for _, v := range schema.Enum {
		if v != nil {
			// we can't have floats as keys in the map, so we convert them to strings
			expectedEnums[fmt.Sprintf("%v", v)] = true
		}
	}

	vStr := fmt.Sprintf("%v", value)
	if len(expectedEnums) > 0 && !expectedEnums[vStr] {
		enumed := types.GetRandomKeyFromMap(expectedEnums)
		f, _ := strconv.ParseFloat(enumed, 64)
		return f
	}

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

// createStringFromPattern creates a string from the given pattern.
// This is a naive implementation to support the most common patterns.
func createStringFromPattern(pattern string) string {
	if pattern == "" {
		return ""
	}
	result, err := reggen.Generate(pattern, 10)
	if err != nil {
		return ""
	}
	return result
}

// ReplaceFromSchemaFallback is the last resort to get a value from the schema.
func ReplaceFromSchemaFallback(ctx *ReplaceContext) any {
	schema, ok := ctx.Schema.(*Schema)
	if !ok || schema == nil {
		return nil
	}

	return schema.Default
}

// IsMatchSchemaReadWriteToState checks if the given schema is read-write match.
// ReadOnly - A property that is only available in a response.
// WriteOnly - A property that is only available in a request.
func IsMatchSchemaReadWriteToState(schema *Schema, state *ReplaceState) bool {
	// unable to determine
	if schema == nil || state == nil {
		return true
	}

	if schema.ReadOnly && !state.IsContentReadOnly {
		return false
	}

	if schema.WriteOnly && !state.IsContentWriteOnly {
		return false
	}

	return true
}
