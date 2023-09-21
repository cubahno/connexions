//go:build !integration

package connexions

import (
	assert2 "github.com/stretchr/testify/assert"
	"testing"
)

func TestReplacers(t *testing.T) {
	assert := assert2.New(t)
	assert.Equal(7, len(Replacers))
}

func TestCreateValueReplacer(t *testing.T) {
	assert := assert2.New(t)
	fooReplacer := func(ctx *ReplaceContext) any { return "foo" }
	intReplacer := func(ctx *ReplaceContext) any { return 1 }
	nilReplacer := func(ctx *ReplaceContext) any { return nil }
	forceNullReplacer := func(ctx *ReplaceContext) any { return NULL }

	t.Run("with-nil-state", func(t *testing.T) {
		cfg := NewDefaultConfig("")
		cfg.Replacers = []Replacer{fooReplacer}
		fn := CreateValueReplacer(cfg, nil)
		res := fn("", nil)
		assert.Equal("foo", res)
	})

	t.Run("with-incorrect-schema-type", func(t *testing.T) {
		cfg := NewDefaultConfig("")
		cfg.Replacers = []Replacer{fooReplacer, intReplacer}
		fn := CreateValueReplacer(cfg, nil)
		schema := CreateSchemaFromString(t, `{"type": "integer"}`)
		res := fn(schema, nil)
		assert.Equal(int64(1), res)
	})

	t.Run("with-force-null", func(t *testing.T) {
		cfg := NewDefaultConfig("")
		cfg.Replacers = []Replacer{forceNullReplacer, fooReplacer}
		fn := CreateValueReplacer(cfg, nil)
		res := fn("", nil)
		assert.Nil(res)
	})

	t.Run("continues-on-nil", func(t *testing.T) {
		cfg := NewDefaultConfig("")
		cfg.Replacers = []Replacer{nilReplacer, fooReplacer}
		fn := CreateValueReplacer(cfg, nil)
		res := fn("foo", nil)
		assert.Equal("foo", res)
	})

	t.Run("finishes-with-nil", func(t *testing.T) {
		cfg := NewDefaultConfig("")
		cfg.Replacers = []Replacer{nilReplacer, nilReplacer}
		fn := CreateValueReplacer(cfg, nil)
		res := fn("foo", nil)
		assert.Nil(res)
	})

	t.Run("schema-with-pattern", func(t *testing.T) {
		schema := CreateSchemaFromString(t, `{"type": "string", "pattern": "^((0[1-9])|(1[0-2]))$"}`)
		cfg := NewDefaultConfig("")
		cfg.Replacers = []Replacer{func(ctx *ReplaceContext) any {
			if len(ctx.State.NamePath) == 0 {
				return "01"
			}
			return "foo"
		}}
		fn := CreateValueReplacer(cfg, nil)

		res1 := fn(schema, nil)
		assert.Equal("01", res1)

		res2 := fn(schema, &ReplaceState{NamePath: []string{"foo"}})
		assert.Nil(res2)
	})
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
