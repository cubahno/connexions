package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResponse_GetSuccess(t *testing.T) {
	t.Run("Returns success response for valid success code", func(t *testing.T) {
		resp := NewResponse(map[int]*ResponseItem{
			200: {StatusCode: 200, Content: &Schema{Type: "object"}},
			400: {StatusCode: 400, Content: &Schema{Type: "object"}},
		}, 200)

		success := resp.GetSuccess()
		assert.NotNil(t, success)
		assert.Equal(t, 200, success.StatusCode)
	})

	t.Run("Returns nil when success code not in map", func(t *testing.T) {
		resp := NewResponse(map[int]*ResponseItem{
			400: {StatusCode: 400, Content: &Schema{Type: "object"}},
		}, 200)

		success := resp.GetSuccess()
		assert.Nil(t, success)
	})

	t.Run("Returns correct response when success code is not 200", func(t *testing.T) {
		resp := NewResponse(map[int]*ResponseItem{
			201: {StatusCode: 201, Content: &Schema{Type: "object"}},
			400: {StatusCode: 400, Content: &Schema{Type: "object"}},
		}, 201)

		success := resp.GetSuccess()
		assert.NotNil(t, success)
		assert.Equal(t, 201, success.StatusCode)
	})
}

func TestResponse_GetResponse(t *testing.T) {
	t.Run("Returns response for existing status code", func(t *testing.T) {
		resp := NewResponse(map[int]*ResponseItem{
			200: {StatusCode: 200, Content: &Schema{Type: "object"}},
			404: {StatusCode: 404, Content: &Schema{Type: "object"}},
			500: {StatusCode: 500, Content: &Schema{Type: "object"}},
		}, 200)

		notFound := resp.GetResponse(404)
		assert.NotNil(t, notFound)
		assert.Equal(t, 404, notFound.StatusCode)

		serverError := resp.GetResponse(500)
		assert.NotNil(t, serverError)
		assert.Equal(t, 500, serverError.StatusCode)
	})

	t.Run("Returns nil for non-existent status code", func(t *testing.T) {
		resp := NewResponse(map[int]*ResponseItem{
			200: {StatusCode: 200, Content: &Schema{Type: "object"}},
		}, 200)

		result := resp.GetResponse(404)
		assert.Nil(t, result)
	})

	t.Run("Can retrieve success code via GetResponse", func(t *testing.T) {
		resp := NewResponse(map[int]*ResponseItem{
			200: {StatusCode: 200, Content: &Schema{Type: "object"}},
		}, 200)

		result := resp.GetResponse(200)
		assert.NotNil(t, result)
		assert.Equal(t, 200, result.StatusCode)
	})
}

func TestNewResponse(t *testing.T) {
	t.Run("Creates response with empty map", func(t *testing.T) {
		resp := NewResponse(map[int]*ResponseItem{}, 200)

		assert.NotNil(t, resp)
		assert.Nil(t, resp.GetSuccess())
		assert.Nil(t, resp.GetResponse(200))
	})

	t.Run("Creates response with nil map", func(t *testing.T) {
		resp := NewResponse(nil, 200)

		assert.NotNil(t, resp)
		assert.Nil(t, resp.GetSuccess())
		assert.Nil(t, resp.GetResponse(200))
	})

	t.Run("Preserves all response items", func(t *testing.T) {
		all := map[int]*ResponseItem{
			200: {StatusCode: 200},
			201: {StatusCode: 201},
			400: {StatusCode: 400},
			404: {StatusCode: 404},
			500: {StatusCode: 500},
		}
		resp := NewResponse(all, 200)

		// Verify all items are accessible
		for code := range all {
			item := resp.GetResponse(code)
			assert.NotNil(t, item, "Expected response for code %d", code)
			assert.Equal(t, code, item.StatusCode)
		}
	})
}
