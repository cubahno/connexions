package xs

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

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
			"userData": map[string]interface{}{
				"name": "John Doe",
				"country": map[string]interface{}{
					"name": "Germany",
				},
			},
		}
		namePath := []string{"user", "country", "name"}
		res := ReplaceValueWithContext(namePath, context)

		assert.Equal(t, "Germany", res)
	})

	t.Run("single-namepath-has-name-prefix", func(t *testing.T) {
		context := map[string]interface{}{
			"nameFull":  "Jane Doe",
			"nameOther": "John Doe",
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


func TestReplaceMapFunctionPlaceholders(t *testing.T) {
	data := map[string]interface{}{
		"user": map[string]interface{}{
			"id": "func:makeUUID",
			"country": map[string]interface{}{
				"id":   "func:makeUUID",
				"code": "func:makeCountryCode",
			},
			"name":  "Jane Doe",
			"email": "",
		},
	}

	funcs := map[string]interface{}{
		"makeUUID":        func() any { return "GeneratedUUID" },
		"makeCountryCode": func() any { return "DE" },
	}

	result := ReplaceMapFunctionPlaceholders(data, funcs)

	if user, ok := result.(map[string]any)["user"].(map[string]any); ok {
		if idFn, ok := user["id"].(func() any); ok {
			idValue := idFn()
			if idValue != "GeneratedUUID" {
				t.Errorf("Expected GeneratedUUID, got %s", idValue)
			}
		} else {
			t.Error("Failed to get function for user id")
		}

		if countryCodeFn, ok := user["country"].(map[string]any)["code"].(func() any); ok {
			codeValue := countryCodeFn()
			if codeValue != "DE" {
				t.Errorf("Expected DE, got %s", codeValue)
			}
		} else {
			t.Error("Failed to get function for country code")
		}
	} else {
		t.Error("Failed to get user map")
	}
}
