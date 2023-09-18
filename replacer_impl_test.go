//go:build !integration

package connexions

import (
	"fmt"
	assert2 "github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestHasCorrectSchemaType(t *testing.T) {
	assert := assert2.New(t)

	t.Run("nil-schema", func(t *testing.T) {
		res := HasCorrectSchemaValue(NewReplaceContext(nil, nil, nil), "nice")
		assert.True(res)
	})

	t.Run("not-a-schema", func(t *testing.T) {
		res := HasCorrectSchemaValue(NewReplaceContext("not-a-schema", nil, nil), "nice")
		assert.True(res)
	})

	t.Run("string-type-ok", func(t *testing.T) {
		schema := &Schema{Type: TypeString}
		res := HasCorrectSchemaValue(NewReplaceContext(schema, nil, nil), "nice")
		assert.True(res)
	})

	t.Run("string-type-error", func(t *testing.T) {
		schema := &Schema{Type: TypeString}
		res := HasCorrectSchemaValue(NewReplaceContext(schema, nil, nil), 123)
		assert.False(res)
	})

	t.Run("int32-ok", func(t *testing.T) {
		schema := &Schema{Type: TypeNumber, Format: "int32"}
		res := HasCorrectSchemaValue(NewReplaceContext(schema, nil, nil), 123)
		assert.True(res)
	})

	t.Run("int32-bad", func(t *testing.T) {
		schema := &Schema{Type: TypeNumber, Format: "int32"}
		res := HasCorrectSchemaValue(NewReplaceContext(schema, nil, nil), fake.Int64())
		assert.False(res)
	})

	t.Run("int64-ok-small", func(t *testing.T) {
		schema := &Schema{Type: TypeNumber, Format: "int64"}
		res := HasCorrectSchemaValue(NewReplaceContext(schema, nil, nil), 123)
		assert.True(res)
	})

	t.Run("int64-ok-big", func(t *testing.T) {
		schema := &Schema{Type: TypeNumber, Format: "int64"}
		res := HasCorrectSchemaValue(NewReplaceContext(schema, nil, nil), fake.Int64())
		assert.True(res)
	})

	t.Run("int64-bad", func(t *testing.T) {
		schema := &Schema{Type: TypeNumber, Format: "int64"}
		res := HasCorrectSchemaValue(NewReplaceContext(schema, nil, nil), 123.1)
		assert.False(res)
	})

	t.Run("string-date-ok", func(t *testing.T) {
		schema := &Schema{Type: TypeString, Format: "date"}
		res := HasCorrectSchemaValue(NewReplaceContext(schema, nil, nil), "2020-01-01")
		assert.True(res)
	})

	t.Run("string-date-bad", func(t *testing.T) {
		schema := &Schema{Type: TypeString, Format: "date"}
		res := HasCorrectSchemaValue(NewReplaceContext(schema, nil, nil), "2020-13-01")
		assert.False(res)
	})

	t.Run("string-datetime-ok", func(t *testing.T) {
		schema := &Schema{Type: TypeString, Format: "date-time"}
		res := HasCorrectSchemaValue(NewReplaceContext(schema, nil, nil), "2020-01-01T15:04:05.000Z")
		assert.True(res)
	})

	t.Run("string-datetime-bad", func(t *testing.T) {
		schema := &Schema{Type: TypeString, Format: "date-time"}
		res := HasCorrectSchemaValue(NewReplaceContext(schema, nil, nil), "2020-01-01T25:04:05.000Z")
		assert.False(res)
	})

	t.Run("string-with-unknown-format", func(t *testing.T) {
		schema := &Schema{Type: TypeString, Format: "x"}
		res := HasCorrectSchemaValue(NewReplaceContext(schema, nil, nil), "xxx")
		assert.True(res)
	})
}

