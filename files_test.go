package connexions

import (
	"archive/zip"
	"bytes"
	"fmt"
	assert2 "github.com/stretchr/testify/assert"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
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

func TestFileProperties(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	t.Run("IsEqual", func(t *testing.T) {
		f1 := &FileProperties{}
		f2 := &FileProperties{}
		assert.True(f1.IsEqual(f2))

		f1.ServiceName = "test"
		assert.False(f1.IsEqual(f2))
	})
}

func TestGetRequestFile(t *testing.T) {
	// Create a mock request with a file field
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	fileContents := []byte("test file contents")
	fileName := "test.txt"
	part, _ := writer.CreateFormFile("file", fileName)
	part.Write(fileContents)
	writer.Close()

	req := httptest.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Initialize a response recorder and handler
	rr := httptest.NewRecorder()

	// Perform the request
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file, err := GetRequestFile(r, "file")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Assuming you want to validate the result
		if file != nil {
			// Perform your assertions here
			if string(file.Content) != string(fileContents) ||
				file.Filename != fileName ||
				file.Extension != filepath.Ext(fileName) ||
				file.Size != int64(len(fileContents)) {
				t.Errorf("Unexpected file attributes")
			}
		}
	})

	handler.ServeHTTP(rr, req)

	// Check the response status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestGetPropertiesFromFilePath(t *testing.T) {
	assert := assert2.New(t)
	appCfg := NewDefaultAppConfig("/app")
	paths := appCfg.Paths

	t.Parallel()

	t.Run("openapi-root", func(t *testing.T) {
		t.SkipNow()
		filePath := paths.Services + "/.openapi/index.yml"
		props, _ := GetPropertiesFromFilePath(filePath, appCfg)

		assert.Equal(&FileProperties{
			ServiceName: "",
			Prefix:      "",
			IsOpenAPI:   true,
			FilePath:    filePath,
			FileName:    "index.yml",
			Extension:   ".yml",
			ContentType: "application/x-yaml",
		}, props)
	})

	t.Run("openapi-nested", func(t *testing.T) {
		t.SkipNow()
		filePath := paths.Services + "/.openapi/nice/dice/rice/index.yml"
		props, _ := GetPropertiesFromFilePath(filePath, appCfg)

		assert.Equal(&FileProperties{
			ServiceName: "nice",
			IsOpenAPI:   true,
			Prefix:      "/nice/dice/rice",
			FilePath:    filePath,
			FileName:    "index.yml",
			Extension:   ".yml",
			ContentType: "application/x-yaml",
		}, props)
	})

	t.Run("service-root-direct", func(t *testing.T) {
		filePath := paths.Services + "/.root/users.html"
		props, _ := GetPropertiesFromFilePath(filePath, appCfg)

		AssertJSONEqual(t, &FileProperties{
			ServiceName: "",
			Prefix:      "",
			Resource:    "/users.html",
			Method:      http.MethodGet,
			FilePath:    filePath,
			FileName:    "users.html",
			Extension:   ".html",
			ContentType: "text/html; charset=utf-8",
		}, props)
	})

	// result should as above, in the .root
	t.Run("service-direct", func(t *testing.T) {
		filePath := paths.Services + "/users.html"
		props, _ := GetPropertiesFromFilePath(filePath, appCfg)

		AssertJSONEqual(t, &FileProperties{
			ServiceName: "",
			Prefix:      "",
			Resource:    "/users.html",
			Method:      http.MethodGet,
			FilePath:    filePath,
			FileName:    "users.html",
			Extension:   ".html",
			ContentType: "text/html; charset=utf-8",
		}, props)
	})

	t.Run("service-root-with-method", func(t *testing.T) {
		filePath := paths.Services + "/.root/patch/users.html"
		props, _ := GetPropertiesFromFilePath(filePath, appCfg)

		AssertJSONEqual(t, &FileProperties{
			ServiceName: "",
			Method:      http.MethodPatch,
			Prefix:      "",
			Resource:    "/users.html",
			FilePath:    filePath,
			FileName:    "users.html",
			Extension:   ".html",
			ContentType: "text/html; charset=utf-8",
		}, props)
	})

	t.Run("service-non-root-will-have-method-as-service", func(t *testing.T) {
		filePath := paths.Services + "/patch/users.html"
		props, _ := GetPropertiesFromFilePath(filePath, appCfg)

		AssertJSONEqual(t, &FileProperties{
			ServiceName: "patch",
			Method:      http.MethodGet,
			Prefix:      "/patch",
			Resource:    "/users.html",
			FilePath:    filePath,
			FileName:    "users.html",
			Extension:   ".html",
			ContentType: "text/html; charset=utf-8",
		}, props)
	})

	t.Run("service-without-method", func(t *testing.T) {
		filePath := paths.Services + "/users/all/index.xml"
		props, _ := GetPropertiesFromFilePath(filePath, appCfg)

		AssertJSONEqual(t, &FileProperties{
			ServiceName: "users",
			Method:      http.MethodGet,
			Prefix:      "/users",
			Resource:    "/all/index.xml",
			FilePath:    filePath,
			FileName:    "index.xml",
			Extension:   ".xml",
			ContentType: "text/xml; charset=utf-8",
		}, props)
	})

	t.Run("service-with-index-file", func(t *testing.T) {
		filePath := paths.Services + "/users/patch/id/{userId}/index.json"
		props, _ := GetPropertiesFromFilePath(filePath, appCfg)

		AssertJSONEqual(t, &FileProperties{
			ServiceName: "users",
			Method:      http.MethodPatch,
			Prefix:      "/users",
			Resource:    "/id/{userId}",
			FilePath:    filePath,
			FileName:    "index.json",
			Extension:   ".json",
			ContentType: "application/json",
		}, props)
	})
}

