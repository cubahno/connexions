package generator

import (
	"testing"

	assert2 "github.com/stretchr/testify/assert"
)

func TestLoadServiceContext(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	t.Run("loads service context with defaults", func(t *testing.T) {
		serviceCtx := []byte(`
user_id: 12345
username: testuser
`)
		defaultContexts := []map[string]map[string]any{
			{
				"common": {
					"id":   "common-id",
					"name": "common-name",
				},
				"fake": {
					"email": "test@example.com",
				},
			},
		}

		result := LoadServiceContext(serviceCtx, defaultContexts)

		// Now we have 4 contexts: service, common, fake, words
		assert.GreaterOrEqual(len(result), 3)

		// Check service context (YAML parses numbers as int)
		assert.Equal(12345, result[0]["user_id"])
		assert.Equal("testuser", result[0]["username"])

		// Check common context
		assert.Equal("common-id", result[1]["id"])
		assert.Equal("common-name", result[1]["name"])

		// Check fake context
		assert.Equal("test@example.com", result[2]["email"])
	})

	t.Run("handles empty service context", func(t *testing.T) {
		serviceCtx := []byte(``)
		defaultContexts := []map[string]map[string]any{
			{
				"common": {
					"id": "common-id",
				},
				"fake": {
					"email": "test@example.com",
				},
			},
		}

		result := LoadServiceContext(serviceCtx, defaultContexts)

		assert.GreaterOrEqual(len(result), 3)
		assert.Equal(0, len(result[0])) // Empty service context
		assert.Equal("common-id", result[1]["id"])
		assert.Equal("test@example.com", result[2]["email"])
	})

	t.Run("handles nil default contexts", func(t *testing.T) {
		serviceCtx := []byte(`
user_id: 12345
`)
		result := LoadServiceContext(serviceCtx, nil)

		assert.GreaterOrEqual(len(result), 1)
		assert.Equal(12345, result[0]["user_id"])
	})
}
