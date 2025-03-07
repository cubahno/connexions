//go:build !integration

package replacer

import (
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/cubahno/connexions/internal/context"
	"github.com/cubahno/connexions/internal/types"
	"github.com/jaswdr/faker/v2"
	assert2 "github.com/stretchr/testify/assert"
)

func TestHasCorrectSchemaType(t *testing.T) {
	assert := assert2.New(t)
	fake := faker.New()

	t.Run("nil-schema", func(t *testing.T) {
		res := HasCorrectSchemaValue(NewTestReplaceContext(nil), "nice")
		assert.True(res)
	})

	t.Run("not-a-schema", func(t *testing.T) {
		res := HasCorrectSchemaValue(NewTestReplaceContext("not-a-schema"), "nice")
		assert.True(res)
	})

	t.Run("string-type-ok", func(t *testing.T) {
		schema := &types.Schema{Type: types.TypeString}
		res := HasCorrectSchemaValue(NewTestReplaceContext(schema), "nice")
		assert.True(res)
	})

	t.Run("string-type-error", func(t *testing.T) {
		schema := &types.Schema{Type: types.TypeString}
		res := HasCorrectSchemaValue(NewTestReplaceContext(schema), 123)
		assert.False(res)
	})

	t.Run("int32-ok", func(t *testing.T) {
		schema := &types.Schema{Type: types.TypeNumber, Format: "int32"}
		res := HasCorrectSchemaValue(NewTestReplaceContext(schema), 123)
		assert.True(res)
	})

	t.Run("int32-bad", func(t *testing.T) {
		schema := &types.Schema{Type: types.TypeNumber, Format: "int32"}
		res := HasCorrectSchemaValue(NewTestReplaceContext(schema), fake.Int64())
		assert.False(res)
	})

	t.Run("int64-ok-small", func(t *testing.T) {
		schema := &types.Schema{Type: types.TypeNumber, Format: "int64"}
		res := HasCorrectSchemaValue(NewTestReplaceContext(schema), 123)
		assert.True(res)
	})

	t.Run("int64-ok-big", func(t *testing.T) {
		schema := &types.Schema{Type: types.TypeNumber, Format: "int64"}
		res := HasCorrectSchemaValue(NewTestReplaceContext(schema), fake.Int64())
		assert.True(res)
	})

	t.Run("int64-bad", func(t *testing.T) {
		schema := &types.Schema{Type: types.TypeNumber, Format: "int64"}
		res := HasCorrectSchemaValue(NewTestReplaceContext(schema), 123.1)
		assert.False(res)
	})

	t.Run("string-date-ok", func(t *testing.T) {
		schema := &types.Schema{Type: types.TypeString, Format: "date"}
		res := HasCorrectSchemaValue(NewTestReplaceContext(schema), "2020-01-01")
		assert.True(res)
	})

	t.Run("string-date-bad", func(t *testing.T) {
		schema := &types.Schema{Type: types.TypeString, Format: "date"}
		res := HasCorrectSchemaValue(NewTestReplaceContext(schema), "2020-13-01")
		assert.False(res)
	})

	t.Run("string-datetime-ok", func(t *testing.T) {
		schema := &types.Schema{Type: types.TypeString, Format: "date-time"}
		res := HasCorrectSchemaValue(NewTestReplaceContext(schema), "2020-01-01T15:04:05.000Z")
		assert.True(res)
	})

	t.Run("string-datetime-bad", func(t *testing.T) {
		schema := &types.Schema{Type: types.TypeString, Format: "date-time"}
		res := HasCorrectSchemaValue(NewTestReplaceContext(schema), "2020-01-01T25:04:05.000Z")
		assert.False(res)
	})

	t.Run("string-with-unknown-format", func(t *testing.T) {
		schema := &types.Schema{Type: types.TypeString, Format: "x"}
		res := HasCorrectSchemaValue(NewTestReplaceContext(schema), "xxx")
		assert.True(res)
	})
}

