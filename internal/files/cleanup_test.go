package files

import (
	"os"
	"path/filepath"
	"testing"

	assert2 "github.com/stretchr/testify/assert"
)

func TestCleanupServiceFileStructure(t *testing.T) {
	assert := assert2.New(t)

	t.Run("removes-empty-directories", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create some subdirectories
		subDir1 := filepath.Join(tempDir, "subdir1")
		subDir2 := filepath.Join(tempDir, "subdir2")
		emptySubDir := filepath.Join(tempDir, "emptysubdir")

		err := os.Mkdir(subDir1, 0755)
		assert.NoError(err)
		err = os.Mkdir(subDir2, 0755)
		assert.NoError(err)
		err = os.Mkdir(emptySubDir, 0755)
		assert.NoError(err)

		err = CleanupServiceFileStructure(tempDir)
		assert.NoError(err)

		// Verify that all empty directories have been removed
		_, err = os.Stat(emptySubDir)
		assert.True(os.IsNotExist(err))
		_, err = os.Stat(subDir1)
		assert.True(os.IsNotExist(err))
		_, err = os.Stat(subDir2)
		assert.True(os.IsNotExist(err))
	})

	t.Run("keeps-directories-with-files", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create directory with file
		subDir := filepath.Join(tempDir, "subdir")
		err := os.Mkdir(subDir, 0755)
		assert.NoError(err)

		filePath := filepath.Join(subDir, "file.txt")
		err = os.WriteFile(filePath, []byte("content"), 0644)
		assert.NoError(err)

		err = CleanupServiceFileStructure(tempDir)
		assert.NoError(err)

		// Verify directory still exists
		_, err = os.Stat(subDir)
		assert.NoError(err)

		// Verify file still exists
		_, err = os.Stat(filePath)
		assert.NoError(err)
	})

	t.Run("handles-nested-empty-directories", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create nested empty directories
		nested := filepath.Join(tempDir, "a", "b", "c")
		err := os.MkdirAll(nested, 0755)
		assert.NoError(err)

		err = CleanupServiceFileStructure(tempDir)
		assert.NoError(err)

		// All should be removed
		_, err = os.Stat(nested)
		assert.True(os.IsNotExist(err))
	})

	t.Run("file-instead-of-directory", func(t *testing.T) {
		tempDir := t.TempDir()
		f := filepath.Join(tempDir, "file.txt")
		err := SaveFile(f, []byte("test"))
		assert.NoError(err)

		err = CleanupServiceFileStructure(f)
		assert.NoError(err)
	})

	t.Run("non-existent-path", func(t *testing.T) {
		// WalkDir handles non-existent paths gracefully by calling the walk function with an error
		// Our implementation skips IsNotExist errors, so this should not return an error
		err := CleanupServiceFileStructure("/non-existent-path-that-does-not-exist")
		assert.NoError(err)
	})

	t.Run("permission-denied-error", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create a subdirectory
		subDir := filepath.Join(tempDir, "restricted")
		err := os.Mkdir(subDir, 0755)
		assert.NoError(err)

		// Create a nested directory
		nestedDir := filepath.Join(subDir, "nested")
		err = os.Mkdir(nestedDir, 0755)
		assert.NoError(err)

		// Remove read permission from the subdirectory to cause a permission error
		err = os.Chmod(subDir, 0000)
		assert.NoError(err)

		// Cleanup should return an error (permission denied)
		err = CleanupServiceFileStructure(tempDir)
		assert.Error(err)

		// Restore permissions for cleanup
		_ = os.Chmod(subDir, 0755)
	})
}
