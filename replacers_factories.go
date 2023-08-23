package connexions

import (
	"github.com/jaswdr/faker"
	"reflect"
)

type ValueReplacer func(schemaOrContent any, state *ReplaceState) any
type ValueReplacerFactory func(resource *Resource) ValueReplacer

type ReplaceContext struct {
	Schema   any
	State    *ReplaceState
	Resource *Resource
	Faker    faker.Faker
}

type Resource struct {
	Service           string
	Path              string
	ContextAreaPrefix string
	ContextData       []map[string]any
}

func CreateValueReplacerFactory() ValueReplacerFactory {
	fake := faker.New()

	fns := []Replacer{
		ReplaceInHeaders,
		ReplaceInPath,
		ReplaceFromContext,
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

			ctx := &ReplaceContext{
				Schema:   content,
				State:    state,
				Resource: resource,
				Faker:    fake,
			}

			for _, fn := range fns {
				res := fn(ctx)
				if res != nil && ctx.Schema != nil {
					if !HasCorrectSchemaType(ctx, res) {
						continue
					}
					res = ApplySchemaConstraints(ctx.Schema, res)
				}

				if res == nil {
					continue
				}

				// return nil if function suggests
				if str, ok := res.(string); ok {
					if str == NULL {
						return nil
					}
				}
				return res
			}

			return nil
		}
	}
}

func IsCorrectlyReplacedType(value any, needed string) bool {
	switch needed {
	case TypeString:
		_, ok := value.(string)
		return ok
	case TypeInteger:
		return IsInteger(value)
	case TypeNumber:
		return IsNumber(value)
	case TypeBoolean:
		_, ok := value.(bool)
		return ok
	case TypeObject:
		return reflect.TypeOf(value).Kind() == reflect.Map
	case TypeArray:
		val := reflect.ValueOf(value)
		return val.Kind() == reflect.Slice || val.Kind() == reflect.Array
	default:
		return false
	}
}