func TestReplaceInHeaders(t *testing.T) {
	assert := assert2.New(t)
	fake := faker.New()

	t.Run("not-a-header", func(t *testing.T) {
		state := NewReplaceStateWithName("userID")
		res := ReplaceInHeaders(&ReplaceContext{
			Faker: fake,
			State: state,
			Data: []map[string]any{
				{
					"user_id": "1234",
				},
			},
		})
		assert.Nil(res)
	})

	t.Run("basic authorization header uses corresponding context with base64", func(t *testing.T) {
		state := NewReplaceState(WithName("authorization"), WithHeader())
		res := ReplaceInHeaders(&ReplaceContext{
			Faker:      fake,
			State:      state,
			AreaPrefix: "in-",
			Data: []map[string]any{
				{
					"in-header": map[string]any{
						"authorization": "user:password",
					},
				},
			},
			Schema: &types.Schema{
				Type:   types.TypeString,
				Format: "basic",
			},
		})
		assert.Equal("Basic dXNlcjpwYXNzd29yZA==", res)
	})

	t.Run("basic authorization header without context uses random password", func(t *testing.T) {
		state := NewReplaceState(WithName("authorization"), WithHeader())
		res, ok := ReplaceInHeaders(&ReplaceContext{
			Faker: fake,
			State: state,
			Schema: &types.Schema{
				Type:   types.TypeString,
				Format: "basic",
			},
		}).(string)

		assert.True(ok)
		assert.True(strings.HasPrefix(res, "Basic "))
		assert.Greater(len(res), 10)
	})

	t.Run("bearer authorization header uses corresponding context", func(t *testing.T) {
		state := NewReplaceState(WithName("authorization"), WithHeader())
		res := ReplaceInHeaders(&ReplaceContext{
			Faker:      fake,
			State:      state,
			AreaPrefix: "in-",
			Data: []map[string]any{
				{
					"in-header": map[string]any{
						"authorization": "token",
					},
				},
			},
			Schema: &types.Schema{
				Type:   types.TypeString,
				Format: "bearer",
			},
		})
		assert.Equal("Bearer token", res)
	})

	t.Run("bearer authorization header without context uses random password", func(t *testing.T) {
		state := NewReplaceState(WithName("authorization"), WithHeader())
		res, ok := ReplaceInHeaders(&ReplaceContext{
			Faker: fake,
			State: state,
			Schema: &types.Schema{
				Type:   types.TypeString,
				Format: "bearer",
			},
		}).(string)

		assert.True(ok)
		assert.True(strings.HasPrefix(res, "Bearer "))
		assert.Greater(len(res), 10)
	})

	t.Run("custom authorization header uses value from the context", func(t *testing.T) {
		state := NewReplaceState(WithName("authorization"), WithHeader())
		res := ReplaceInHeaders(&ReplaceContext{
			Faker:      fake,
			State:      state,
			AreaPrefix: "in-",
			Data: []map[string]any{
				{
					"in-header": map[string]any{
						"authorization": "custom-token",
					},
				},
			},
			Schema: &types.Schema{
				Type:   types.TypeString,
				Format: "custom",
			},
		}).(string)

		assert.Equal("custom-token", res)
	})

	t.Run("custom header uses corresponding context value", func(t *testing.T) {
		state := NewReplaceStateWithName("userID").WithOptions(WithHeader())
		res := ReplaceInHeaders(&ReplaceContext{
			Faker:      fake,
			State:      state,
			AreaPrefix: "in-",
			Data: []map[string]any{
				{
					"user_id": "1234",
					"in-header": map[string]string{
						"user_id": "5678",
					},
				},
			},
		})
		assert.Equal("5678", res)
	})
}

func TestReplaceInPath(t *testing.T) {
	assert := assert2.New(t)
	fake := faker.New()

	t.Run("not-a-path", func(t *testing.T) {
		state := NewReplaceStateWithName("userID")
		res := ReplaceInPath(&ReplaceContext{
			Faker: fake,
			State: state,
			Data: []map[string]any{
				{
					"user_id": "1234",
				},
			},
		})
		assert.Nil(res)
	})

	t.Run("in-path", func(t *testing.T) {
		state := NewReplaceStateWithName("userID").WithOptions(WithPath())
		res := ReplaceInPath(&ReplaceContext{
			Faker:      fake,
			State:      state,
			AreaPrefix: "in-",
			Data: []map[string]any{
				{
					"user_id": "1234",
					"in-path": map[string]string{
						"user_id": "5678",
					},
				},
			},
		})
		assert.Equal("5678", res)
	})

	t.Run("not-in-ctx", func(t *testing.T) {
		state := NewReplaceStateWithName("userID").WithOptions(WithPath())
		res := ReplaceInPath(&ReplaceContext{
			Faker:      fake,
			State:      state,
			AreaPrefix: "in-",
			Data: []map[string]any{
				{
					"user_id": "1234",
				},
			},
		})
		assert.Nil(res)
	})
}

