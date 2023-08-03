package files

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	assert2 "github.com/stretchr/testify/assert"
)

type mockFile struct {
	readErr bool
}

func (m *mockFile) Read(p []byte) (n int, err error) {
	if m.readErr {
		return 0, fmt.Errorf("simulated error")
	}
	return len(p), nil
}

func (m *mockFile) Close() error {
	return nil
}

func TestGetFileHash(t *testing.T) {
	assert := assert2.New(t)

	t.Run("hash-from-file", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "test.txt")
		file, err := os.Create(filePath)
		if err != nil {
			t.FailNow()
		}
		_, _ = file.WriteString("test")

		// Seek to beginning to read
		_, _ = file.Seek(0, 0)

		hash := GetFileHash(file)
		_ = file.Close()

		assert.NotEmpty(hash)
		// SHA256 of "test" is 9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08
		assert.Equal("9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08", hash)
	})

	t.Run("hash-from-bytes", func(t *testing.T) {
		content := []byte("hello world")
		reader := bytes.NewReader(content)
		hash := GetFileHash(reader)

		assert.NotEmpty(hash)
		// SHA256 of "hello world"
		assert.Equal("b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9", hash)
	})

	t.Run("hash-empty-content", func(t *testing.T) {
		reader := strings.NewReader("")
		hash := GetFileHash(reader)

		assert.NotEmpty(hash)
		// SHA256 of empty string
		assert.Equal("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", hash)
	})

	t.Run("hash-with-read-error", func(t *testing.T) {
		m := &mockFile{readErr: true}
		hash := GetFileHash(m)

		assert.Empty(hash)
	})

	t.Run("consistent-hashing", func(t *testing.T) {
		content := []byte("consistent test")

		hash1 := GetFileHash(bytes.NewReader(content))
		hash2 := GetFileHash(bytes.NewReader(content))

		assert.Equal(hash1, hash2)
	})
}
