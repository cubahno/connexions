package connexions

import (
	"github.com/cubahno/connexions/config"
	"github.com/cubahno/connexions/openapi"
	"github.com/cubahno/connexions/openapi/providers/kin"
	"github.com/cubahno/connexions/openapi/providers/lib"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"testing"
)

func getSchemaFromKin(t *testing.T, fileName, componentID string, parseConfig *config.ParseConfig) *openapi.Schema {
	t.Helper()
	assert := require.New(t)

	kinDoc, err := kin.NewDocumentFromFile(filepath.Join("testdata", fileName))
	assert.Nil(err)
	doc := kinDoc.(*kin.Document)
	kinSchema := doc.Components.Schemas[componentID].Value
	assert.NotNil(kinSchema)

	return kin.NewSchemaFromKin(kinSchema, parseConfig)
}

func getSchemaFromLib(t *testing.T, fileName, componentID string, parseConfig *config.ParseConfig) *openapi.Schema {
	t.Helper()
	assert := require.New(t)

	libDoc, err := lib.NewDocumentFromFile(filepath.Join("testdata", fileName))
	assert.Nil(err)
	doc := libDoc.(*lib.V3Document)
	libSchema := doc.Model.Components.Schemas[componentID].Schema()
	assert.NotNil(libSchema)

	return lib.NewSchema(libSchema, parseConfig)
}

// TestNewSchema tests the NewSchema* functions from providers, so we don't have to repeat the same tests for each
func TestNewSchema(t *testing.T) {
	assert := require.New(t)

	tc := []struct {
		name      config.SchemaProvider
		getSchema func(t *testing.T, fileName, componentID string, parseConfig *config.ParseConfig) *openapi.Schema
	}{
		{config.KinOpenAPIProvider, getSchemaFromKin},
		{config.LibOpenAPIProvider, getSchemaFromLib},
	}

	for _, schemaFromProvider := range tc {
		t.Run("WithParseConfig-max-recursive-levels-"+string(schemaFromProvider.name), func(t *testing.T) {
			res := schemaFromProvider.getSchema(t, "document-circular-ucr.yml", "OrgByIdResponseWrapperModel",
				&config.ParseConfig{MaxRecursionLevels: 1})

			types := []any{
				"Department",
				"Division",
				"Organization",
			}

			example := []any{
				map[string]any{
					"type":        "string",
					"code":        "string",
					"description": "string",
					"isActive":    true,
				},
				map[string]any{
					"type":        "string",
					"code":        "string",
					"description": "string",
					"isActive":    true,
				},
			}

			assert.NotNil(res)
			assert.Equal(openapi.TypeObject, res.Type)

			success := res.Properties["success"]
			assert.Equal(&openapi.Schema{Type: openapi.TypeBoolean}, success)

			response := res.Properties["response"]
			assert.Equal(openapi.TypeObject, response.Type)

			typ := response.Properties["type"]
			assert.Equal(&openapi.Schema{
				Type: openapi.TypeString,
				Enum: types,
			}, typ)

			parent := response.Properties["parent"]
			assert.Equal(&openapi.Schema{
				Type: openapi.TypeObject,
				Properties: map[string]*openapi.Schema{
					"parent": nil,
					"type": {
						Type: openapi.TypeString,
						Enum: types,
					},
					"children": {
						Type:    openapi.TypeArray,
						Items:   &openapi.Schema{Type: openapi.TypeString},
						Example: example,
					},
				},
			}, parent)

			children := response.Properties["children"]
			assert.Equal(openapi.TypeArray, children.Type)
			childrenItems := children.Items
			assert.Equal(openapi.TypeObject, childrenItems.Type)

			childrenParent := childrenItems.Properties["parent"]
			assert.Nil(childrenParent)

			childrenChildren := childrenItems.Properties["children"]
			assert.Equal(&openapi.Schema{
				Type:    openapi.TypeArray,
				Items:   &openapi.Schema{Type: openapi.TypeString},
				Example: example,
			}, childrenChildren)

			childrenType := childrenItems.Properties["type"]
			assert.Equal(&openapi.Schema{
				Type: openapi.TypeString,
				Enum: types,
			}, childrenType)

			childrenExample := children.Example
			assert.Equal(example, childrenExample)
		})
	}

	for _, schemaFromProvider := range tc {
		t.Run("circular-with-additional-properties", func(t *testing.T) {
			res := schemaFromProvider.getSchema(t, "document-connexions.yml", "Map",
				&config.ParseConfig{MaxRecursionLevels: 0})

			expected := &openapi.Schema{
				Type: openapi.TypeObject,
				Properties: map[string]*openapi.Schema{
					"extra-1": {
						Type: openapi.TypeObject,
					},
					"extra-2": {
						Type: openapi.TypeObject,
					},
					"extra-3": {
						Type: openapi.TypeObject,
					},
				},
			}
			assert.NotNil(res)
			assert.Equal(expected, res)
		})
	}
}