func TestReplaceInHeaders(t *testing.T) {
	assert := assert2.New(t)

	t.Run("not-a-header", func(t *testing.T) {
		state := &ReplaceState{
			NamePath: []string{"userID"},
		}
		resource := &Resource{
			ContextData: []map[string]any{
				{
					"user_id": "1234",
				},
			},
		}
		res := ReplaceInHeaders(NewReplaceContext(nil, state, resource))
		assert.Nil(res)
	})

	t.Run("in-headers", func(t *testing.T) {
		state := &ReplaceState{
			NamePath: []string{"userID"},
			IsHeader: true,
		}
		resource := &Resource{
			ContextAreaPrefix: "in-",
			ContextData: []map[string]any{
				{
					"user_id": "1234",
					"in-header": map[string]string{
						"user_id": "5678",
					},
				},
			},
		}
		res := ReplaceInHeaders(NewReplaceContext(nil, state, resource))
		assert.Equal("5678", res)
	})
}

func TestReplaceInPath(t *testing.T) {
	assert := assert2.New(t)

	t.Run("not-a-path", func(t *testing.T) {
		state := &ReplaceState{
			NamePath:    []string{"userID"},
			IsPathParam: false,
		}
		resource := &Resource{
			ContextData: []map[string]any{
				{
					"user_id": "1234",
				},
			},
		}
		res := ReplaceInPath(NewReplaceContext(nil, state, resource))
		assert.Nil(res)
	})

	t.Run("in-path", func(t *testing.T) {
		state := &ReplaceState{
			NamePath:    []string{"userID"},
			IsPathParam: true,
		}
		resource := &Resource{
			ContextAreaPrefix: "in-",
			ContextData: []map[string]any{
				{
					"user_id": "1234",
					"in-path": map[string]string{
						"user_id": "5678",
					},
				},
			},
		}
		res := ReplaceInPath(NewReplaceContext(nil, state, resource))
		assert.Equal("5678", res)
	})

	t.Run("not-in-ctx", func(t *testing.T) {
		state := &ReplaceState{
			NamePath:    []string{"userID"},
			IsPathParam: true,
		}
		resource := &Resource{
			ContextAreaPrefix: "in-",
			ContextData: []map[string]any{
				{
					"user_id": "1234",
				},
			},
		}
		res := ReplaceInPath(NewReplaceContext(nil, state, resource))
		assert.Nil(res)
	})
}

func TestReplaceInArea(t *testing.T) {
	assert := assert2.New(t)

	t.Run("missing-prefix", func(t *testing.T) {
		state := &ReplaceState{
			NamePath:    []string{"userID"},
			IsPathParam: true,
		}
		resource := &Resource{
			ContextData: []map[string]any{
				{
					"user_id": "1234",
				},
			},
		}
		res := replaceInArea(NewReplaceContext(nil, state, resource), "path")
		assert.Nil(res)
	})
}

func TestReplaceFromContext(t *testing.T) {
	assert := assert2.New(t)

	t.Run("happy-path", func(t *testing.T) {
		schema := &Schema{
			Type: TypeString,
		}
		state := &ReplaceState{
			NamePath: []string{"Person", "dateOfBirth"},
		}
		resource := &Resource{
			ContextData: []map[string]any{
				{
					"person": map[string]any{
						"date_of_birth": "1980-01-01",
					},
				},
			},
		}
		res := ReplaceFromContext(NewReplaceContext(schema, state, resource))
		assert.Equal("1980-01-01", res)
	})

	t.Run("not-found-in-path", func(t *testing.T) {
		schema := &Schema{
			Type: TypeString,
		}
		state := &ReplaceState{
			NamePath: []string{"Person", "dateOfBirth"},
		}
		resource := &Resource{
			ContextData: []map[string]any{
				{
					"people": map[string]any{
						"date_of_birth": "1980-01-01",
					},
				},
			},
		}
		res := ReplaceFromContext(NewReplaceContext(schema, state, resource))
		assert.Nil(res)
	})
}