func TestReplaceInArea(t *testing.T) {
	assert := assert2.New(t)
	fake := faker.New()

	t.Run("missing-prefix", func(t *testing.T) {
		state := NewReplaceStateWithName("userID").WithOptions(WithPath())
		res := replaceInArea(&ReplaceContext{
			Faker: fake,
			State: state,
			Data: []map[string]any{
				{
					"user_id": "1234",
				},
			},
		}, "path")
		assert.Nil(res)
	})
}

func TestReplaceFromContext(t *testing.T) {
	assert := assert2.New(t)
	fake := faker.New()

	t.Run("happy-path", func(t *testing.T) {
		schema := &types.Schema{
			Type: types.TypeString,
		}
		state := NewReplaceStateWithName("Person").WithOptions(WithName("dateOfBirth"))

		res := ReplaceFromContext(&ReplaceContext{
			Faker:  fake,
			Schema: schema,
			State:  state,
			Data: []map[string]any{
				{
					"person": map[string]any{
						"date_of_birth": "1980-01-01",
					},
				},
			},
		})
		assert.Equal("1980-01-01", res)
	})

	t.Run("not-found-in-path", func(t *testing.T) {
		schema := &types.Schema{
			Type: types.TypeString,
		}
		state := NewReplaceStateWithName("Person").WithOptions(WithName("dateOfBirth"))
		res := ReplaceFromContext(&ReplaceContext{
			Faker:  fake,
			Schema: schema,
			State:  state,
			Data: []map[string]any{
				{
					"people": map[string]any{
						"date_of_birth": "1980-01-01",
					},
				},
			},
		})
		assert.Nil(res)
	})
}

