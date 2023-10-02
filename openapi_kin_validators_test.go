package connexions

import (
	"github.com/stretchr/testify/require"
	"path/filepath"
	"testing"
)

func TestNewKinOpenAPIValidator(t *testing.T) {
	assert := require.New(t)
	doc, err := NewKinDocumentFromFile(filepath.Join("test_fixtures", "document-petstore.yml"))
	assert.Nil(err)
	inst := NewKinOpenAPIValidator(doc)
	assert.NotNil(inst)
}
