package connexions

import (
	"encoding/json"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

func CreateKinSchemaFromFile(t *testing.T, filePath string) *openapi3.Schema {
	cont, err := os.ReadFile(filePath)
	if err != nil {
		t.Errorf("Error reading file: %v", err)
		t.FailNow()
	}

	// remove schema key if pre
	tmp := make(map[string]any)
	_ = json.Unmarshal(cont, &tmp)
	if _, ok := tmp["schema"]; ok {
		cont, _ = json.Marshal(tmp["schema"])
	}

	return CreateKinSchemaFromString(t, string(cont))
}

func TestMergeSubSchemas(t *testing.T) {
	t.Run("MergeKinSubSchemas", func(t *testing.T) {
		schema := CreateKinSchemaFromFile(t, filepath.Join(TestSchemaPath, "schema-with-sub-schemas.json"))
		res := MergeKinSubSchemas(schema)
		expectedProperties := []string{"user", "limit", "tag1", "tag2", "offset", "first"}

		resProps := make([]string, 0)
		for name, _ := range res.Properties {
			resProps = append(resProps, name)
		}

		assert.ElementsMatch(t, expectedProperties, resProps)
	})

	t.Run("without-all-of-and-empty-one-of-schema", func(t *testing.T) {
		schema := CreateKinSchemaFromFile(t, filepath.Join(TestSchemaPath, "schema-without-all-of.json"))
		res := MergeKinSubSchemas(schema)
		expectedProperties := []string{"first", "second"}

		resProps := make([]string, 0)
		for name, _ := range res.Properties {
			resProps = append(resProps, name)
		}

		assert.ElementsMatch(t, expectedProperties, resProps)
	})

	t.Run("with-allof-nil-schema", func(t *testing.T) {
		schema := CreateKinSchemaFromString(t, `{"AllOf": null}`)
		res := MergeKinSubSchemas(schema)
		assert.Equal(t, "object", res.Type)
	})

	t.Run("with-anyof-nil-schema", func(t *testing.T) {
		schema := CreateKinSchemaFromString(t, `{"AnyOf": null}`)
		res := MergeKinSubSchemas(schema)
		assert.Equal(t, "object", res.Type)
	})

	t.Run("empty-type-defaults-in-object", func(t *testing.T) {
		schema := CreateKinSchemaFromString(t, `{"type": ""}`)
		res := MergeKinSubSchemas(schema)
		assert.Equal(t, "object", res.Type)
	})
}
