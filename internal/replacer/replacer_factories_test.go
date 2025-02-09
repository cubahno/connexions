//go:build !integration

package replacer

import (
	"testing"

	"github.com/cubahno/connexions/internal/config"
	"github.com/cubahno/connexions/internal/types"
	assert2 "github.com/stretchr/testify/assert"
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
		cfg := config.NewDefaultConfig("")
		fn := CreateValueReplacer(cfg, []Replacer{fooReplacer}, nil)
		res := fn("", nil)
		assert.Equal("foo", res)
	})

	t.Run("with-incorrect-schema-type", func(t *testing.T) {
		cfg := config.NewDefaultConfig("")
		fn := CreateValueReplacer(cfg, []Replacer{fooReplacer, intReplacer}, nil)
		schema := &types.Schema{Type: types.TypeInteger}
		res := fn(schema, nil)
		assert.Equal(int64(1), res)
	})

	t.Run("with-force-null", func(t *testing.T) {
		cfg := config.NewDefaultConfig("")
		fn := CreateValueReplacer(cfg, []Replacer{forceNullReplacer, fooReplacer}, nil)
		res := fn("", nil)
		assert.Nil(res)
	})

	t.Run("continues-on-nil", func(t *testing.T) {
		cfg := config.NewDefaultConfig("")
		fn := CreateValueReplacer(cfg, []Replacer{nilReplacer, fooReplacer}, nil)
		res := fn("foo", nil)
		assert.Equal("foo", res)
	})

	t.Run("finishes-with-nil", func(t *testing.T) {
		cfg := config.NewDefaultConfig("")
		fn := CreateValueReplacer(cfg, []Replacer{nilReplacer, nilReplacer}, nil)
		res := fn("foo", nil)
		assert.Nil(res)
	})

	t.Run("schema-with-pattern", func(t *testing.T) {
		schema := &types.Schema{
			Type:    types.TypeString,
			Pattern: "^((0[1-9])|(1[0-2]))$",
		}
		cfg := config.NewDefaultConfig("")
		replacers := []Replacer{func(ctx *ReplaceContext) any {
			if len(ctx.State.NamePath) == 0 {
				return "01"
			}
			return "foo"
		}}
		fn := CreateValueReplacer(cfg, replacers, nil)

		res1 := fn(schema, nil)
		assert.Equal("01", res1)

		res2 := fn(schema, NewReplaceStateWithName("foo"))
		// foo is invalid and valid value will be re-generated
		assert.NotNil(res2)
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
		{"foo", types.TypeString, true},
		{1, types.TypeString, false},
		{"foo", types.TypeInteger, false},
		{1, types.TypeInteger, true},
		{"1", types.TypeNumber, false},
		{1, types.TypeNumber, true},
		{1.12, types.TypeNumber, true},
		{"true", types.TypeBoolean, false},
		{true, types.TypeBoolean, true},
		{[]string{"foo", "bar"}, types.TypeArray, true},
		{[]int{1, 2}, types.TypeArray, true},
		{[]bool{true, false}, types.TypeArray, true},
		{map[string]string{"foo": "bar"}, types.TypeObject, true},
		{map[string]int{"foo": 1}, types.TypeObject, true},
		{map[string]bool{"foo": true}, types.TypeObject, true},
		{map[string]any{"foo": "bar"}, types.TypeObject, true},
		{"foo", types.TypeObject, false},
		{"foo", "bar", false},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			assert.Equal(tc.expected, IsCorrectlyReplacedType(tc.value, tc.needed))
		})
	}
}
