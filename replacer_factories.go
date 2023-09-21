package connexions

import (
	"github.com/jaswdr/faker"
	"reflect"
)

type ValueReplacer func(schemaOrContent any, state *ReplaceState) any

type ReplaceContext struct {
	Schema     any
	State      *ReplaceState
	AreaPrefix string
	Data       []map[string]any
	Faker      faker.Faker
}

var Replacers = []Replacer{
	ReplaceInHeaders,
	ReplaceInPath,
	ReplaceFromContext,
	ReplaceFromSchemaFormat,
	ReplaceFromSchemaPrimitive,
	ReplaceFromSchemaExample,
	ReplaceFromSchemaFallback,
}

var fake = faker.New()

func CreateValueReplacer(cfg *Config, contexts []map[string]any) ValueReplacer {
	return func(content any, state *ReplaceState) any {
		if state == nil {
			state = &ReplaceState{}
		}

		ctx := &ReplaceContext{
			Schema:     content,
			State:      state,
			Faker:      fake,
			AreaPrefix: cfg.App.ContextAreaPrefix,
			Data:       contexts,
		}

		for _, fn := range cfg.Replacers {
			res := fn(ctx)
			if res != nil && ctx.Schema != nil {
				if !HasCorrectSchemaValue(ctx, res) {
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

func IsCorrectlyReplacedType(value any, neededType string) bool {
	switch neededType {
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
