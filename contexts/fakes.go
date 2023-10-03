package contexts

import (
	"github.com/cubahno/connexions/internal"
	"github.com/jaswdr/faker"
	"reflect"
)

// FakeFunc is a function that returns a MixedValue.
// This is u unified way to work with different return types from fake library.
type FakeFunc func() MixedValue

// FakeFuncFactoryWithString is a function that returns a FakeFunc.
type FakeFuncFactoryWithString func(value string) FakeFunc

// Fakes is a map of registered fake functions.
var Fakes = GetFakes()

// MixedValue is a value that can represent string, int, float64, or bool type.
type MixedValue interface {
	Get() any
}

type StringValue string
type IntValue int
type Float64Value float64
type BoolValue bool

func (s StringValue) Get() any {
	return string(s)
}

func (i IntValue) Get() any {
	return int64(i)
}

func (f Float64Value) Get() any {
	return float64(f)
}

func (b BoolValue) Get() any {
	return bool(b)
}

// GetFakeFuncFactoryWithString returns a map of utility fake functions.
func GetFakeFuncFactoryWithString() map[string]FakeFuncFactoryWithString {
	fake := faker.New()
	return map[string]FakeFuncFactoryWithString{
		"botify": func(pattern string) FakeFunc {
			return func() MixedValue {
				return StringValue(fake.Bothify(pattern))
			}
		},
		"echo": func(pattern string) FakeFunc {
			return func() MixedValue {
				return StringValue(pattern)
			}
		},
	}
}

// GetFakes returns a map of fake functions from underlying fake library by
// gathering all exported methods from the faker.Faker struct into map.
// The keys are the snake_cased dot-separated method names, which reflect the location of the function:
// For example: person.first_name will return a fake first name from the Person struct.
func GetFakes() map[string]FakeFunc {
	return getFakeFuncs(faker.New(), "")
}

// GetFakesFromStruct returns a map of fake functions from a struct.
// The keys are the snake_cased struct field names, and the values are the fake functions.
// The fake functions can be called to get a MixedValue, which can be converted to a string, int, float64, or bool.
func getFakeFuncs(obj any, prefix string) map[string]FakeFunc {
	res := make(map[string]FakeFunc)

	ref := reflect.ValueOf(obj)
	for i := 0; i < ref.NumMethod(); i++ {
		mType := ref.Type().Method(i)
		name := mType.Name
		mappedName := internal.ToSnakeCase(name)

		fn := ref.MethodByName(name)
		numIn := mType.Type.NumIn() - 1

		if numIn > 0 {
			continue
		}

		returnType := mType.Type.Out(0).Kind()

		switch returnType {
		case reflect.Struct:
			structInstance := fn.Call(nil)[0].Interface()
			fromStruct := getFakeFuncs(structInstance, mappedName+".")
			for k, v := range fromStruct {
				res[k] = v
			}
		case reflect.Float32, reflect.Float64:
			res[prefix+mappedName] = fromReflectedFloat64Value(fn)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			res[prefix+mappedName] = fromReflectedIntValue(fn)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			res[prefix+mappedName] = fromReflectedUIntValue(fn)
		case reflect.Bool:
			res[prefix+mappedName] = fromReflectedBoolValue(fn)
		case reflect.String:
			res[prefix+mappedName] = fromReflectedStringValue(fn)
		default:
		}
	}

	return res
}

func fromReflectedStringValue(fn reflect.Value) FakeFunc {
	return func() MixedValue {
		return StringValue(fn.Call(nil)[0].String())
	}
}

func fromReflectedBoolValue(fn reflect.Value) FakeFunc {
	return func() MixedValue {
		return BoolValue(fn.Call(nil)[0].Bool())
	}
}

func fromReflectedIntValue(fn reflect.Value) FakeFunc {
	return func() MixedValue {
		return IntValue(fn.Call(nil)[0].Int())
	}
}

func fromReflectedUIntValue(fn reflect.Value) FakeFunc {
	return func() MixedValue {
		return IntValue(fn.Call(nil)[0].Uint())
	}
}

func fromReflectedFloat64Value(fn reflect.Value) FakeFunc {
	return func() MixedValue {
		return Float64Value(fn.Call(nil)[0].Float())
	}
}
