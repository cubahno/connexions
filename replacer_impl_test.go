package connexions

import (
	assert2 "github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestHasCorrectSchemaType(t *testing.T) {

}

func TestReplaceInHeaders(t *testing.T) {

}

func TestReplaceInPath(t *testing.T) {

}

func TestReplaceFromContext(t *testing.T) {

}

func TestReplaceValueWithContext(t *testing.T) {
	assert := assert2.New(t)

	t.Run("happy-path", func(t *testing.T) {
		context := map[string]interface{}{
			"user": map[string]interface{}{
				"age":  30,
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
		assert.Greater(len(value), 6)
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
}

func TestReplaceFromSchemaPrimitive(t *testing.T) {

}

func TestReplaceFromSchemaExample(t *testing.T) {

}

func TestReplaceFallback(t *testing.T) {

}

func TestApplySchemaConstraints(t *testing.T) {

}