func TestCastToSchemaFormat(t *testing.T) {
	assert := assert2.New(t)

	t.Run("no-schema", func(t *testing.T) {
		res := CastToSchemaFormat(NewReplaceContext(nil, nil, nil), 123)
		assert.Equal(123, res)
	})

	t.Run("int32-ok", func(t *testing.T) {
		schema := &Schema{
			Type:   TypeNumber,
			Format: "int32",
		}
		res := CastToSchemaFormat(NewReplaceContext(schema, nil, nil), 123.0)
		assert.Equal(int32(123), res)
	})

	t.Run("int32-not", func(t *testing.T) {
		schema := &Schema{
			Type:   TypeNumber,
			Format: "int32",
		}
		res := CastToSchemaFormat(NewReplaceContext(schema, nil, nil), 123.4)
		assert.Equal(123.4, res)
	})

	t.Run("int64-ok", func(t *testing.T) {
		schema := &Schema{
			Type:   TypeNumber,
			Format: "int64",
		}
		res := CastToSchemaFormat(NewReplaceContext(schema, nil, nil), 123.0)
		assert.Equal(int64(123), res)
	})

	t.Run("int64-not", func(t *testing.T) {
		schema := &Schema{
			Type:   TypeNumber,
			Format: "int64",
		}
		res := CastToSchemaFormat(NewReplaceContext(schema, nil, nil), 123.4)
		assert.Equal(123.4, res)
	})
}

func TestReplaceValueWithContext(t *testing.T) {
	assert := assert2.New(t)

	t.Run("happy-path", func(t *testing.T) {
		context := map[string]interface{}{
			"user": map[string]interface{}{
				"age": 30,
				"country": map[string]interface{}{
					"name": "Germany",
					"code": "DE",
				},
			},
		}
		namePath := []string{"user", "country", "name"}
		res := ReplaceValueWithContext(namePath, context)

		assert.Equal("Germany", res)
	})

	t.Run("happy-path-with-ints", func(t *testing.T) {
		context := map[string]any{
			"user": map[string]any{
				"name": "Jane Doe",
				"age":  30,
			},
		}
		namePath := []string{"user", "age"}
		res := ReplaceValueWithContext(namePath, context)

		assert.Equal(30, res)
	})

	t.Run("unmapped-type", func(t *testing.T) {
		namePath := []string{"rank"}
		ctx := map[string]int64{
			"rank": 123,
		}
		res := ReplaceValueWithContext(namePath, ctx)
		assert.Nil(res)
	})

	t.Run("has-name-prefix", func(t *testing.T) {
		context := map[string]interface{}{
			"user": map[string]interface{}{
				"country": map[string]interface{}{
					"^name": "Germany",
				},
			},
		}
		namePath := []string{"user", "country", "name"}
		res := ReplaceValueWithContext(namePath, context)

		assert.Equal("Germany", res)
	})

	t.Run("single-namepath-has-name-prefix", func(t *testing.T) {
		context := map[string]interface{}{
			"^name": "Jane Doe",
		}
		namePath := []string{"name"}
		res := ReplaceValueWithContext(namePath, context)

		assert.Equal("Jane Doe", res)
	})

	t.Run("random-slice-value", func(t *testing.T) {
		names := []string{"Jane", "John", "Zena"}
		context := map[string]interface{}{
			"user": map[string]interface{}{
				"name": names,
			},
		}
		namePath := []string{"user", "name"}
		res := ReplaceValueWithContext(namePath, context)

		assert.Contains(names, res)
	})

	t.Run("with-map-of-strings-ctx", func(t *testing.T) {
		context := map[string]string{
			"name": "Jane Doe",
		}
		namePath := []string{"name"}
		res := ReplaceValueWithContext(namePath, context)

		assert.Equal("Jane Doe", res)
	})

	t.Run("with-map-of-ints-ctx", func(t *testing.T) {
		context := map[string]int{
			"age": 30,
		}
		namePath := []string{"name", "age"}
		res := ReplaceValueWithContext(namePath, context)

		assert.Equal(30, res)
	})

	t.Run("with-map-of-float64s-ctx", func(t *testing.T) {
		id := float64(123)
		context := map[string]float64{
			"rank": id,
		}
		namePath := []string{"name", "rank"}
		res := ReplaceValueWithContext(namePath, context)

		assert.Equal(id, res)
	})

	t.Run("with-map-of-bools-ctx", func(t *testing.T) {
		context := map[string]bool{
			"is_married": true,
		}
		namePath := []string{"name", "is_married"}
		res := ReplaceValueWithContext(namePath, context)

		assert.Equal(true, res)
	})

	t.Run("with-fake-func-ctx", func(t *testing.T) {
		fn := FakeFunc(func() MixedValue {
			return IntValue(123)
		})
		namePath := []string{"name", "rank"}
		res := ReplaceValueWithContext(namePath, fn)

		assert.Equal(int64(123), res)
	})

	t.Run("with-string-ctx", func(t *testing.T) {
		namePath := []string{"name"}
		res := ReplaceValueWithContext(namePath, "Jane")
		assert.Equal("Jane", res)
	})

	t.Run("with-int-ctx", func(t *testing.T) {
		namePath := []string{"age"}
		res := ReplaceValueWithContext(namePath, 30)
		assert.Equal(30, res)
	})

	t.Run("with-float64-ctx", func(t *testing.T) {
		namePath := []string{"rank"}
		res := ReplaceValueWithContext(namePath, 123.0)
		assert.Equal(123.0, res)
	})

	t.Run("with-bool-ctx", func(t *testing.T) {
		namePath := []string{"is_married"}
		res := ReplaceValueWithContext(namePath, true)
		assert.Equal(true, res)
	})

	t.Run("with-string-slice-ctx", func(t *testing.T) {
		namePath := []string{"name"}
		values := []string{"Jane", "John"}
		res := ReplaceValueWithContext(namePath, values)
		assert.Contains(values, res)
	})

	t.Run("with-int-slice-ctx", func(t *testing.T) {
		namePath := []string{"age"}
		values := []int{30, 40}
		res := ReplaceValueWithContext(namePath, values)
		assert.Contains(values, res)
	})

	t.Run("with-bool-slice-ctx", func(t *testing.T) {
		namePath := []string{"is_married"}
		values := []bool{true, false}
		res := ReplaceValueWithContext(namePath, values)
		assert.Contains(values, res)
	})

	t.Run("with-float64-slice-ctx", func(t *testing.T) {
		namePath := []string{"rank"}
		values := []float64{123.0, 1.0, 12.0}
		res := ReplaceValueWithContext(namePath, values)
		assert.Contains(values, res)
	})

	t.Run("with-any-slice-ctx", func(t *testing.T) {
		namePath := []string{"nickname"}
		values := []any{"j", 1}
		res := ReplaceValueWithContext(namePath, values)
		assert.Contains(values, res)
	})
}

