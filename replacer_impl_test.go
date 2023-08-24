package connexions

import (
	"github.com/stretchr/testify/assert"
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
	t.Run("happy-path", func(t *testing.T) {
		context := map[string]interface{}{
			"user": map[string]interface{}{
				"name": "Jane Doe",
				"age":  30,
				"country": map[string]interface{}{
					"name": "Germany",
					"code": "DE",
				},
			},
		}
		namePath := []string{"user", "country", "name"}
		res := ReplaceValueWithContext(namePath, context)

		assert.Equal(t, "Germany", res)
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

		assert.Equal(t, 30, res)
	})

	t.Run("has-name-prefix", func(t *testing.T) {
		context := map[string]interface{}{
			"user": map[string]interface{}{
				"name": "John Doe",
				"country": map[string]interface{}{
					"^name": "Germany",
				},
			},
		}
		namePath := []string{"user", "country", "name"}
		res := ReplaceValueWithContext(namePath, context)

		assert.Equal(t, "Germany", res)
	})

	t.Run("single-namepath-has-name-prefix", func(t *testing.T) {
		context := map[string]interface{}{
			"^name": "Jane Doe",
		}
		namePath := []string{"name"}
		res := ReplaceValueWithContext(namePath, context)

		assert.Equal(t, "Jane Doe", res)
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

		assert.Contains(t, names, res)
	})
}

func TestReplaceValueWithMapContext(t *testing.T) {

}

func TestReplaceFromSchemaFormat(t *testing.T) {

}

func TestReplaceFromSchemaPrimitive(t *testing.T) {

}

func TestReplaceFromSchemaExample(t *testing.T) {

}

func TestReplaceFallback(t *testing.T) {

}
