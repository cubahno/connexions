//go:build !integration

package connexions

import (
	assert2 "github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

func TestLibV3Operation(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	withFriendsPath := filepath.Join("test_fixtures", "document-person-with-friends.yml")
	docWithFriends, err := NewLibOpenAPIDocumentFromFile(withFriendsPath)
	assert.Nil(err)

	t.Run("getContent-empty", func(t *testing.T) {
		op := docWithFriends.FindOperation(&FindOperationOptions{"", "/person/{id}/find", "GET", nil})
		assert.NotNil(op)
		opLib := op.(*LibV3Operation)

		res, contentType := opLib.getContent(nil)

		assert.Nil(res)
		assert.Equal("", contentType)
	})
}