func TestCastToSchemaFormat(t *testing.T) {
	assert := assert2.New(t)

	t.Run("no-schema", func(t *testing.T) {
		res := CastToSchemaFormat(NewTestReplaceContext(nil), 123)
		assert.Equal(123, res)
	})

	t.Run("int32-ok", func(t *testing.T) {
		schema := &types.Schema{
			Type:   types.TypeNumber,
			Format: "int32",
		}
		res := CastToSchemaFormat(NewTestReplaceContext(schema), 123.0)
		assert.Equal(int32(123), res)
	})

	t.Run("int32-not", func(t *testing.T) {
		schema := &types.Schema{
			Type:   types.TypeNumber,
			Format: "int32",
		}
		res := CastToSchemaFormat(NewTestReplaceContext(schema), 123.4)
		assert.Equal(123.4, res)
	})

	t.Run("int64-ok", func(t *testing.T) {
		schema := &types.Schema{
			Type:   types.TypeNumber,
			Format: "int64",
		}
		res := CastToSchemaFormat(NewTestReplaceContext(schema), 123.0)
		assert.Equal(int64(123), res)
	})

	t.Run("int64-not", func(t *testing.T) {
		schema := &types.Schema{
			Type:   types.TypeNumber,
			Format: "int64",
		}
		res := CastToSchemaFormat(NewTestReplaceContext(schema), 123.4)
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
		fn := context.FakeFunc(func() context.MixedValue {
			return context.IntValue(123)
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
		res := ReplaceFromSchemaFormat(NewTestReplaceContext("not-a-schema"))
		assert.Nil(res)
	})

	t.Run("unknown-format", func(t *testing.T) {
		schema := &types.Schema{
			Format: "my-format",
		}
		res := ReplaceFromSchemaFormat(NewTestReplaceContext(schema))
		assert.Nil(res)
	})

	t.Run("date", func(t *testing.T) {
		schema := &types.Schema{
			Format: "date",
		}
		res := ReplaceFromSchemaFormat(NewTestReplaceContext(schema))
		assert.NotNil(res)
		value, _ := res.(string)
		assert.Equal(len(value), 10)
	})

	t.Run("date-time", func(t *testing.T) {
		schema := &types.Schema{
			Format: "date-time",
		}
		res := ReplaceFromSchemaFormat(NewTestReplaceContext(schema))
		assert.NotNil(res)
		value, _ := res.(string)
		assert.Equal(len(value), 24)
	})

	t.Run("email", func(t *testing.T) {
		schema := &types.Schema{
			Format: "email",
		}
		res := ReplaceFromSchemaFormat(NewTestReplaceContext(schema))
		assert.NotNil(res)
		value, _ := res.(string)
		assert.Greater(len(value), 6)
		assert.Contains(value, "@")
	})

	t.Run("uuid", func(t *testing.T) {
		schema := &types.Schema{
			Format: "uuid",
		}
		res := ReplaceFromSchemaFormat(NewTestReplaceContext(schema))
		assert.NotNil(res)
		value, _ := res.(string)
		assert.Equal(len(value), 36)
	})

	t.Run("password", func(t *testing.T) {
		schema := &types.Schema{
			Format: "password",
		}
		res := ReplaceFromSchemaFormat(NewTestReplaceContext(schema))
		assert.NotNil(res)
		value, _ := res.(string)
		assert.GreaterOrEqual(len(value), 6)
	})

	t.Run("hostname", func(t *testing.T) {
		schema := &types.Schema{
			Format: "hostname",
		}
		res := ReplaceFromSchemaFormat(NewTestReplaceContext(schema))
		assert.NotNil(res)
		value, _ := res.(string)
		assert.Greater(len(value), 6)
		assert.Contains(value, ".")
	})

	t.Run("url", func(t *testing.T) {
		schema := &types.Schema{
			Format: "url",
		}
		res := ReplaceFromSchemaFormat(NewTestReplaceContext(schema))
		assert.NotNil(res)
		value, _ := res.(string)
		assert.Greater(len(value), 6)
		assert.Contains(value, ".")
		assert.True(strings.HasPrefix(value, "http"))
	})

	t.Run("int32", func(t *testing.T) {
		schema := &types.Schema{
			Format: "int32",
		}
		res := ReplaceFromSchemaFormat(NewTestReplaceContext(schema))
		assert.NotNil(res)

		v, ok := types.ToInt32(res)
		assert.True(ok)
		assert.Greater(v, int32(0))
	})

	t.Run("int64", func(t *testing.T) {
		schema := &types.Schema{
			Format: "int64",
		}
		res := ReplaceFromSchemaFormat(NewTestReplaceContext(schema))
		assert.NotNil(res)
		v, ok := types.ToInt64(res)
		assert.True(ok)
		assert.Greater(v, int64(0))
	})

	t.Run("ipv4", func(t *testing.T) {
		schema := &types.Schema{
			Format: "ipv4",
		}
		res := ReplaceFromSchemaFormat(NewTestReplaceContext(schema))
		assert.NotNil(res)
		value, _ := res.(string)
		assert.Greater(len(value), 6)
		addr := net.ParseIP(value)
		assert.NotNil(addr)
	})

	t.Run("ipv6", func(t *testing.T) {
		schema := &types.Schema{
			Format: "ipv6",
		}
		res := ReplaceFromSchemaFormat(NewTestReplaceContext(schema))
		assert.NotNil(res)
		value, _ := res.(string)
		assert.Greater(len(value), 6)
		addr := net.ParseIP(value)
		assert.NotNil(addr)
	})
}

func TestReplaceFromSchemaPrimitive(t *testing.T) {
	assert := assert2.New(t)

	t.Run("not-a-schema", func(t *testing.T) {
		res := ReplaceFromSchemaPrimitive(NewTestReplaceContext("not-a-schema"))
		assert.Nil(res)
	})

	t.Run("string", func(t *testing.T) {
		schema := &types.Schema{Type: types.TypeString}
		res := ReplaceFromSchemaPrimitive(NewTestReplaceContext(schema))
		value, _ := res.(string)
		assert.Greater(len(value), 0)
	})

	t.Run("integer", func(t *testing.T) {
		schema := &types.Schema{Type: types.TypeInteger}
		res := ReplaceFromSchemaPrimitive(NewTestReplaceContext(schema))
		_, ok := res.(uint32)
		assert.True(ok)
	})

	t.Run("number", func(t *testing.T) {
		schema := &types.Schema{Type: types.TypeNumber}
		res := ReplaceFromSchemaPrimitive(NewTestReplaceContext(schema))
		_, ok := res.(uint32)
		assert.True(ok)
	})

	t.Run("boolean", func(t *testing.T) {
		schema := &types.Schema{Type: types.TypeBoolean}
		res := ReplaceFromSchemaPrimitive(NewTestReplaceContext(schema))
		_, ok := res.(bool)
		assert.True(ok)
	})

	t.Run("other", func(t *testing.T) {
		schema := &types.Schema{Type: types.TypeObject}
		res := ReplaceFromSchemaPrimitive(NewTestReplaceContext(schema))
		assert.Nil(res)
	})

}

func TestReplaceFromSchemaExample(t *testing.T) {
	assert := assert2.New(t)

	t.Run("not-a-schema", func(t *testing.T) {
		res := ReplaceFromSchemaExample(NewTestReplaceContext("not-a-schema"))
		assert.Nil(res)
	})

	t.Run("with-a-schema", func(t *testing.T) {
		schema := &types.Schema{Example: "hallo, welt!"}
		res := ReplaceFromSchemaExample(NewTestReplaceContext(schema))
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
		schema := &types.Schema{Type: types.TypeBoolean}
		res := ApplySchemaConstraints(schema, true)
		assert.Equal(true, res)
	})

	t.Run("number-conv-fails", func(t *testing.T) {
		schema := &types.Schema{Type: types.TypeNumber}
		res := ApplySchemaConstraints(schema, "abc")
		assert.Nil(res)
	})

	t.Run("int-conv-fails", func(t *testing.T) {
		schema := &types.Schema{Type: types.TypeInteger}
		res := ApplySchemaConstraints(schema, "abc")
		assert.Nil(res)
	})

	t.Run("string-ok", func(t *testing.T) {
		schema := &types.Schema{Type: types.TypeString, MinLength: 5}
		res := ApplySchemaConstraints(schema, "hallo, welt!")
		assert.Equal("hallo, welt!", res)
	})

	t.Run("number-ok", func(t *testing.T) {
		schema := &types.Schema{Type: types.TypeNumber, Minimum: 100}
		res := ApplySchemaConstraints(schema, 133)
		assert.Equal(133.0, res)
	})

	t.Run("int-ok", func(t *testing.T) {
		schema := &types.Schema{Type: types.TypeInteger, Maximum: 10}
		res := ApplySchemaConstraints(schema, 6)
		assert.Equal(int64(6), res)
	})

	t.Run("bool-ok", func(t *testing.T) {
		schema := &types.Schema{Type: types.TypeBoolean}
		res := ApplySchemaConstraints(schema, true)
		assert.True(res.(bool))
	})

	t.Run("bool-ok-with-enum", func(t *testing.T) {
		schema := &types.Schema{Type: types.TypeBoolean, Enum: []any{true}}
		res := ApplySchemaConstraints(schema, false)
		assert.True(res.(bool))
	})
}

func TestApplySchemaStringConstraints(t *testing.T) {
	assert := assert2.New(t)

	t.Run("nil-schema", func(t *testing.T) {
		res := applySchemaStringConstraints(nil, "some-value")
		assert.Equal("some-value", res)
	})

	t.Run("no-constraints", func(t *testing.T) {
		schema := &types.Schema{Type: types.TypeString}
		res := applySchemaStringConstraints(schema, "hallo welt!")
		assert.Equal("hallo welt!", res)
	})

	t.Run("pattern-ok", func(t *testing.T) {
		schema := &types.Schema{
			Type:    types.TypeString,
			Pattern: "^[0-9]{2}[a-z]+$",
		}
		res := applySchemaStringConstraints(schema, "12go")
		assert.Equal("12go", res)
	})

	t.Run("pattern-fails", func(t *testing.T) {
		schema := &types.Schema{
			Type:    types.TypeString,
			Pattern: "^[0-9]{2}$",
		}
		res := applySchemaStringConstraints(schema, "12go")
		assert.NotNil(res)
	})

	t.Run("enum-ok", func(t *testing.T) {
		schema := &types.Schema{
			Type: types.TypeString,
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
		schema := &types.Schema{
			Type: types.TypeString,
			Enum: enum,
		}
		res := applySchemaStringConstraints(schema, "mice")
		assert.Contains(enum, res)
	})

	t.Run("min-length-ok", func(t *testing.T) {
		schema := &types.Schema{
			Type:      types.TypeString,
			MinLength: 5,
		}
		res := applySchemaStringConstraints(schema, "hallo")
		assert.Equal("hallo", res)
	})

	t.Run("min-length-applied", func(t *testing.T) {
		schema := &types.Schema{
			Type:      types.TypeString,
			MinLength: 5,
		}
		res := applySchemaStringConstraints(schema, "ha")
		assert.Equal("ha---", res)
	})

	t.Run("max-length-ok", func(t *testing.T) {
		schema := &types.Schema{
			Type:      types.TypeString,
			MaxLength: 5,
		}
		res := applySchemaStringConstraints(schema, "hallo")
		assert.Equal("hallo", res)
	})

	t.Run("max-length-applied", func(t *testing.T) {
		schema := &types.Schema{
			Type:      types.TypeString,
			MaxLength: 5,
		}
		res := applySchemaStringConstraints(schema, "hallo welt!")
		assert.Equal("hallo", res)
	})

	t.Run("invalid-pattern-with-example", func(t *testing.T) {
		schema := &types.Schema{
			Type:    types.TypeString,
			Pattern: "[0-9]+",
			Example: "21",
		}
		res := applySchemaStringConstraints(schema, "hallo welt!")
		assert.Equal("21", res)
	})

	t.Run("invalid-pattern-without-example-create-failed", func(t *testing.T) {
		schema := &types.Schema{
			Type:    types.TypeString,
			Pattern: "^/nice[0-9]+",
		}
		res := applySchemaStringConstraints(schema, "/nice")
		assert.NotNil(res)
	})

	t.Run("invalid-pattern-without-example-created", func(t *testing.T) {
		schema := &types.Schema{
			Type:    types.TypeString,
			Pattern: "^/nice/dice$",
		}
		res := applySchemaStringConstraints(schema, "hallo welt!")
		assert.Equal("/nice/dice", res)
	})
}

func TestApplySchemaNumberConstraints(t *testing.T) {
	assert := assert2.New(t)

	t.Run("nil-schema", func(t *testing.T) {
		res := applySchemaNumberConstraints(nil, 123)
		assert.Equal(123.0, res)
	})

	t.Run("no-constraints", func(t *testing.T) {
		schema := &types.Schema{Type: types.TypeNumber}
		res := applySchemaNumberConstraints(schema, 123)
		assert.Equal(123.0, res)
	})

	t.Run("min-ok", func(t *testing.T) {
		schema := &types.Schema{Type: types.TypeNumber, Minimum: 100}
		res := applySchemaNumberConstraints(schema, 100)
		assert.Equal(100.0, res)
	})

	t.Run("min-applied", func(t *testing.T) {
		schema := &types.Schema{Type: types.TypeNumber, Minimum: 100}
		res := applySchemaNumberConstraints(schema, 99)
		assert.Equal(100.0, res)
	})

	t.Run("max-ok", func(t *testing.T) {
		schema := &types.Schema{Type: types.TypeNumber, Maximum: 100}
		res := applySchemaNumberConstraints(schema, 100)
		assert.Equal(100.0, res)
	})

	t.Run("max-applied", func(t *testing.T) {
		schema := &types.Schema{Type: types.TypeNumber, Maximum: 100}
		res := applySchemaNumberConstraints(schema, 123)
		assert.Equal(100.0, res)
	})

	t.Run("mult-of-ok", func(t *testing.T) {
		schema := &types.Schema{Type: types.TypeNumber, MultipleOf: 5}
		res := applySchemaNumberConstraints(schema, 15)
		assert.Equal(15.0, res)
	})

	t.Run("mult-of-applied", func(t *testing.T) {
		schema := &types.Schema{Type: types.TypeNumber, MultipleOf: 3}
		res := applySchemaNumberConstraints(schema, 100)
		assert.Equal(99.0, res)
	})

	t.Run("min-max-mult-of-applied", func(t *testing.T) {
		schema := &types.Schema{
			Type:       types.TypeNumber,
			MultipleOf: 3,
			Minimum:    12,
			Maximum:    21,
		}
		res := applySchemaNumberConstraints(schema, 100)
		assert.Equal(21.0, res)
	})

	t.Run("enum-ints", func(t *testing.T) {
		schema := &types.Schema{
			Type: types.TypeNumber,
			Enum: []any{10, 20, 30},
		}
		res := applySchemaNumberConstraints(schema, 100)
		assert.Contains([]float64{10, 20, 30}, res)
	})

	t.Run("enum-floats", func(t *testing.T) {
		schema := &types.Schema{
			Type: types.TypeNumber,
			Enum: []any{10.1, 20.2, 30.3},
		}
		res := applySchemaNumberConstraints(schema, 100)
		assert.Contains([]float64{10.1, 20.2, 30.3}, res)
	})
}

func TestCreateStringFromPattern(t *testing.T) {
	assert := assert2.New(t)

	type testCase struct {
		pattern           string
		expectedLength    int
		expectedMinLength int
		expectedMaxLength int
	}

	testCases := []testCase{
		{pattern: `^/abc$`, expectedLength: 4},
		{pattern: `^/abc`, expectedMinLength: 4},
		{pattern: "^/v1/calculations/[^/]+/items"},
		{pattern: `^((-?[0-9]+)|(-?([0-9]+)?[.][0-9]+))$`},
		{pattern: `^([A-Z]{2}|C2)$`, expectedLength: 2},
		{pattern: `^[0-9]{1,10}(\.[0-9]{1,2})?$`, expectedMinLength: 1, expectedMaxLength: 13},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("case-%d", i+1), func(t *testing.T) {
			res := createStringFromPattern(tc.pattern)

			assert.True(types.ValidateStringWithPattern(res, tc.pattern))

			if tc.expectedLength > 0 {
				assert.Len(res, tc.expectedLength,
					fmt.Sprintf("case %d: expected %d, got %d", i+1, tc.expectedLength, len(res)))
			} else if tc.expectedMinLength > 0 {
				assert.GreaterOrEqual(len(res), tc.expectedMinLength,
					fmt.Sprintf("case %d: expected min %d, got %d", i+1, tc.expectedMinLength, len(res)))
			} else if tc.expectedMaxLength > 0 {
				assert.LessOrEqual(len(res), tc.expectedMaxLength,
					fmt.Sprintf("case %d: expected max %d, got %d", i+1, tc.expectedMaxLength, len(res)))
			}
		})
	}
}

func TestReplaceFromSchemaFallback(t *testing.T) {
	assert := assert2.New(t)

	t.Run("not-a-schema", func(t *testing.T) {
		res := ReplaceFromSchemaFallback(NewTestReplaceContext("not-a-schema"))
		assert.Nil(res)
	})

	t.Run("with-a-schema", func(t *testing.T) {
		schema := &types.Schema{Default: "hallo, welt!"}
		res := ReplaceFromSchemaFallback(NewTestReplaceContext(schema))
		assert.Equal("hallo, welt!", res)
	})
}

func TestIsReadWriteMatch(t *testing.T) {
	type testcase struct {
		schema   *types.Schema
		state    *ReplaceState
		expected bool
	}

	testcases := []testcase{
		{nil, nil, true},
		{&types.Schema{}, nil, true},
		{&types.Schema{ReadOnly: true}, nil, true},
		{&types.Schema{WriteOnly: true}, nil, true},
		{&types.Schema{ReadOnly: true}, &ReplaceState{IsContentReadOnly: true}, true},
		{&types.Schema{WriteOnly: true}, &ReplaceState{IsContentWriteOnly: true}, true},
		{&types.Schema{ReadOnly: true}, &ReplaceState{IsContentWriteOnly: true}, false},
		{&types.Schema{WriteOnly: true}, &ReplaceState{IsContentReadOnly: true}, false},
	}

	for i, tc := range testcases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			res := IsMatchSchemaReadWriteToState(tc.schema, tc.state)
			if tc.expected != res {
				t.Errorf("[%d] expected %v, got %v", i, tc.expected, res)
			}
		})
	}
}
