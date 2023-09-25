package connexions

import (
	"github.com/stretchr/testify/require"
	"path/filepath"
	"testing"
)

func getSchemaFromKin(t *testing.T, fileName, componentID string, parseConfig *ParseConfig) *Schema {
	t.Helper()
	assert := require.New(t)

	kinDoc, err := NewKinDocumentFromFile(filepath.Join("test_fixtures", fileName))
	assert.Nil(err)
	doc := kinDoc.(*KinDocument)
	kinSchema := doc.Components.Schemas[componentID].Value
	assert.NotNil(kinSchema)

	return NewSchemaFromKin(kinSchema, parseConfig)
}

func getSchemaFromLib(t *testing.T, fileName, componentID string, parseConfig *ParseConfig) *Schema {
	t.Helper()
	assert := require.New(t)

	libDoc, err := NewLibOpenAPIDocumentFromFile(filepath.Join("test_fixtures", fileName))
	assert.Nil(err)
	doc := libDoc.(*LibV3Document)
	libSchema := doc.Model.Components.Schemas[componentID].Schema()
	assert.NotNil(libSchema)

	return NewSchemaFromLibOpenAPI(libSchema, parseConfig)
}

// TestNewSchema tests the NewSchema* functions from providers, so we don't have to repeat the same tests for each
func TestNewSchema(t *testing.T) {
	assert := require.New(t)

	tc := []struct {
		name      SchemaProvider
		getSchema func(t *testing.T, fileName, componentID string, parseConfig *ParseConfig) *Schema
	}{
		{KinOpenAPIProvider, getSchemaFromKin},
		{LibOpenAPIProvider, getSchemaFromLib},
	}

	for _, schemaFromProvider := range tc {
		t.Run("WithParseConfig-max-recursive-levels-"+string(schemaFromProvider.name), func(t *testing.T) {
			res := schemaFromProvider.getSchema(t, "document-circular-ucr.yml", "OrgByIdResponseWrapperModel",
				&ParseConfig{MaxRecursionLevels: 1})

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
			assert.Equal(TypeObject, res.Type)

			success := res.Properties["success"]
			assert.Equal(&Schema{Type: TypeBoolean}, success)

			response := res.Properties["response"]
			assert.Equal(TypeObject, response.Type)

			typ := response.Properties["type"]
			assert.Equal(&Schema{
				Type: TypeString,
				Enum: types,
			}, typ)

			parent := response.Properties["parent"]
			assert.Equal(&Schema{
				Type: TypeObject,
				Properties: map[string]*Schema{
					"parent": nil,
					"type": {
						Type: TypeString,
						Enum: types,
					},
					"children": {
						Type:    TypeArray,
						Items:   &Schema{Type: TypeString},
						Example: example,
					},
				},
			}, parent)

			children := response.Properties["children"]
			assert.Equal(TypeArray, children.Type)
			childrenItems := children.Items
			assert.Equal(TypeObject, childrenItems.Type)

			childrenParent := childrenItems.Properties["parent"]
			assert.Nil(childrenParent)

			childrenChildren := childrenItems.Properties["children"]
			assert.Equal(&Schema{
				Type:    TypeArray,
				Items:   &Schema{Type: TypeString},
				Example: example,
			}, childrenChildren)

			childrenType := childrenItems.Properties["type"]
			assert.Equal(&Schema{
				Type: TypeString,
				Enum: types,
			}, childrenType)

			childrenExample := children.Example
			assert.Equal(example, childrenExample)
		})
	}
}
