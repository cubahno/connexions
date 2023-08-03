package files

import (
	"os"
	"path/filepath"
	"testing"

	assert2 "github.com/stretchr/testify/assert"
)

func TestIsEmptyDir(t *testing.T) {
	assert := assert2.New(t)

	t.Run("empty-directory", func(t *testing.T) {
		tempDir := t.TempDir()
		assert.True(IsEmptyDir(tempDir))
	})

	t.Run("directory-with-file", func(t *testing.T) {
		tempDir := t.TempDir()
		file, err := os.Create(filepath.Join(tempDir, "test.txt"))
		if err != nil {
			t.FailNow()
		}
		_ = file.Close()
		assert.False(IsEmptyDir(tempDir))
	})

	t.Run("directory-with-subdirectory", func(t *testing.T) {
		tempDir := t.TempDir()
		_ = os.Mkdir(filepath.Join(tempDir, "subdir"), 0755)
		assert.False(IsEmptyDir(tempDir))
	})

	t.Run("non-existent-directory", func(t *testing.T) {
		assert.False(IsEmptyDir("/non-existent-path"))
	})
}

func TestIsJsonType(t *testing.T) {
	assert := assert2.New(t)

	t.Run("valid-json-object", func(t *testing.T) {
		assert.True(IsJsonType([]byte(`{"key": "value"}`)))
	})

	t.Run("valid-json-array", func(t *testing.T) {
		// Arrays don't unmarshal into map[string]interface{}, so this will be false
		assert.False(IsJsonType([]byte(`[1, 2, 3]`)))
	})

	t.Run("valid-json-nested", func(t *testing.T) {
		assert.True(IsJsonType([]byte(`{"user": {"name": "John", "age": 30}}`)))
	})

	t.Run("invalid-json-yaml", func(t *testing.T) {
		assert.False(IsJsonType([]byte(`foo: bar`)))
	})

	t.Run("invalid-json-plain-text", func(t *testing.T) {
		assert.False(IsJsonType([]byte(`this is not json`)))
	})

	t.Run("empty-content", func(t *testing.T) {
		assert.False(IsJsonType([]byte(``)))
	})
}

func TestIsYamlType(t *testing.T) {
	assert := assert2.New(t)

	t.Run("valid-yaml-simple", func(t *testing.T) {
		assert.True(IsYamlType([]byte(`foo: bar`)))
	})

	t.Run("valid-yaml-nested", func(t *testing.T) {
		assert.True(IsYamlType([]byte(`
user:
  name: John
  age: 30
`)))
	})

	t.Run("valid-yaml-array", func(t *testing.T) {
		// Arrays don't unmarshal into map[string]interface{}, so this will be false
		assert.False(IsYamlType([]byte(`
- item1
- item2
- item3
`)))
	})

	t.Run("valid-json-is-also-yaml", func(t *testing.T) {
		assert.True(IsYamlType([]byte(`{"key": "value"}`)))
	})

	t.Run("invalid-yaml-number-only", func(t *testing.T) {
		assert.False(IsYamlType([]byte(`100`)))
	})

	t.Run("empty-content", func(t *testing.T) {
		// Empty content unmarshals to nil map, which is considered valid YAML
		assert.True(IsYamlType([]byte(``)))
	})
}
