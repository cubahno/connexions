package files

import (
	"embed"
	"os"
	"path/filepath"
	"testing"

	assert2 "github.com/stretchr/testify/assert"
)

//go:embed testdata/*
var testDataFS embed.FS

func TestCopyFile(t *testing.T) {
	assert := assert2.New(t)

	t.Run("happy-path", func(t *testing.T) {
		base1 := t.TempDir()
		_ = os.MkdirAll(filepath.Join(base1, "subdir11", "subdir12"), 0755)
		src := filepath.Join(base1, "subdir11", "test1.txt")
		_ = os.WriteFile(src, []byte("test content"), 0644)

		base2 := t.TempDir()
		dest := filepath.Join(base2, "subdir11", "subdir2", "target.txt")

		err := CopyFile(src, dest)
		assert.Nil(err)

		// Verify file exists
		_, err = os.Stat(dest)
		assert.Nil(err)

		// Verify content matches
		content, err := os.ReadFile(dest)
		assert.Nil(err)
		assert.Equal("test content", string(content))
	})

	t.Run("invalid-source", func(t *testing.T) {
		err := CopyFile("/non-existent", "/")
		assert.Error(err)
	})

	t.Run("invalid-dest-dir", func(t *testing.T) {
		// Create a temp file to use as source
		tempDir := t.TempDir()
		src := filepath.Join(tempDir, "operation.yml")

		// Read embedded file and write to temp location
		content, err := testDataFS.ReadFile("testdata/operation.yml")
		assert.NoError(err)
		err = os.WriteFile(src, content, 0644)
		assert.NoError(err)

		err = CopyFile(src, "/root/foo/op.yml")
		assert.Error(err)
	})

	t.Run("invalid-dest-path", func(t *testing.T) {
		// Create a temp file to use as source
		tempDir := t.TempDir()
		src := filepath.Join(tempDir, "operation.yml")

		// Read embedded file and write to temp location
		content, err := testDataFS.ReadFile("testdata/operation.yml")
		assert.NoError(err)
		err = os.WriteFile(src, content, 0644)
		assert.NoError(err)

		err = CopyFile(src, "/root")
		assert.Error(err)
	})

	t.Run("creates-nested-directories", func(t *testing.T) {
		base1 := t.TempDir()
		src := filepath.Join(base1, "source.txt")
		_ = os.WriteFile(src, []byte("nested test"), 0644)

		base2 := t.TempDir()
		dest := filepath.Join(base2, "a", "b", "c", "d", "target.txt")

		err := CopyFile(src, dest)
		assert.Nil(err)

		content, err := os.ReadFile(dest)
		assert.Nil(err)
		assert.Equal("nested test", string(content))
	})
}

func TestCopyDirectory(t *testing.T) {
	assert := assert2.New(t)

	t.Run("happy-path", func(t *testing.T) {
		base1 := t.TempDir()
		_ = os.MkdirAll(filepath.Join(base1, "subdir11", "subdir12"), 0755)
		_ = os.WriteFile(filepath.Join(base1, "subdir11", "test1.txt"), []byte("test"), 0644)
		_ = os.WriteFile(filepath.Join(base1, "subdir11", "subdir12", "test2.txt"), []byte("test2"), 0644)

		base2 := t.TempDir()
		err := CopyDirectory(base1, base2)
		assert.Nil(err)

		// Verify files exist
		_, err = os.Stat(filepath.Join(base2, "subdir11", "test1.txt"))
		assert.Nil(err)
		_, err = os.Stat(filepath.Join(base2, "subdir11", "subdir12", "test2.txt"))
		assert.Nil(err)

		// Verify content
		content, err := os.ReadFile(filepath.Join(base2, "subdir11", "subdir12", "test2.txt"))
		assert.Nil(err)
		assert.Equal("test2", string(content))
	})

	t.Run("invalid-source", func(t *testing.T) {
		err := CopyDirectory("/non-existent", "/")
		assert.Error(err)
	})

	t.Run("empty-directory", func(t *testing.T) {
		base1 := t.TempDir()
		base2 := t.TempDir()

		err := CopyDirectory(base1, base2)
		assert.Nil(err)
	})

	t.Run("single-file", func(t *testing.T) {
		base1 := t.TempDir()
		_ = os.WriteFile(filepath.Join(base1, "single.txt"), []byte("single file"), 0644)

		base2 := t.TempDir()
		err := CopyDirectory(base1, base2)
		assert.Nil(err)

		content, err := os.ReadFile(filepath.Join(base2, "single.txt"))
		assert.Nil(err)
		assert.Equal("single file", string(content))
	})
}