func TestReplaceValueWithMapContext(t *testing.T) {
	assert := assert2.New(t)

	t.Run("empty-path", func(t *testing.T) {
		res := replaceValueWithMapContext[string]([]string{}, map[string]string{})
		assert.Nil(res)
	})

	t.Run("direct-match", func(t *testing.T) {
		path := []string{"name"}
		data := map[string]string{
			"name": "Jane Doe",
		}
		res := replaceValueWithMapContext[string](path, data)
		assert.Equal("Jane Doe", res)
	})

	t.Run("no-match", func(t *testing.T) {
		path := []string{"user"}
		data := map[string]string{
			"name": "Jane Doe",
		}
		res := replaceValueWithMapContext[string](path, data)
		assert.Nil(res)
	})
}

func TestReplaceFromSchemaFormat(t *testing.T) {
	assert := assert2.New(t)

	t.Run("not-a-schema", func(t *testing.T) {
		res := ReplaceFromSchemaFormat(NewReplaceContext("not-a-schema", nil, nil))
		assert.Nil(res)
	})

	t.Run("unknown-format", func(t *testing.T) {
		schema := &Schema{
			Format: "my-format",
		}
		res := ReplaceFromSchemaFormat(NewReplaceContext(schema, nil, nil))
		assert.Nil(res)
	})

	t.Run("date", func(t *testing.T) {
		schema := &Schema{
			Format: "date",
		}
		res := ReplaceFromSchemaFormat(NewReplaceContext(schema, nil, nil))
		assert.NotNil(res)
		value, _ := res.(string)
		assert.Equal(len(value), 10)
	})

	t.Run("date-time", func(t *testing.T) {
		schema := &Schema{
			Format: "date-time",
		}
		res := ReplaceFromSchemaFormat(NewReplaceContext(schema, nil, nil))
		assert.NotNil(res)
		value, _ := res.(string)
		assert.Equal(len(value), 24)
	})

	t.Run("email", func(t *testing.T) {
		schema := &Schema{
			Format: "email",
		}
		res := ReplaceFromSchemaFormat(NewReplaceContext(schema, nil, nil))
		assert.NotNil(res)
		value, _ := res.(string)
		assert.Greater(len(value), 6)
		assert.Contains(value, "@")
	})

	t.Run("uuid", func(t *testing.T) {
		schema := &Schema{
			Format: "uuid",
		}
		res := ReplaceFromSchemaFormat(NewReplaceContext(schema, nil, nil))
		assert.NotNil(res)
		value, _ := res.(string)
		assert.Equal(len(value), 36)
	})

	t.Run("password", func(t *testing.T) {
		schema := &Schema{
			Format: "password",
		}
		res := ReplaceFromSchemaFormat(NewReplaceContext(schema, nil, nil))
		assert.NotNil(res)
		value, _ := res.(string)
		assert.GreaterOrEqual(len(value), 6)
	})

	t.Run("hostname", func(t *testing.T) {
		schema := &Schema{
			Format: "hostname",
		}
		res := ReplaceFromSchemaFormat(NewReplaceContext(schema, nil, nil))
		assert.NotNil(res)
		value, _ := res.(string)
		assert.Greater(len(value), 6)
		assert.Contains(value, ".")
	})

	t.Run("url", func(t *testing.T) {
		schema := &Schema{
			Format: "url",
		}
		res := ReplaceFromSchemaFormat(NewReplaceContext(schema, nil, nil))
		assert.NotNil(res)
		value, _ := res.(string)
		assert.Greater(len(value), 6)
		assert.Contains(value, ".")
		assert.True(strings.HasPrefix(value, "http"))
	})

	t.Run("int32", func(t *testing.T) {
		schema := &Schema{
			Format: "int32",
		}
		res := ReplaceFromSchemaFormat(NewReplaceContext(schema, nil, nil))
		assert.NotNil(res)
		println("res:", res, "type:", fmt.Sprintf("%T", res), "value:", fmt.Sprintf("%v", res))
		v, ok := ToInt32(res)
		assert.True(ok)
		assert.Greater(v, int32(0))
	})

	t.Run("int64", func(t *testing.T) {
		schema := &Schema{
			Format: "int64",
		}
		res := ReplaceFromSchemaFormat(NewReplaceContext(schema, nil, nil))
		assert.NotNil(res)
		v, ok := ToInt64(res)
		assert.True(ok)
		assert.Greater(v, int64(0))
	})
}

