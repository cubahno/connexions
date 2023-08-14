package xs

import (
	"github.com/brianvoe/gofakeit/v6"
	"github.com/getkin/kin-openapi/openapi3"
	"reflect"
)

type ValueReplacer func(schemaOrContent any, state *ReplaceState) any
type ValueReplacerFactory func(resource *Resource) ValueReplacer

type ReplaceContext struct {
	Schema any
	State        *ReplaceState
	Resource     *Resource
	Name         string
	OriginalName string
	Faker        *gofakeit.Faker
}

type Resource struct {
	Service          string
	Path             string
	UserReplacements map[string]any
}

func CreateValueReplacerFactory() ValueReplacerFactory {
	faker := gofakeit.New(0)

	fns := []Replacer{
		ReplaceInHeaders,
		ReplaceFromPredefined,
		// from contexts
		// from alias, maybe not needed
		ReplaceFromSchemaFormat,
		ReplaceFromSchemaPrimitive,
		ReplaceFromSchemaExample,
		ReplaceFallback,
	}

	return func(resource *Resource) ValueReplacer {
		if resource == nil {
			resource = &Resource{}
		}

		return func(content any, state *ReplaceState) any {
			if state == nil {
				state = &ReplaceState{}
			}

			name, original := ExtractNames(state.NamePath)

			ctx := &ReplaceContext{
				Name: name,
				OriginalName: original,
				Schema: content,
				State:        state,
				Resource:     resource,
				Faker: faker,
			}

			for _, fn := range fns {
				res := fn(ctx)
				if res != nil && HasCorrectSchemaType(ctx, res) {
					return res
				}
				// return nil if function suggests
				if str, ok := res.(string); ok {
					if str == NULL {
						return nil
					}
				}
			}

			return nil
		}
	}
}

func IsCorrectlyReplacedType(value any, needed string) bool {
	switch needed {
	case openapi3.TypeString:
		_, ok := value.(string)
		return ok
	case openapi3.TypeInteger:
		return IsInteger(value)
	case openapi3.TypeNumber:
		return IsNumber(value)
	case openapi3.TypeBoolean:
		_, ok := value.(bool)
		return ok
	case openapi3.TypeObject:
		return reflect.TypeOf(value).Kind() == reflect.Map
	case openapi3.TypeArray:
		val := reflect.ValueOf(value)
		return val.Kind() == reflect.Slice || val.Kind() == reflect.Array
	default:
		return false
	}
}
