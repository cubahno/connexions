package xs

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"reflect"
	"sync"
	"testing"
)

func TestReplaceState(t *testing.T) {
	t.Run("NewFrom", func(t *testing.T) {
		src := &ReplaceState{
			NamePath: []string{"foo", "bar"},
		}
		wanted := &ReplaceState{
			NamePath:                 []string{"foo", "bar"},
			IsHeader:                 false,
			ContentType:              "",
			stopCircularArrayTripOn:  0,
			stopCircularObjectTripOn: "",
		}
		if got := src.NewFrom(src); !reflect.DeepEqual(got, wanted) {
			t.Errorf("NewFrom() = %v, expected %v", got, wanted)
		}
	})

	t.Run("WithName", func(t *testing.T) {
		src := &ReplaceState{
			NamePath: []string{"dice", "nice"},
		}

		res := src.WithName("mice")
		assert.Equal(t, []string{"dice", "nice", "mice"}, res.NamePath)
	})

	t.Run("WithElementIndex", func(t *testing.T) {
		src := &ReplaceState{}

		res := src.WithElementIndex(10)
		assert.Equal(t, 10, res.ElementIndex)
	})

	t.Run("WithElementIndexRacing", func(t *testing.T) {
		const numGoroutines = 1000
		const targetValue = 42

		// Create a shared ReplaceState
		state := &ReplaceState{}

		// Use a WaitGroup to wait for all goroutines to finish
		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		// Start multiple goroutines that concurrently call WithElementIndex
		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				state.WithElementIndex(targetValue)
			}()
		}

		// Wait for all goroutines to finish
		wg.Wait()

		if state.ElementIndex != targetValue || state.stopCircularArrayTripOn != targetValue+1 {
			t.Errorf("State not consistent: ElementIndex = %d, stopCircularArrayTripOn = %d", state.ElementIndex, state.stopCircularArrayTripOn)
		}
	})

	t.Run("WithHeader", func(t *testing.T) {
		src := &ReplaceState{}

		res := src.WithHeader()
		assert.True(t, res.IsHeader)
	})

	t.Run("WithContentType", func(t *testing.T) {
		src := &ReplaceState{}

		res := src.WithContentType("application/json")
		assert.Equal(t, "application/json", res.ContentType)
	})

	t.Run("IsCircularObjectTrip", func(t *testing.T) {
		src := &ReplaceState{}

		res := src.WithName("foo").WithName("bar")
		assert.True(t, res.IsCircularObjectTrip())
	})

	t.Run("IsCircularArrayTrip", func(t *testing.T) {
		src := &ReplaceState{}

		res := src.WithElementIndex(10)
		assert.True(t, res.IsCircularArrayTrip(10))
	})
}

func TestCreateValueSchemaReplacer(t *testing.T) {
	fn := CreateValueSchemaReplacerFactory()(&Resource{})

	t.Run("from-example", func(t *testing.T) {
		schema := CreateSchemaFromString(t, `{"type": "string", "example": "foo"}`)
		res := fn(schema, nil)
		assert.Equal(t, "foo", res)
	})

	t.Run("string", func(t *testing.T) {
		schema := CreateSchemaFromString(t, `{"type": "string"}`)
		res := fn(schema, nil)

		v, ok := res.(string)
		assert.True(t, ok)
		assert.Greater(t, len(v), 0)
	})

	t.Run("integer", func(t *testing.T) {
		schema := CreateSchemaFromString(t, `{"type": "integer"}`)
		res := fn(schema, nil)

		v, ok := res.(uint32)
		assert.True(t, ok)
		assert.Greater(t, v, uint32(0))
	})

	t.Run("number", func(t *testing.T) {
		schema := CreateSchemaFromString(t, `{"type": "number"}`)
		res := fn(schema, nil)

		v, ok := res.(uint32)
		assert.True(t, ok)
		assert.Greater(t, v, uint32(0))
	})

	t.Run("boolean", func(t *testing.T) {
		schema := CreateSchemaFromString(t, `{"type": "boolean"}`)
		res := fn(schema, nil)

		_, ok := res.(bool)
		assert.True(t, ok)
	})

	t.Run("unknown", func(t *testing.T) {
		schema := CreateSchemaFromString(t, `{"type": "x"}`)
		res := fn(schema, nil)
		assert.Nil(t, res)
	})
}

func TestIsCorrectlyReplacedType(t *testing.T) {
	type testCase struct {
		value    any
		needed   string
		expected bool
	}

	testCases := []testCase{
		{"foo", openapi3.TypeString, true},
		{1, openapi3.TypeString, false},
		{"foo", openapi3.TypeInteger, false},
		{1, openapi3.TypeInteger, true},
		{"1", openapi3.TypeNumber, false},
		{1, openapi3.TypeNumber, true},
		{1.12, openapi3.TypeNumber, true},
		{"true", openapi3.TypeBoolean, false},
		{true, openapi3.TypeBoolean, true},
		{[]string{"foo", "bar"}, openapi3.TypeArray, true},
		{[]int{1, 2}, openapi3.TypeArray, true},
		{[]bool{true, false}, openapi3.TypeArray, true},
		{map[string]string{"foo": "bar"}, openapi3.TypeObject, true},
		{map[string]int{"foo": 1}, openapi3.TypeObject, true},
		{map[string]bool{"foo": true}, openapi3.TypeObject, true},
		{map[string]any{"foo": "bar"}, openapi3.TypeObject, true},
		{"foo", openapi3.TypeObject, false},
		{"foo", "bar", false},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			assert.Equal(t, tc.expected, IsCorrectlyReplacedType(tc.value, tc.needed))
		})
	}
}
