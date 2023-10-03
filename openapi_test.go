//go:build !integration

package connexions

import (
	"github.com/cubahno/connexions/config"
	"github.com/cubahno/connexions/openapi"
	"github.com/cubahno/connexions/openapi/providers/kin"
	"github.com/cubahno/connexions/openapi/providers/lib"
	assert2 "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"testing"
)

func TestNewDocumentFromFileFactory(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()
	filePath := filepath.Join("testdata", "document-petstore.yml")

	t.Run("KinOpenAPIProvider", func(t *testing.T) {
		res, err := NewDocumentFromFileFactory(config.KinOpenAPIProvider)(filePath)
		assert.Nil(err)
		assert.Equal(config.KinOpenAPIProvider, res.Provider())
		assert.Greater(len(res.GetResources()), 0)
	})

	t.Run("LibOpenAPIProvider", func(t *testing.T) {
		res, err := NewDocumentFromFileFactory(config.LibOpenAPIProvider)(filePath)
		assert.Nil(err)
		assert.Equal(config.LibOpenAPIProvider, res.Provider())
		assert.Greater(len(res.GetResources()), 0)
	})

	t.Run("unknown-fallbacks-to-LibOpenAPIProvider", func(t *testing.T) {
		res, err := NewDocumentFromFileFactory("unknown")(filePath)
		assert.Nil(err)
		assert.Equal(config.LibOpenAPIProvider, res.Provider())
		assert.Greater(len(res.GetResources()), 0)
	})
}

type OtherTestDocument struct {
	openapi.Document
}

func (d *OtherTestDocument) Provider() config.SchemaProvider {
	return "other"
}

func TestNewOpenAPIValidator(t *testing.T) {
	assert := require.New(t)
	t.Run("KinOpenAPIProvider", func(t *testing.T) {
		doc, err := kin.NewDocumentFromFile(filepath.Join("testdata", "document-petstore.yml"))
		assert.Nil(err)
		res := NewOpenAPIValidator(doc)
		assert.NotNil(res)
		_, ok := res.(*kin.Validator)
		assert.True(ok)
	})

	t.Run("LibOpenAPIProvider", func(t *testing.T) {
		doc, err := lib.NewDocumentFromFile(filepath.Join("testdata", "document-petstore.yml"))
		assert.Nil(err)
		res := NewOpenAPIValidator(doc)
		assert.NotNil(res)
		_, ok := res.(*lib.Validator)
		assert.True(ok)
	})

	t.Run("unknown", func(t *testing.T) {
		doc, _ := kin.NewDocumentFromFile(filepath.Join("testdata", "document-petstore.yml"))
		kinDoc, _ := doc.(*kin.Document)

		other := &OtherTestDocument{Document: kinDoc}

		res := NewOpenAPIValidator(other)
		assert.Nil(res)
	})
}
