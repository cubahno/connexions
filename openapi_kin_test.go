package connexions

import (
	"github.com/getkin/kin-openapi/openapi3"
	assert2 "github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

func TestNewSchemaFromKin(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	t.Run("nested-all-of", func(t *testing.T) {
		target := openapi3.NewSchema()
		CreateSchemaFromYAMLFile(t, filepath.Join("test_fixtures", "schema-with-nested-all-of.yml"), target)

		res := NewSchemaFromKin(target, nil)
		assert.NotNil(res)

		expected := &Schema{
			Type: TypeObject,
			Properties: map[string]*Schema{
				"name":   {Type: TypeString},
				"age":    {Type: TypeInteger},
				"league": {Type: TypeString},
				"rating": {Type: TypeInteger},
				"tag":    {Type: TypeString},
			},
		}
		a, b := GetJSONPair(expected, res)
		if a != b {
			t.Errorf("expected / actual: \n%s\n%s", a, b)
		}
	})
}
