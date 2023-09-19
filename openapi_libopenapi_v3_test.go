//go:build !integration

package connexions

import (
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
	assert2 "github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

func TestLibV3Document(t *testing.T) {
	assert := assert2.New(t)

	t.Run("empty", func(t *testing.T) {
		doc := &LibV3Document{}
		res := doc.GetResources()
		assert.Equal(0, len(res))
	})
}

func TestLibV3Operation(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	withFriendsPath := filepath.Join("test_fixtures", "document-person-with-friends.yml")
	docWithFriends, err := NewLibOpenAPIDocumentFromFile(withFriendsPath)
	assert.Nil(err)

	t.Run("ID", func(t *testing.T) {
		operation := &LibV3Operation{Operation: &v3high.Operation{OperationId: "findNice"}}
		res := operation.ID()
		assert.Equal("findNice", res)
	})

	t.Run("getContent-empty", func(t *testing.T) {
		op := docWithFriends.FindOperation(&OperationDescription{"", "/person/{id}/find", "GET"})
		assert.NotNil(op)
		opLib := op.(*LibV3Operation)

		res, contentType := opLib.getContent(nil)

		assert.Nil(res)
		assert.Equal("", contentType)
	})
}
