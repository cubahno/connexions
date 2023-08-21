package xs

import (
	"github.com/jaswdr/faker"
	"reflect"
)

type FakeValue interface {
	~string | ~int | ~float64 | ~bool
}
type FakeFunc func() MixedValue
type FakeFuncFactoryWithString func(value string) FakeFunc

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

func AsString(f func() string) FakeFunc {
	return func() MixedValue {
		return StringValue(f())
	}
}

func AsInt64(f func() int64) FakeFunc {
	return func() MixedValue {
		return IntValue(f())
	}
}

func AsFloat64(f func() float64) FakeFunc {
	return func() MixedValue {
		return Float64Value(f())
	}
}

func AsBool(f func() bool) FakeFunc {
	return func() MixedValue {
		return BoolValue(f())
	}
}

func FromReflectedStringValue(fn reflect.Value) FakeFunc {
	return func() MixedValue {
		return StringValue(fn.Call(nil)[0].String())
	}
}

func FromReflectedBoolValue(fn reflect.Value) FakeFunc {
	return func() MixedValue {
		return BoolValue(fn.Call(nil)[0].Bool())
	}
}

func FromReflectedIntValue(fn reflect.Value) FakeFunc {
	return func() MixedValue {
		return IntValue(fn.Call(nil)[0].Int())
	}
}

func FromReflectedUIntValue(fn reflect.Value) FakeFunc {
	return func() MixedValue {
		return IntValue(fn.Call(nil)[0].Uint())
	}
}

func FromReflectedFloat64Value(fn reflect.Value) FakeFunc {
	return func() MixedValue {
		return Float64Value(fn.Call(nil)[0].Float())
	}
}

func GetFakeFuncFactoryWithString() map[string]FakeFuncFactoryWithString {
	fake := faker.New()

	return map[string]FakeFuncFactoryWithString{
		"Botify": func(pattern string) FakeFunc {
			return func() MixedValue {
				return StringValue(fake.Bothify(pattern))
			}
		},
	}
}

// GetFakes returns a map of fake functions from underlying fake lib by
// gathering all exported methods from the faker.Faker struct into map.
func GetFakes() map[string]FakeFunc {
	fake := faker.New()
	return getFakeFuncs(fake, "")
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
		mappedName := ToSnakeCase(name)

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
			res[prefix+mappedName] = FromReflectedFloat64Value(fn)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			res[prefix+mappedName] = FromReflectedIntValue(fn)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			res[prefix+mappedName] = FromReflectedUIntValue(fn)
		case reflect.Bool:
			res[prefix+mappedName] = FromReflectedBoolValue(fn)
		case reflect.String:
			res[prefix+mappedName] = FromReflectedStringValue(fn)
		default:
		}
	}

	return res
}
