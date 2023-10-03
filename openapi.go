package connexions

import (
	"github.com/cubahno/connexions/config"
	"github.com/cubahno/connexions/openapi"
	"github.com/cubahno/connexions/openapi/providers/kin"
	"github.com/cubahno/connexions/openapi/providers/lib"
)

// NewOpenAPIValidator returns a new Validator based on the Document provider.
func NewOpenAPIValidator(doc openapi.Document) openapi.Validator {
	switch doc.Provider() {
	case config.KinOpenAPIProvider:
		return kin.NewValidator(doc)
	case config.LibOpenAPIProvider:
		return lib.NewValidator(doc)
	}
	return nil
}

// NewDocumentFromFileFactory returns a function that creates a new Document from a file.
func NewDocumentFromFileFactory(provider config.SchemaProvider) func(filePath string) (openapi.Document, error) {
	switch provider {
	case config.KinOpenAPIProvider:
		return kin.NewDocumentFromFile
	case config.LibOpenAPIProvider:
		return lib.NewDocumentFromFile
	default:
		return lib.NewDocumentFromFile
	}
}
