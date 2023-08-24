package connexions

import (
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

func TestOperation(t *testing.T) {
	t.Run("GetResponse", func(t *testing.T) {
		operation := CreateOperationFromFile(t, filepath.Join(TestSchemaPath, "operation-responses-500-200.json"))
		response, code := operation.GetResponse()

		assert.Equal(t, 200, code)
		assert.Equal(t, "OK", *response.Description)
		assert.NotNil(t, response.Content["application/json"])
	})

	t.Run("get-first-defined", func(t *testing.T) {
		operation := CreateOperationFromString(t, `
			{
				"responses": {
                    "500": {
                        "description": "Internal Server Error"
                    },
                    "400": {
                        "description": "Bad request"
                    }
				}
			}
		`)
		response, code := operation.GetResponse()

		assert.Contains(t, []int{500, 400}, code)
		assert.Contains(t, []string{"Internal Server Error", "Bad request"}, *response.Description)
	})

	t.Run("get-default-if-nothing-else", func(t *testing.T) {
		operation := CreateOperationFromString(t, `
			{
				"responses": {
                    "default": {
                        "description": "unexpected error"
                    }
				}
			}
		`)
		response, code := operation.GetResponse()

		assert.Equal(t, 200, code)
		assert.Equal(t, "unexpected error", *response.Description)
	})
}
