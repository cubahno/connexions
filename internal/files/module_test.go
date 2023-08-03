package files

import (
	"os"
	"path/filepath"
	"testing"

	assert2 "github.com/stretchr/testify/assert"
)

func TestGetModuleInfo(t *testing.T) {
	assert := assert2.New(t)

	t.Run("finds-go-mod-in-current-directory", func(t *testing.T) {
		tempDir := t.TempDir()
		goModPath := filepath.Join(tempDir, "go.mod")
		goModContent := []byte("module github.com/example/myproject\n\ngo 1.21\n")
		err := os.WriteFile(goModPath, goModContent, 0644)
		assert.NoError(err)

		moduleName, moduleRoot, err := GetModuleInfo(tempDir)
		assert.NoError(err)
		assert.Equal("github.com/example/myproject", moduleName)
		assert.Equal(tempDir, moduleRoot)
	})

	t.Run("finds-go-mod-in-parent-directory", func(t *testing.T) {
		tempDir := t.TempDir()
		goModPath := filepath.Join(tempDir, "go.mod")
		goModContent := []byte("module github.com/example/parent\n")
		err := os.WriteFile(goModPath, goModContent, 0644)
		assert.NoError(err)

		// Create nested directory
		nestedDir := filepath.Join(tempDir, "a", "b", "c")
		err = os.MkdirAll(nestedDir, 0755)
		assert.NoError(err)

		moduleName, moduleRoot, err := GetModuleInfo(nestedDir)
		assert.NoError(err)
		assert.Equal("github.com/example/parent", moduleName)
		assert.Equal(tempDir, moduleRoot)
	})

	t.Run("error-when-no-go-mod-found", func(t *testing.T) {
		tempDir := t.TempDir()

		_, _, err := GetModuleInfo(tempDir)
		assert.Error(err)
		assert.Contains(err.Error(), "go.mod not found")
	})

	t.Run("error-when-go-mod-has-no-module-declaration", func(t *testing.T) {
		tempDir := t.TempDir()
		goModPath := filepath.Join(tempDir, "go.mod")
		goModContent := []byte("go 1.21\n")
		err := os.WriteFile(goModPath, goModContent, 0644)
		assert.NoError(err)

		_, _, err = GetModuleInfo(tempDir)
		assert.Error(err)
		assert.Contains(err.Error(), "module declaration not found")
	})

	t.Run("handles-module-with-version", func(t *testing.T) {
		tempDir := t.TempDir()
		goModPath := filepath.Join(tempDir, "go.mod")
		goModContent := []byte("module github.com/example/versioned/v2\n\ngo 1.21\n")
		err := os.WriteFile(goModPath, goModContent, 0644)
		assert.NoError(err)

		moduleName, moduleRoot, err := GetModuleInfo(tempDir)
		assert.NoError(err)
		assert.Equal("github.com/example/versioned/v2", moduleName)
		assert.Equal(tempDir, moduleRoot)
	})

	t.Run("handles-module-with-extra-whitespace", func(t *testing.T) {
		tempDir := t.TempDir()
		goModPath := filepath.Join(tempDir, "go.mod")
		goModContent := []byte("  module   github.com/example/whitespace  \n\ngo 1.21\n")
		err := os.WriteFile(goModPath, goModContent, 0644)
		assert.NoError(err)

		moduleName, moduleRoot, err := GetModuleInfo(tempDir)
		assert.NoError(err)
		assert.Equal("github.com/example/whitespace", moduleName)
		assert.Equal(tempDir, moduleRoot)
	})
}
