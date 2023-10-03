//go:build !integration

package lib

import (
	"github.com/cubahno/connexions/openapi"
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
	assert2 "github.com/stretchr/testify/assert"
	"net/http"
	"path/filepath"
	"testing"
)

func TestLibV3Document(t *testing.T) {
	assert := assert2.New(t)

	t.Run("empty", func(t *testing.T) {
		doc := &V3Document{}
		res := doc.GetResources()
		assert.Equal(0, len(res))
	})
}

func TestLibV3Operation(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()
	testData := filepath.Join("..", "..", "..", "testdata")

	withFriendsPath := filepath.Join(testData, "document-person-with-friends.yml")
	docWithFriends, err := NewDocumentFromFile(withFriendsPath)
	assert.Nil(err)

	t.Run("ID", func(t *testing.T) {
		operation := &V3Operation{Operation: &v3high.Operation{OperationId: "findNice"}}
		res := operation.ID()
		assert.Equal("findNice", res)
	})

	t.Run("GetResponse-without-responses", func(t *testing.T) {
		op := v3high.Operation{}
		res := (&V3Operation{Operation: &op}).GetResponse()
		expected := &openapi.Response{
			StatusCode: http.StatusOK,
		}
		assert.Equal(expected, res)
	})

	t.Run("getContent-empty", func(t *testing.T) {
		op := docWithFriends.FindOperation(&openapi.OperationDescription{Resource: "/person/{id}/find", Method: "GET"})
		assert.NotNil(op)
		opLib := op.(*V3Operation)

		res, contentType := opLib.getContent(nil)

		assert.Nil(res)
		assert.Equal("", contentType)
	})
}
