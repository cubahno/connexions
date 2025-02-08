package api

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/cubahno/connexions/internal"
	"github.com/cubahno/connexions/internal/config"
	"github.com/stretchr/testify/assert"
)

// Custom testingLogWriter that discards log output
type testingLogWriter struct{}

func (lw *testingLogWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func TestMain(m *testing.M) {
	// Disable global log output for tests
	_ = os.Setenv("DISABLE_LOGGER", "true")
	log.SetOutput(&testingLogWriter{})

	// Run tests
	code := m.Run()

	os.Exit(code)
}

func SetupApp(appDir string) (*Router, error) {
	cfg := config.MustConfig(appDir)
	err := MustFileStructure(cfg.App.Paths)
	if err != nil {
		return nil, err
	}
	_ = internal.SaveFile(cfg.App.Paths.ConfigFile, []byte(""))

	return NewRouter(cfg), nil
}

func UnmarshallResponse[T any](t *testing.T, res *bytes.Buffer) *T {
	t.Helper()
	target := new(T)
	err := json.Unmarshal(res.Bytes(), &target)
	if err != nil {
		t.Errorf("Error unmarshaling JSON: %v\n", err)
		t.FailNow()
	}
	return target
}

func AddTestFileToForm(writer *multipart.Writer, fieldName, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	baseName := filepath.Base(filePath)

	part, err := writer.CreateFormFile(fieldName, baseName)
	if err != nil {
		return err
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return err
	}

	return nil
}

func CreateTestZip(files map[string]string) *bytes.Buffer {
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	if len(files) == 0 {
		files = map[string]string{
			"file1.txt": "This is file 1 content.",
			"file2.txt": "This is file 2 content.",
		}
	}

	for name, contents := range files {
		file, _ := zipWriter.Create(name)
		_, _ = file.Write([]byte(contents))
	}

	_ = zipWriter.Close()
	return &buf
}

func CreateTestMapFormReader(data map[string]string) (*multipart.Writer, *bytes.Buffer) {
	var bodyBuffer bytes.Buffer
	writer := multipart.NewWriter(&bodyBuffer)

	for k, v := range data {
		err := writer.WriteField(k, v)
		if err != nil {
			return nil, nil
		}
	}

	writer.Close()

	return writer, &bodyBuffer
}

func createMockServer(t *testing.T, contentType, responseBody string, responseStatus int) *httptest.Server {
	t.Helper()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(responseStatus)
		_, err := w.Write([]byte(responseBody))
		if err != nil {
			t.Errorf("Error writing response: %v", err)
			t.FailNow()
		}
	})
	return httptest.NewServer(handler)
}

func AssertJSONEqual(t *testing.T, expected, actual any) {
	t.Helper()
	expectedJSON, _ := json.Marshal(expected)
	actualJSON, _ := json.Marshal(actual)

	// Compare JSON representations
	assert.Equal(t, string(expectedJSON), string(actualJSON), "JSON representations should match")
}