func TestReplaceFromSchemaPrimitive(t *testing.T) {
	assert := assert2.New(t)

	t.Run("not-a-schema", func(t *testing.T) {
		res := ReplaceFromSchemaPrimitive(NewReplaceContext("not-a-schema", nil, nil))
		assert.Nil(res)
	})

	t.Run("string", func(t *testing.T) {
		schema := &Schema{Type: TypeString}
		res := ReplaceFromSchemaPrimitive(NewReplaceContext(schema, nil, nil))
		value, _ := res.(string)
		assert.Greater(len(value), 0)
	})

	t.Run("integer", func(t *testing.T) {
		schema := &Schema{Type: TypeInteger}
		res := ReplaceFromSchemaPrimitive(NewReplaceContext(schema, nil, nil))
		_, ok := res.(uint32)
		assert.True(ok)
	})

	t.Run("number", func(t *testing.T) {
		schema := &Schema{Type: TypeNumber}
		res := ReplaceFromSchemaPrimitive(NewReplaceContext(schema, nil, nil))
		_, ok := res.(uint32)
		assert.True(ok)
	})

	t.Run("boolean", func(t *testing.T) {
		schema := &Schema{Type: TypeBoolean}
		res := ReplaceFromSchemaPrimitive(NewReplaceContext(schema, nil, nil))
		_, ok := res.(bool)
		assert.True(ok)
	})

	t.Run("other", func(t *testing.T) {
		schema := &Schema{Type: TypeObject}
		res := ReplaceFromSchemaPrimitive(NewReplaceContext(schema, nil, nil))
		assert.Nil(res)
	})

}

