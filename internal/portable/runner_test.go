package portable

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractFS(t *testing.T) {
	mapFS := fstest.MapFS{
		"petstore.yml":              &fstest.MapFile{Data: []byte("openapi: 3.0.0")},
		"app.yml":                   &fstest.MapFile{Data: []byte("port: 3000")},
		"context.yml":               &fstest.MapFile{Data: []byte("key: value")},
		"static/svc/GET_hello.json": &fstest.MapFile{Data: []byte(`{"msg":"hi"}`)},
	}

	dir := t.TempDir()
	err := extractFS(mapFS, dir)
	require.NoError(t, err)

	t.Run("extracts files at root", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(dir, "petstore.yml"))
		require.NoError(t, err)
		assert.Equal(t, "openapi: 3.0.0", string(data))
	})

	t.Run("extracts nested files", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(dir, "static", "svc", "GET_hello.json"))
		require.NoError(t, err)
		assert.Equal(t, `{"msg":"hi"}`, string(data))
	})

	t.Run("creates directories", func(t *testing.T) {
		info, err := os.Stat(filepath.Join(dir, "static", "svc"))
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})
}

func TestExtractFS_empty(t *testing.T) {
	mapFS := fstest.MapFS{}
	dir := t.TempDir()
	err := extractFS(mapFS, dir)
	require.NoError(t, err)
}

func TestFileExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "exists.txt")
	require.NoError(t, os.WriteFile(path, []byte("hi"), 0o644))

	assert.True(t, fileExists(path))
	assert.False(t, fileExists(filepath.Join(dir, "nope.txt")))
}

func TestExtractFS_withEmbeddedSpec(t *testing.T) {
	specBytes := loadTestSpec(t, "petstore.yml")

	mapFS := fstest.MapFS{
		"petstore.yml": &fstest.MapFile{Data: specBytes},
	}

	dir := t.TempDir()
	require.NoError(t, extractFS(mapFS, dir))

	specPath := filepath.Join(dir, "petstore.yml")
	require.True(t, fileExists(specPath))

	// Verify the extracted spec can be used to create a handler
	data, err := os.ReadFile(specPath)
	require.NoError(t, err)

	h, err := newHandler(data)
	require.NoError(t, err)
	assert.NotEmpty(t, h.Routes())
}