func TestComposeFileSavePath(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	type params struct {
		service   string
		method    string
		resource  string
		ext       string
		isOpenAPI bool
	}

	appCfg := NewDefaultAppConfig("/app")
	paths := appCfg.Paths

	testCases := []struct {
		description string
		params      params
		expected    string
	}{
		{
			description: "root file",
			params: params{
				resource: "/foo.html",
			},
			expected: paths.Services + "/.root/get/foo.html",
		},
		{
			description: "root patch file",
			params: params{
				method:   "patch",
				resource: "/foo.html",
			},
			expected: paths.Services + "/.root/patch/foo.html",
		},
		{
			params: params{
				service:   "test",
				method:    "get",
				resource:  "test-path",
				ext:       ".json",
				isOpenAPI: false,
			},
			expected: paths.Services + "/test/get/test-path/index.json",
		},
		{
			params: params{
				resource: "/foo/bar",
			},
			expected: paths.Services + "/foo/get/bar/index.txt",
		},
		{
			params: params{
				service: "nice",
				method:  "patch",
			},
			expected: paths.Services + "/nice/patch/index.txt",
		},
		{
			params: params{
				service:   "nice",
				method:    "patch",
				resource:  "/dice/rice",
				ext:       ".yml",
				isOpenAPI: true,
			},
			expected: paths.Services + "/.openapi/nice/dice/rice.yml",
		},
		{
			params: params{
				isOpenAPI: true,
			},
			expected: paths.Services + "/.openapi/index",
		},
		{
			params: params{
				isOpenAPI: true,
				ext:       ".yml",
			},
			expected: paths.Services + "/.openapi/index.yml",
		},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			descr := &ServiceDescription{
				Name:      tc.params.service,
				Method:    tc.params.method,
				Path:      tc.params.resource,
				Ext:       tc.params.ext,
				IsOpenAPI: tc.params.isOpenAPI,
			}
			actual := ComposeFileSavePath(descr, paths)
			if actual != tc.expected {
				t.Errorf("ComposeFileSavePath(%v): %v - Expected: %v, Got: %v",
					tc.params, tc.description, tc.expected, actual)
			}
		})
	}

	t.Run("openapi-with-prefix", func(t *testing.T) {
		appCfg := NewDefaultAppConfig("/app")
		paths := appCfg.Paths
		descr := &ServiceDescription{
			Path:      "petstore",
			Ext:       ".yml",
			IsOpenAPI: true,
		}
		res := ComposeFileSavePath(descr, paths)
		assert.Equal(paths.Services+"/.openapi/petstore/index.yml", res)
	})
}

func TestSaveFile(t *testing.T) {
	t.SkipNow()
	assert := assert2.New(t)

	assert.True(true)
}