func TestReplaceFromSchemaExample(t *testing.T) {
	assert := assert2.New(t)

	t.Run("not-a-schema", func(t *testing.T) {
		res := ReplaceFromSchemaExample(NewReplaceContext("not-a-schema", nil, nil))
		assert.Nil(res)
	})

	t.Run("with-a-schema", func(t *testing.T) {
		schema := &Schema{Example: "hallo, welt!"}
		res := ReplaceFromSchemaExample(NewReplaceContext(schema, nil, nil))
		assert.Equal("hallo, welt!", res)
	})
}

func TestApplySchemaConstraints(t *testing.T) {
	assert := assert2.New(t)

	t.Run("nil-schema", func(t *testing.T) {
		res := ApplySchemaConstraints(nil, "some-value")
		assert.Equal("some-value", res)
	})

	t.Run("not-a-schema", func(t *testing.T) {
		res := ApplySchemaConstraints("not-a-schema", "some-value")
		assert.Equal("some-value", res)
	})

	t.Run("case-not-applied", func(t *testing.T) {
		schema := &Schema{Type: TypeBoolean}
		res := ApplySchemaConstraints(schema, true)
		assert.Equal(true, res)
	})

	t.Run("number-conv-fails", func(t *testing.T) {
		schema := &Schema{Type: TypeNumber}
		res := ApplySchemaConstraints(schema, "abc")
		assert.Nil(res)
	})

	t.Run("int-conv-fails", func(t *testing.T) {
		schema := &Schema{Type: TypeInteger}
		res := ApplySchemaConstraints(schema, "abc")
		assert.Nil(res)
	})

	t.Run("string-ok", func(t *testing.T) {
		schema := &Schema{Type: TypeString, MinLength: 5}
		res := ApplySchemaConstraints(schema, "hallo, welt!")
		assert.Equal("hallo, welt!", res)
	})

	t.Run("number-ok", func(t *testing.T) {
		schema := &Schema{Type: TypeNumber, Minimum: 100}
		res := ApplySchemaConstraints(schema, 133)
		assert.Equal(133.0, res)
	})

	t.Run("int-ok", func(t *testing.T) {
		schema := &Schema{Type: TypeInteger, Maximum: 10}
		res := ApplySchemaConstraints(schema, 6)
		assert.Equal(int64(6), res)
	})
}

func TestApplySchemaStringConstraints(t *testing.T) {
	assert := assert2.New(t)

	t.Run("nil-schema", func(t *testing.T) {
		res := applySchemaStringConstraints(nil, "some-value")
		assert.Equal("some-value", res)
	})

	t.Run("no-constraints", func(t *testing.T) {
		schema := &Schema{Type: TypeString}
		res := applySchemaStringConstraints(schema, "hallo welt!")
		assert.Equal("hallo welt!", res)
	})

	t.Run("pattern-ok", func(t *testing.T) {
		schema := &Schema{
			Type:    TypeString,
			Pattern: "^[0-9]{2}[a-z]+$",
		}
		res := applySchemaStringConstraints(schema, "12go")
		assert.Equal("12go", res)
	})

	t.Run("pattern-fails", func(t *testing.T) {
		schema := &Schema{
			Type:    TypeString,
			Pattern: "^[0-9]{2}$",
		}
		res := applySchemaStringConstraints(schema, "12go")
		assert.Nil(res)
	})

	t.Run("enum-ok", func(t *testing.T) {
		schema := &Schema{
			Type: TypeString,
			Enum: []any{
				"nice",
				"rice",
				"dice",
			},
		}
		res := applySchemaStringConstraints(schema, "dice")
		assert.Equal("dice", res)
	})

	t.Run("enum-applied", func(t *testing.T) {
		enum := []any{
			"nice",
			"rice",
			"dice",
		}
		schema := &Schema{
			Type: TypeString,
			Enum: enum,
		}
		res := applySchemaStringConstraints(schema, "mice")
		assert.Contains(enum, res)
	})

	t.Run("min-length-ok", func(t *testing.T) {
		schema := &Schema{
			Type:      TypeString,
			MinLength: 5,
		}
		res := applySchemaStringConstraints(schema, "hallo")
		assert.Equal("hallo", res)
	})

	t.Run("min-length-applied", func(t *testing.T) {
		schema := &Schema{
			Type:      TypeString,
			MinLength: 5,
		}
		res := applySchemaStringConstraints(schema, "ha")
		assert.Equal("ha---", res)
	})

	t.Run("max-length-ok", func(t *testing.T) {
		schema := &Schema{
			Type:      TypeString,
			MaxLength: 5,
		}
		res := applySchemaStringConstraints(schema, "hallo")
		assert.Equal("hallo", res)
	})

	t.Run("max-length-applied", func(t *testing.T) {
		schema := &Schema{
			Type:      TypeString,
			MaxLength: 5,
		}
		res := applySchemaStringConstraints(schema, "hallo welt!")
		assert.Equal("hallo", res)
	})
}

