package replacer

import (
	"reflect"

	"github.com/cubahno/connexions/internal/config"
	"github.com/cubahno/connexions/internal/types"
	"github.com/jaswdr/faker/v2"
)

// ValueReplacer is a function that replaces value in schema or content.
// This function should encapsulate all the logic, data, contexts etc. of replacing values.
type ValueReplacer func(schemaOrContent any, state *ReplaceState) any

// ReplaceContext is a context that is used to replace values in schemas and contents.
//
// Schema is a schema that is used to replace values.
// Currently only OpenAPI Schema is supported.
// It does not depend on schema provider as this is already converted to internal Schema type.
//
// State is a state of the current replace operation.
// It is used to store information about the current element, including its name, index, content type etc.
//
// AreaPrefix is a prefix that is used to identify the correct section
// in the context config for specific replacement area.
// e.g. in-
// then in the contexts we should have:
// in-header:
//
//	X-GeneratedRequest-ID: 123
//
// in-path:
//
//	user_id: 123
//
// Data is a list of contexts that are used to replace values.
// Faker is a faker instance that is used to generate fake data.
type ReplaceContext struct {
	Schema     any
	State      *ReplaceState
	AreaPrefix string
	Data       []map[string]any
	Faker      faker.Faker
}

// Replacers is a list of replacers that are used to replace values in schemas and contents in the specified order.
var Replacers = []Replacer{
	ReplaceInHeaders,
	ReplaceInPath,
	ReplaceFromContext,
	ReplaceFromSchemaExample,
	ReplaceFromSchemaFormat,
	ReplaceFromSchemaPrimitive,
	ReplaceFromSchemaFallback,
}

// CreateValueReplacer is a factory that creates a new ValueReplacer instance from the given config and contexts.
func CreateValueReplacer(cfg *config.Config, replacers []Replacer, contexts []map[string]any) ValueReplacer {
	return func(content any, state *ReplaceState) any {
		if state == nil {
			state = NewReplaceState()
		}

		ctx := &ReplaceContext{
			Schema: content,
			State:  state,
			// initialize faker here, global var is racy
			Faker:      faker.New(),
			AreaPrefix: cfg.App.ContextAreaPrefix,
			Data:       contexts,
		}

		for _, fn := range replacers {
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

// IsCorrectlyReplacedType checks if the given value is of the correct schema type.
func IsCorrectlyReplacedType(value any, neededType string) bool {
	switch neededType {
	case types.TypeString:
		_, ok := value.(string)
		return ok
	case types.TypeInteger:
		return types.IsInteger(value)
	case types.TypeNumber:
		return types.IsNumber(value)
	case types.TypeBoolean:
		_, ok := value.(bool)
		return ok
	case types.TypeObject:
		return reflect.TypeOf(value).Kind() == reflect.Map
	case types.TypeArray:
		val := reflect.ValueOf(value)
		return val.Kind() == reflect.Slice || val.Kind() == reflect.Array
	default:
		return false
	}
}
