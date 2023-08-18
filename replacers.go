package xs

import (
	"strings"
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
	userData := ctx.Resource.UserReplacements
	if userData == nil {
		return nil
	}
	// TODO(igor): add aliases and list of other contexts to the ctx.

	for _, name := range ctx.Resource.ContextOrder {
		replacements := ctx.Resource.Contexts[name]
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
	// case map[string]map[string]any:
	// 	return ReplaceValueWithMapContext[ReplacementContext](path, valueType)
	// base cases below:
	case FakeFunc:
		v := valueType().Get()
		println(v)
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
		return ctx.Faker.Date().Format("2006-01-02")
	case "date-time":
		return ctx.Faker.Date().Format("2006-01-02T15:04:05.000Z")
	case "email":
		return ctx.Faker.Email()
	case "uuid":
		return ctx.Faker.UUID()
	case "password":
		return ctx.Faker.Password(true, true, true, true, true, 12)
	case "hostname":
		return ctx.Faker.DomainName()
	case "ipv4", "ipv6":
		return ctx.Faker.IPv4Address()
	case "uri", "url":
		return ctx.Faker.URL()
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
		return faker.Word()
	case TypeInteger:
		return faker.Uint32()
	case TypeNumber:
		return faker.Uint32()
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

func ReplaceFallback(ctx *ReplaceContext) any {
	return nil
}