func TestApplySchemaNumberConstraints(t *testing.T) {
	assert := assert2.New(t)

	t.Run("nil-schema", func(t *testing.T) {
		res := applySchemaNumberConstraints(nil, 123)
		assert.Equal(123.0, res)
	})

	t.Run("no-constraints", func(t *testing.T) {
		schema := &Schema{Type: TypeNumber}
		res := applySchemaNumberConstraints(schema, 123)
		assert.Equal(123.0, res)
	})

	t.Run("min-ok", func(t *testing.T) {
		schema := &Schema{Type: TypeNumber, Minimum: 100}
		res := applySchemaNumberConstraints(schema, 100)
		assert.Equal(100.0, res)
	})

	t.Run("min-applied", func(t *testing.T) {
		schema := &Schema{Type: TypeNumber, Minimum: 100}
		res := applySchemaNumberConstraints(schema, 99)
		assert.Equal(100.0, res)
	})

	t.Run("max-ok", func(t *testing.T) {
		schema := &Schema{Type: TypeNumber, Maximum: 100}
		res := applySchemaNumberConstraints(schema, 100)
		assert.Equal(100.0, res)
	})

	t.Run("max-applied", func(t *testing.T) {
		schema := &Schema{Type: TypeNumber, Maximum: 100}
		res := applySchemaNumberConstraints(schema, 123)
		assert.Equal(100.0, res)
	})

	t.Run("mult-of-ok", func(t *testing.T) {
		schema := &Schema{Type: TypeNumber, MultipleOf: 5}
		res := applySchemaNumberConstraints(schema, 15)
		assert.Equal(15.0, res)
	})

	t.Run("mult-of-applied", func(t *testing.T) {
		schema := &Schema{Type: TypeNumber, MultipleOf: 3}
		res := applySchemaNumberConstraints(schema, 100)
		assert.Equal(99.0, res)
	})

	t.Run("min-max-mult-of-applied", func(t *testing.T) {
		schema := &Schema{
			Type:       TypeNumber,
			MultipleOf: 3,
			Minimum:    12,
			Maximum:    21,
		}
		res := applySchemaNumberConstraints(schema, 100)
		assert.Equal(21.0, res)
	})

	t.Run("enum-ints", func(t *testing.T) {
		schema := &Schema{
			Type: TypeNumber,
			Enum: []any{10, 20, 30},
		}
		res := applySchemaNumberConstraints(schema, 100)
		assert.Contains([]float64{10, 20, 30}, res)
	})

	t.Run("enum-floats", func(t *testing.T) {
		schema := &Schema{
			Type: TypeNumber,
			Enum: []any{10.1, 20.2, 30.3},
		}
		res := applySchemaNumberConstraints(schema, 100)
		assert.Contains([]float64{10.1, 20.2, 30.3}, res)
	})
}

func TestReplaceFromSchemaFallback(t *testing.T) {
	assert := assert2.New(t)

	t.Run("not-a-schema", func(t *testing.T) {
		res := ReplaceFromSchemaFallback(NewReplaceContext("not-a-schema", nil, nil))
		assert.Nil(res)
	})

	t.Run("with-a-schema", func(t *testing.T) {
		schema := &Schema{Default: "hallo, welt!"}
		res := ReplaceFromSchemaFallback(NewReplaceContext(schema, nil, nil))
		assert.Equal("hallo, welt!", res)
	})
}
