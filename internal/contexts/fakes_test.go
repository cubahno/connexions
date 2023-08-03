//go:build !integration

package contexts

import (
	"reflect"
	"testing"

	"github.com/jaswdr/faker/v2"
	assert2 "github.com/stretchr/testify/assert"
)

func TestMixedValues(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	s := StringValue("hello")
	assert.Equal("hello", s.Get())

	i := IntValue(123)
	assert.Equal(int64(123), i.Get())

	f := Float64Value(123.456)
	assert.Equal(123.456, f.Get())

	b := BoolValue(true)
	assert.Equal(true, b.Get())
}

func TestFromReflectedStringValue(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	fn := reflect.ValueOf(func() string {
		return "hello"
	})
	f := fromReflectedStringValue(fn)
	assert.Equal("hello", f().Get())
}

func TestFromReflectedIntValue(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	fn := reflect.ValueOf(func() int {
		return 123
	})
	f := fromReflectedIntValue(fn)
	assert.Equal(int64(123), f().Get())
}

func TestFromReflectedUIntValue(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	fn := reflect.ValueOf(func() uint {
		return 123
	})
	f := fromReflectedUIntValue(fn)
	assert.Equal(int64(123), f().Get())
}

func TestFromReflectedBoolValue(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	fn := reflect.ValueOf(func() bool {
		return true
	})
	f := fromReflectedBoolValue(fn)
	assert.Equal(true, f().Get())
}

func TestFromReflectedFloat64Value(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	fn := reflect.ValueOf(func() float64 {
		return 123.456
	})
	f := fromReflectedFloat64Value(fn)
	assert.Equal(123.456, f().Get())
}

func TestGetFakeFuncFactoryWithString(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	funcs := getFakeFuncFactoryWithString()
	assert.NotNil(funcs)

	expectedKeys := []string{
		"botify",
		"echo",
	}
	var keys []string
	for key, fn := range funcs {
		assert.NotNil(fn)
		keys = append(keys, key)
		res := fn("hello")()
		assert.Greater(len(res.Get().(string)), 0)
	}

	assert.ElementsMatch(expectedKeys, keys)
}

func TestGetFakeFuncFactoryWith2Strings(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	funcs := getFakeFuncFactoryWith2Strings()
	assert.NotNil(funcs)

	expectedKeys := []string{
		"int8_between",
	}
	var keys []string
	for key := range funcs {
		keys = append(keys, key)
	}

	assert.ElementsMatch(expectedKeys, keys)

	t.Run("int8_between valid range", func(t *testing.T) {
		fn := funcs["int8_between"]("10", "20")
		for i := 0; i < 100; i++ {
			val := fn().Get().(int64)
			assert.GreaterOrEqual(val, int64(10))
			assert.LessOrEqual(val, int64(20))
		}
	})

	t.Run("int8_between single value", func(t *testing.T) {
		fn := funcs["int8_between"]("5", "5")
		val := fn().Get().(int64)
		assert.Equal(int64(5), val)
	})

	t.Run("int8_between boundary values", func(t *testing.T) {
		fn := funcs["int8_between"]("1", "10")
		for i := 0; i < 100; i++ {
			val := fn().Get().(int64)
			assert.GreaterOrEqual(val, int64(1))
			assert.LessOrEqual(val, int64(10))
		}
	})
}

func TestGetFakes(t *testing.T) {
	assert := assert2.New(t)

	fakes := getFakes()
	assert.Greater(len(fakes), 0)

	assert.Equal("bar", fakes["foo"]().Get())
}

func TestGetFakeFuncs(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	visited := make(map[reflect.Type]bool)
	fakes := getFakeFuncs(faker.New(), "", visited)
	assert.Greater(len(fakes), 0)
}
