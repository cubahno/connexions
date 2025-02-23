package files

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/cubahno/connexions/internal/config"
	"github.com/cubahno/connexions/internal/testhelpers"
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

func TestSaveFile(t *testing.T) {
	assert := assert2.New(t)

	t.Run("happy-path", func(t *testing.T) {
		contents := []byte("test file contents")
		filePath := filepath.Join(t.TempDir(), "a", "b", "c", "test.txt")
		err := SaveFile(filePath, contents)
		assert.NoError(err)
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
}

func TestCopyFile(t *testing.T) {
	assert := assert2.New(t)

	t.Run("happy-path", func(t *testing.T) {
		base1 := t.TempDir()
		_ = os.MkdirAll(filepath.Join(base1, "subdir11", "subdir12"), 0755)
		src := filepath.Join(base1, "subdir11", "test1.txt")
		_ = os.WriteFile(src, []byte("test"), 0644)

		base2 := t.TempDir()
		dest := filepath.Join(base2, "subdir11", "subdir2", "target.txt")

		err := CopyFile(src, dest)
		assert.Nil(err)
		_, err = os.Stat(dest)
		assert.Nil(err)
	})

	t.Run("invalid-source", func(t *testing.T) {
		err := CopyFile("/non-existent", "/")
		assert.Error(err)
	})

	t.Run("invalid-dest-dir", func(t *testing.T) {
		src := filepath.Join(testDataPath, "operation.yml")
		err := CopyFile(src, "/root/foo/op.yml")
		assert.Error(err)
	})

	t.Run("invalid-dest-path", func(t *testing.T) {
		src := filepath.Join(testDataPath, "operation.yml")
		err := CopyFile(src, "/root")
		assert.Error(err)
	})
}

func TestCopyDirectory(t *testing.T) {
	assert := assert2.New(t)

	t.Run("happy-path", func(t *testing.T) {
		base1 := t.TempDir()
		_ = os.MkdirAll(filepath.Join(base1, "subdir11", "subdir12"), 0755)
		_ = os.WriteFile(filepath.Join(base1, "subdir11", "test1.txt"), []byte("test"), 0644)
		_ = os.WriteFile(filepath.Join(base1, "subdir11", "subdir12", "test1.txt"), []byte("test"), 0644)

		base2 := t.TempDir()
		err := CopyDirectory(base1, base2)
		assert.Nil(err)
		_, err = os.Stat(filepath.Join(base2, "subdir11", "test1.txt"))
		assert.Nil(err)
		_, err = os.Stat(filepath.Join(base2, "subdir11", "subdir12", "test1.txt"))
		assert.Nil(err)
	})

	t.Run("invalid-source", func(t *testing.T) {
		err := CopyDirectory("/non-existent", "/")
		assert.Error(err)
	})
}

func TestIsEmptyDir(t *testing.T) {
	assert := assert2.New(t)
	tempDir := t.TempDir()
	assert.True(IsEmptyDir(tempDir))

	file, err := os.Create(filepath.Join(tempDir, "test.txt"))
	if err != nil {
		t.FailNow()
	}
	_ = file.Close()
	assert.False(IsEmptyDir(tempDir))
}

func TestIsJsonType(t *testing.T) {
	assert := assert2.New(t)
	assert.True(IsJsonType([]byte(`{"key": "value"}`)))
	assert.False(IsJsonType([]byte(`foo: bar`)))
}

func TestIsYamlType(t *testing.T) {
	assert := assert2.New(t)
	assert.True(IsYamlType([]byte(`foo: bar`)))
	assert.False(IsYamlType([]byte(`100`)))
}

func TestExtractZip(t *testing.T) {
	assert := assert2.New(t)

	getFileDirs := func(baseDir string) []string {
		return []string{
			filepath.Join(baseDir, "empty-dir"),
		}
	}

	getFilePaths := func(baseDir string) []string {
		return []string{
			filepath.Join(baseDir, "take-this", "index.json"),
			filepath.Join(baseDir, "take-this", config.RootOpenAPIName, "svc-1", "index.yml"),
			filepath.Join(baseDir, "take-this", config.RootOpenAPIName, "svc-2", "index.yml"),
			filepath.Join(baseDir, "take-this", config.RootServiceName, "svc-1", "get", "users", "index.json"),
			filepath.Join(baseDir, "ignore", config.RootServiceName, "svc-2", "get", "users", "all", "index.json"),
			filepath.Join(baseDir, "ignore", "svc-3", "patch", "users", "{userID}", "index.json"),
			filepath.Join(baseDir, "take-too", "ctx-1.yml"),
			filepath.Join(baseDir, "take-too", "ctx-2.yml"),
		}
	}

	createZip := func(dir string) string {
		var createdFiles []string

		for _, dirPath := range getFileDirs(dir) {
			if err := os.MkdirAll(dirPath, 0755); err != nil {
				t.FailNow()
			}
			createdFiles = append(createdFiles, dirPath)
		}

		for _, filePath := range getFilePaths(dir) {
			// Ensure the directory exists
			dir := filepath.Dir(filePath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.FailNow()
			}

			// Create the file
			file, err := os.Create(filePath)
			if err != nil {
				t.FailNow()
			}
			file.Close()

			createdFiles = append(createdFiles, filePath)
		}

		zipPath := filepath.Join(dir, "test.zip")
		zipFile, err := os.Create(zipPath)
		if err != nil {
			t.Fatalf("Failed to create zip file: %v", err)
		}
		defer zipFile.Close()

		zipWriter := zip.NewWriter(zipFile)

		for _, filePath := range createdFiles {
			file, err := os.Open(filePath)
			if err != nil {
				t.Fatalf("Failed to open file: %v", err)
			}
			defer file.Close()

			// Get the relative path for the zip entry
			relPath, err := filepath.Rel(dir, filePath)
			if err != nil {
				t.Fatalf("Failed to get relative path: %v", err)
			}

			zipEntry, err := zipWriter.Create(relPath)
			if err != nil {
				t.Fatalf("Failed to create zip entry: %v", err)
			}

			// Copy the file contents to the zip entry
			inf, _ := file.Stat()
			if inf.IsDir() {
				continue
			}

			_, err = io.Copy(zipEntry, file)
			if err != nil {
				t.Fatalf("Failed to copy to zip entry: %v", err)
			}
		}

		if err := zipWriter.Close(); err != nil {
			t.Fatalf("Failed to close zip writer: %v", err)
		}

		return zipPath
	}

	expectedFilePaths := func(baseDir string) []string {
		return []string{
			filepath.Join(baseDir, "take-this", "index.json"),
			filepath.Join(baseDir, "take-this", config.RootOpenAPIName, "svc-1", "index.yml"),
			filepath.Join(baseDir, "take-this", config.RootOpenAPIName, "svc-2", "index.yml"),
			filepath.Join(baseDir, "take-this", config.RootServiceName, "svc-1", "get", "users", "index.json"),
			filepath.Join(baseDir, "take-too", "ctx-1.yml"),
			filepath.Join(baseDir, "take-too", "ctx-2.yml"),
		}
	}

	t.Run("happy-path", func(t *testing.T) {
		zipPath := createZip(t.TempDir())

		zipReader, err := zip.OpenReader(zipPath)
		if err != nil {
			t.Fatalf("Failed to open zip filePath: %v", err)
		}
		defer zipReader.Close()

		// Extract and copy the zip contents
		targetDir := t.TempDir()
		err = ExtractZip(&zipReader.Reader, targetDir, []string{"take-this", "take-too"})
		if err != nil {
			t.Fatalf("Error extracting and copying files: %v", err)
		}

		var extracted []string
		_ = filepath.WalkDir(targetDir, func(path string, info os.DirEntry, err error) error {
			if info != nil && info.IsDir() {
				return nil
			}
			extracted = append(extracted, path)
			return nil
		})

		// Check if the target directory contains the extracted file
		expected := expectedFilePaths(targetDir)

		assert.ElementsMatch(expected, extracted)
	})

	t.Run("invalid-dest", func(t *testing.T) {
		zipPath := createZip(t.TempDir())
		zipReader, _ := zip.OpenReader(zipPath)
		defer zipReader.Close()

		dest := filepath.Join("/root", "test")
		res := ExtractZip(&zipReader.Reader, dest, []string{"take-this", "take-too"})
		assert.Error(res)
	})
}

func TestGetFileHash(t *testing.T) {
	assert := assert2.New(t)
	tempDir := t.TempDir()

	filePath := filepath.Join(tempDir, "test.txt")
	file, err := os.Create(filePath)
	if err != nil {
		t.FailNow()
	}
	_, _ = file.WriteString("test")
	defer func() {
		_ = file.Close()
	}()

	hash := GetFileHash(file)
	assert.Nil(err)
	assert.Equal("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", hash)
}

func TestHashFileWithError(t *testing.T) {
	m := &mockFile{readErr: true}
	hash := GetFileHash(m)

	if hash != "" {
		t.Errorf("Expected hash to be empty, got: %s", hash)
	}
}

func TestGetFileContentsFromURL(t *testing.T) {
	assert := assert2.New(t)

	t.Run("invalid-url", func(t *testing.T) {
		mockServer := testhelpers.CreateMockServer(t, "text/plain", "Hallo, Welt!", http.StatusOK)
		defer mockServer.Close()

		_, _, err := GetFileContentsFromURL(nil, "unknown-url")
		assert.Error(err)
	})

	t.Run("status-error", func(t *testing.T) {
		mockServer := testhelpers.CreateMockServer(t, "text/plain", "Hallo, Welt!", http.StatusNotFound)
		defer mockServer.Close()

		_, _, err := GetFileContentsFromURL(nil, mockServer.URL)
		assert.Equal(ErrGettingFileFromURL, err)
	})

	t.Run("read-error", func(t *testing.T) {
		mockServerWithReadError := testhelpers.CreateMockServer(t, "text/plain", "Hello, World!", http.StatusOK)
		defer mockServerWithReadError.Close()

		// Replace the response body with a reader that returns an error when read
		client := mockServerWithReadError.Client()
		client.Transport = &testhelpers.MockTransportWithReadError{}

		_, _, err := GetFileContentsFromURL(client, mockServerWithReadError.URL)
		if err == nil {
			t.Error("Expected an error while reading the response body, but got none")
		}
	})

	t.Run("happy-path", func(t *testing.T) {
		mockServer := testhelpers.CreateMockServer(t, "text/plain", "Hallo, Welt!", http.StatusOK)
		defer mockServer.Close()

		content, contentType, err := GetFileContentsFromURL(nil, mockServer.URL)
		assert.NoError(err)
		assert.Equal("text/plain", contentType)
		assert.Equal("Hallo, Welt!", string(content))
	})
}

func TestCleanupServiceFileStructure(t *testing.T) {
	assert := assert2.New(t)

	t.Run("happy-path", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create some subdirectories and files in the tempDir
		subDir1 := filepath.Join(tempDir, "subdir1")
		subDir2 := filepath.Join(tempDir, "subdir2")
		emptySubDir := filepath.Join(tempDir, "emptysubdir")
		err := os.Mkdir(subDir1, 0755)
		if err != nil {
			t.Fatal(err)
		}
		err = os.Mkdir(subDir2, 0755)
		if err != nil {
			t.Fatal(err)
		}
		err = os.Mkdir(emptySubDir, 0755)
		if err != nil {
			t.Fatal(err)
		}

		err = CleanupServiceFileStructure(tempDir)
		if err != nil {
			t.Errorf("Expected no error, but got: %v", err)
		}

		// Verify that emptySubDir has been removed
		if _, err := os.Stat(emptySubDir); !os.IsNotExist(err) {
			t.Error("Expected emptySubDir to be removed, but it still exists")
		}

		// Verify that subDir1 and subDir2 still exist
		if _, err := os.Stat(subDir1); !os.IsNotExist(err) {
			t.Error("Expected subDir1 to be removed, but it still exists")
		}
		if _, err := os.Stat(subDir2); !os.IsNotExist(err) {
			t.Error("Expected subDir2 to be removed, but it still exists")
		}
	})

	t.Run("file-insteadof-dir", func(t *testing.T) {
		tempDir := t.TempDir()
		f := filepath.Join(tempDir, "file.txt")
		_ = SaveFile(f, []byte("test"))

		err := CleanupServiceFileStructure(f)
		assert.Nil(err)
	})
}
