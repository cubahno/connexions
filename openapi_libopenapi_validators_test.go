package connexions

import (
	"github.com/stretchr/testify/require"
	"path/filepath"
	"testing"
)

func TestNewLibOpenAPIValidator(t *testing.T) {
	assert := require.New(t)
	doc, err := NewLibOpenAPIDocumentFromFile(filepath.Join("testdata", "document-petstore.yml"))
	assert.Nil(err)

	inst := NewLibOpenAPIValidator(doc)
	assert.NotNil(inst)

	inst = NewLibOpenAPIValidator(nil)
	assert.Nil(inst)
}
