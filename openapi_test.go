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

// func TestOpenAPIResponse(t *testing.T) {
// 	t.Run("GetResponse-get-first-prioritized", func(t *testing.T) {
// 		content := OpenAPIContent{
// 			"text/html": {
// 				Schema: &SchemaRef{Value: &Schema{}},
// 			},
// 			"application/json": {
// 				Schema: &SchemaRef{Value: &Schema{}},
// 			},
// 			"text/plain": {
// 				Schema: &SchemaRef{Value: &Schema{}},
// 			},
// 		}
// 		contentType, schema := GetContentType(content)
//
// 		assert.Equal(t, "application/json", contentType)
// 		assert.NotNil(t, schema)
// 	})
//
// 	t.Run("GetResponse-get-first-found", func(t *testing.T) {
// 		content := OpenAPIContent{
// 			"multipart/form-data; boundary=something": {
// 				Schema: &SchemaRef{},
// 			},
// 			"application/xml": {
// 				Schema: &SchemaRef{},
// 			},
// 		}
// 		contentType, _ := GetContentType(content)
//
// 		assert.Contains(t, []string{"multipart/form-data; boundary=something", "application/xml"}, contentType)
// 	})
//
// 	t.Run("GetResponse-nothing-found", func(t *testing.T) {
// 		content := OpenAPIContent{}
// 		contentType, schema := GetContentType(content)
//
// 		assert.Equal(t, "", contentType)
// 		assert.Nil(t, schema)
// 	})
// }
