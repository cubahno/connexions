package config

import (
	"path/filepath"
	"testing"

	assert2 "github.com/stretchr/testify/assert"
)

func TestNewPaths(t *testing.T) {
	assert := assert2.New(t)

	t.Run("creates paths with correct structure", func(t *testing.T) {
		baseDir := "/test/base"
		paths := NewPaths(baseDir)

		assert.NotNil(paths)
		assert.Equal(baseDir, paths.Base)
		assert.Equal(filepath.Join(baseDir, "resources"), paths.Resources)
		assert.Equal(filepath.Join(baseDir, "resources", "data", "services"), paths.Services)
		assert.Equal(filepath.Join(baseDir, "resources", "docs"), paths.Docs)
		assert.Equal(filepath.Join(baseDir, "resources", "ui"), paths.UI)
	})

	t.Run("handles relative paths", func(t *testing.T) {
		baseDir := "."
		paths := NewPaths(baseDir)

		assert.NotNil(paths)
		assert.Equal(".", paths.Base)
		assert.Equal(filepath.Join(".", "resources"), paths.Resources)
	})

	t.Run("handles empty base dir", func(t *testing.T) {
		baseDir := ""
		paths := NewPaths(baseDir)

		assert.NotNil(paths)
		assert.Equal("", paths.Base)
		assert.Equal("resources", paths.Resources)
	})
}