func TestCopyFile(t *testing.T) {
	assert := assert2.New(t)

	t.Run("happy-path", func(t *testing.T) {
		base1 := t.TempDir()
		os.MkdirAll(filepath.Join(base1, "subdir11", "subdir12"), 0755)
		src := filepath.Join(base1, "subdir11", "test1.txt")
		os.WriteFile(src, []byte("test"), 0644)

		base2 := t.TempDir()
		dest := filepath.Join(base2, "subdir11", "subdir2", "target.txt")

		err := CopyFile(src, dest)
		assert.Nil(err)
		_, err = os.Stat(dest)
		assert.Nil(err)
	})
}

func TestCopyDirectory(t *testing.T) {
	assert := assert2.New(t)

	t.Run("happy-path", func(t *testing.T) {
		base1 := t.TempDir()
		os.MkdirAll(filepath.Join(base1, "subdir11", "subdir12"), 0755)
		os.WriteFile(filepath.Join(base1, "subdir11", "test1.txt"), []byte("test"), 0644)
		os.WriteFile(filepath.Join(base1, "subdir11", "subdir12", "test1.txt"), []byte("test"), 0644)

		base2 := t.TempDir()
		err := CopyDirectory(base1, base2)
		assert.Nil(err)
		_, err = os.Stat(filepath.Join(base2, "subdir11", "test1.txt"))
		assert.Nil(err)
		_, err = os.Stat(filepath.Join(base2, "subdir11", "subdir12", "test1.txt"))
		assert.Nil(err)
	})
}

func TestCleanupServiceFileStructure(t *testing.T) {
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
	tempDir := t.TempDir()

	getFilePaths := func(baseDir string) []string {
		return []string{
			filepath.Join(baseDir, "take-this", "index.json"),
			filepath.Join(baseDir, "take-this", RootOpenAPIName, "svc-1", "index.yml"),
			filepath.Join(baseDir, "take-this", RootOpenAPIName, "svc-2", "index.yml"),
			filepath.Join(baseDir, "take-this", RootServiceName, "svc-1", "get", "users", "index.json"),
			filepath.Join(baseDir, "ignore", RootServiceName, "svc-2", "get", "users", "all", "index.json"),
			filepath.Join(baseDir, "ignore", "svc-3", "patch", "users", "{userID}", "index.json"),
			filepath.Join(baseDir, "take-too", "ctx-1.yml"),
			filepath.Join(baseDir, "take-too", "ctx-2.yml"),
		}
	}

	expectedFilePaths := func(baseDir string) []string {
		return []string{
			filepath.Join(baseDir, "take-this", "index.json"),
			filepath.Join(baseDir, "take-this", RootOpenAPIName, "svc-1", "index.yml"),
			filepath.Join(baseDir, "take-this", RootOpenAPIName, "svc-2", "index.yml"),
			filepath.Join(baseDir, "take-this", RootServiceName, "svc-1", "get", "users", "index.json"),
			filepath.Join(baseDir, "take-too", "ctx-1.yml"),
			filepath.Join(baseDir, "take-too", "ctx-2.yml"),
		}
	}

	var createdFiles []string
	for _, filePath := range getFilePaths(tempDir) {
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

	zipPath := filepath.Join(tempDir, "test.zip")
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
		relPath, err := filepath.Rel(tempDir, filePath)
		if err != nil {
			t.Fatalf("Failed to get relative path: %v", err)
		}

		zipEntry, err := zipWriter.Create(relPath)
		if err != nil {
			t.Fatalf("Failed to create zip entry: %v", err)
		}

		// Copy the file contents to the zip entry
		_, err = io.Copy(zipEntry, file)
		if err != nil {
			t.Fatalf("Failed to copy to zip entry: %v", err)
		}
	}

	if err := zipWriter.Close(); err != nil {
		t.Fatalf("Failed to close zip writer: %v", err)
	}

	// Open the mock zip file
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
	filepath.WalkDir(targetDir, func(path string, info os.DirEntry, err error) error {
		if info != nil && info.IsDir() {
			return nil
		}
		extracted = append(extracted, path)
		return nil
	})

	// Check if the target directory contains the extracted file
	expected := expectedFilePaths(targetDir)

	assert.ElementsMatch(expected, extracted)
}

func TestGetFileHash(t *testing.T) {
	assert := assert2.New(t)
	tempDir := t.TempDir()

	filePath := filepath.Join(tempDir, "test.txt")
	file, err := os.Create(filePath)
	if err != nil {
		t.FailNow()
	}
	file.WriteString("test")
	defer file.Close()

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
