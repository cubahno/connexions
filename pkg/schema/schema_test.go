package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResponse(t *testing.T) {
	t.Run("GetSuccess returns success response", func(t *testing.T) {
		resp := &Response{
			All: map[int]*ResponseItem{
				200: {StatusCode: 200, Content: &Schema{Type: "object"}},
				404: {StatusCode: 404, Content: &Schema{Type: "object"}},
			},
			SuccessCode: 200,
		}

		success := resp.GetSuccess()
		assert.NotNil(t, success)
		assert.Equal(t, 200, success.StatusCode)
	})

	t.Run("GetResponse returns specific response", func(t *testing.T) {
		resp := &Response{
			All: map[int]*ResponseItem{
				200: {StatusCode: 200, Content: &Schema{Type: "object"}},
				404: {StatusCode: 404, Content: &Schema{Type: "object"}},
			},
			SuccessCode: 200,
		}

		notFound := resp.GetResponse(404)
		assert.NotNil(t, notFound)
		assert.Equal(t, 404, notFound.StatusCode)
	})

	t.Run("GetResponse returns nil for non-existent code", func(t *testing.T) {
		resp := &Response{
			All: map[int]*ResponseItem{
				200: {StatusCode: 200, Content: &Schema{Type: "object"}},
			},
			SuccessCode: 200,
		}

		result := resp.GetResponse(500)
		assert.Nil(t, result)
	})
}
