package xs

import (
	"github.com/brianvoe/gofakeit/v6"
	"github.com/getkin/kin-openapi/openapi3"
	"log"
	"reflect"
)

type ValueReplacer func(schemaOrContent any, state *ReplaceState) any
type ValueReplacerFactory func(resource *Resource) ValueReplacer

type ReplaceContext struct {
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

func CreateValueSchemaReplacerFactory() func(resource *Resource) ValueReplacer {
	faker := gofakeit.New(0)
	return func(resource *Resource) ValueReplacer {
		return func(content any, state *ReplaceState) any {
			schema, ok := content.(*openapi3.Schema)
			if !ok {
				log.Printf("content is not *openapi3.Schema, but %s", reflect.TypeOf(content))
				return nil
			}

			if schema.Example != nil {
				return schema.Example
			}

			switch schema.Type {
			case openapi3.TypeString:
				return faker.Word()
			case openapi3.TypeInteger:
				return faker.Uint32()
			case openapi3.TypeNumber:
				return faker.Uint32()
			case openapi3.TypeBoolean:
				return faker.Bool()
			}

			return nil
		}
	}
}

func CreateValueContentReplacerFactory() ValueReplacerFactory {
	faker := gofakeit.New(0)
	return func(resource *Resource) ValueReplacer {
		return func(content any, state *ReplaceState) any {
			switch content.(type) {
			case string:
				if state.IsURLParam {
					return faker.Uint8()
				}
			}
			return content
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
