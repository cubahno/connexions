//go:build !integration

package connexions

import (
	assert2 "github.com/stretchr/testify/assert"
	"testing"
)

func TestNewReplaceContext(t *testing.T) {
	assert := assert2.New(t)
	res := NewReplaceContext(nil, nil, nil)
	assert.NotNil(res.Faker)
}

func TestReplacers(t *testing.T) {
	assert := assert2.New(t)
	assert.Equal(7, len(Replacers))
}

func TestCreateValueReplacerFactory(t *testing.T) {
	assert := assert2.New(t)
	fooReplacer := func(ctx *ReplaceContext) any { return "foo" }
	intReplacer := func(ctx *ReplaceContext) any { return 1 }
	nilReplacer := func(ctx *ReplaceContext) any { return nil }
	forceNullReplacer := func(ctx *ReplaceContext) any { return NULL }

	t.Run("with-nil-state", func(t *testing.T) {
		fn := CreateValueReplacerFactory([]Replacer{fooReplacer})(nil)
		res := fn("", nil)
		assert.Equal("foo", res)
	})

	t.Run("with-incorrect-schema-type", func(t *testing.T) {
		fn := CreateValueReplacerFactory([]Replacer{fooReplacer, intReplacer})(nil)
		schema := CreateSchemaFromString(t, `{"type": "integer"}`)
		res := fn(schema, nil)
		assert.Equal(int64(1), res)
	})

	t.Run("with-force-null", func(t *testing.T) {
		fn := CreateValueReplacerFactory([]Replacer{forceNullReplacer, fooReplacer})(nil)
		res := fn("", nil)
		assert.Nil(res)
	})

	t.Run("continues-on-nil", func(t *testing.T) {
		fn := CreateValueReplacerFactory([]Replacer{nilReplacer, fooReplacer})(nil)
		res := fn("foo", nil)
		assert.Equal("foo", res)
	})

	t.Run("finishes-with-nil", func(t *testing.T) {
		fn := CreateValueReplacerFactory([]Replacer{nilReplacer, nilReplacer})(nil)
		res := fn("foo", nil)
		assert.Nil(res)
	})

	// t.Run("from-example", func(t *testing.T) {
	// 	schema := CreateSchemaFromString(t, `{"type": "string", "example": "foo"}`)
	// 	res := fn(schema, nil)
	// 	assert.Equal(t, "foo", res)
	// })

	// t.Run("string", func(t *testing.T) {
	// 	schema := CreateSchemaFromString(t, `{"type": "string"}`)
	// 	res := fn(schema, nil)
	//
	// 	v, ok := res.(string)
	// 	assert.True(ok)
	// 	assert.Greater(t, len(v), 0)
	// })
	//
	// t.Run("integer", func(t *testing.T) {
	// 	schema := CreateSchemaFromString(t, `{"type": "integer"}`)
	// 	res := fn(schema, nil)
	//
	// 	v, ok := res.(int64)
	// 	assert.True(ok)
	// 	assert.Greater(v, int64(0))
	// })
	//
	// t.Run("number", func(t *testing.T) {
	// 	schema := CreateSchemaFromString(t, `{"type": "number"}`)
	// 	res := fn(schema, nil)
	//
	// 	v, ok := res.(float64)
	// 	assert.True(ok)
	// 	assert.Greater(v, float64(0))
	// })
	//
	// t.Run("boolean", func(t *testing.T) {
	// 	schema := CreateSchemaFromString(t, `{"type": "boolean"}`)
	// 	res := fn(schema, nil)
	//
	// 	_, ok := res.(bool)
	// 	assert.True(ok)
	// })
	//
	// t.Run("unknown", func(t *testing.T) {
	// 	schema := CreateSchemaFromString(t, `{"type": "x"}`)
	// 	res := fn(schema, nil)
	// 	assert.Nil(res)
	// })
}

func TestIsCorrectlyReplacedType(t *testing.T) {
	assert := assert2.New(t)
	type testCase struct {
		value    any
		needed   string
		expected bool
	}

	testCases := []testCase{
		{"foo", TypeString, true},
		{1, TypeString, false},
		{"foo", TypeInteger, false},
		{1, TypeInteger, true},
		{"1", TypeNumber, false},
		{1, TypeNumber, true},
		{1.12, TypeNumber, true},
		{"true", TypeBoolean, false},
		{true, TypeBoolean, true},
		{[]string{"foo", "bar"}, TypeArray, true},
		{[]int{1, 2}, TypeArray, true},
		{[]bool{true, false}, TypeArray, true},
		{map[string]string{"foo": "bar"}, TypeObject, true},
		{map[string]int{"foo": 1}, TypeObject, true},
		{map[string]bool{"foo": true}, TypeObject, true},
		{map[string]any{"foo": "bar"}, TypeObject, true},
		{"foo", TypeObject, false},
		{"foo", "bar", false},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			assert.Equal(tc.expected, IsCorrectlyReplacedType(tc.value, tc.needed))
		})
	}
}
