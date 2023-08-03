package files

import (
	"os"
	"path/filepath"
	"testing"

	assert2 "github.com/stretchr/testify/assert"
)

func TestSaveFile(t *testing.T) {
	assert := assert2.New(t)

	t.Run("happy-path", func(t *testing.T) {
		contents := []byte("test file contents")
		filePath := filepath.Join(t.TempDir(), "a", "b", "c", "test.txt")
		err := SaveFile(filePath, contents)
		assert.NoError(err)

		// Verify file exists and content matches
		savedContent, err := os.ReadFile(filePath)
		assert.NoError(err)
		assert.Equal(contents, savedContent)
	})

	t.Run("invalid-dir", func(t *testing.T) {
		filePath := filepath.Join("/root", "a", "test.txt")
		err := SaveFile(filePath, []byte(""))
		assert.Error(err)
	})

	t.Run("invalid-path", func(t *testing.T) {
		filePath := filepath.Join("/root", "test.txt")
		err := SaveFile(filePath, []byte(""))
		assert.Error(err)
	})

	t.Run("empty-content", func(t *testing.T) {
		filePath := filepath.Join(t.TempDir(), "empty.txt")
		err := SaveFile(filePath, []byte(""))
		assert.NoError(err)

		content, err := os.ReadFile(filePath)
		assert.NoError(err)
		assert.Equal([]byte(""), content)
	})

	t.Run("overwrites-existing-file", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "overwrite.txt")

		// Write initial content
		err := SaveFile(filePath, []byte("initial"))
		assert.NoError(err)

		// Overwrite with new content
		err = SaveFile(filePath, []byte("updated"))
		assert.NoError(err)

		content, err := os.ReadFile(filePath)
		assert.NoError(err)
		assert.Equal("updated", string(content))
	})
}
