//go:build !integration

package connexions

import (
	assert2 "github.com/stretchr/testify/assert"
	"reflect"
	"testing"
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

	funcs := GetFakeFuncFactoryWithString()
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

func TestGetFakes(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	fakes := GetFakes()
	assert.Greater(len(fakes), 0)

	assert.True(true)
}
