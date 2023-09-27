package connexions

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/jaswdr/faker"
	base2 "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
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

func CreateSchemaFromYAMLFile(t *testing.T, filePath string, target any) {
	t.Helper()
	cont, err := os.ReadFile(filePath)
	if err != nil {
		t.Errorf("Error reading file: %v", err)
		t.FailNow()
	}

	// remove schema key if pre
	tmp := make(map[string]any)
	_ = yaml.Unmarshal(cont, &tmp)
	if _, ok := tmp["schema"]; ok {
		cont, _ = yaml.Marshal(tmp["schema"])
	}

	// to json
	var data any
	err = yaml.Unmarshal(cont, &data)
	if err != nil {
		t.Errorf("Error unmarshaling YAML: %v\n", err)
		t.FailNow()
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		t.Errorf("Error marshaling JSON: %v\n", err)
		t.FailNow()
	}

	// target := openapi3.NewSchema()
	err = json.Unmarshal(jsonBytes, target)
	if err != nil {
		t.Errorf("Error parsing JSON: %v", err)
		t.FailNow()
	}
}

func CreateOperationFromYAMLFile(t *testing.T, filePath string, target any) {
	t.Helper()
	cont, err := os.ReadFile(filePath)
	if err != nil {
		t.Errorf("Error reading file: %v", err)
		t.FailNow()
	}

	// remove schema key if pre
	tmp := make(map[string]any)
	_ = json.Unmarshal(cont, &tmp)
	if _, ok := tmp["operation"]; ok {
		cont, _ = json.Marshal(tmp["operation"])
	}

	// to json
	var data any
	err = yaml.Unmarshal(cont, &data)
	if err != nil {
		t.Errorf("Error unmarshaling YAML: %v\n", err)
		t.FailNow()
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		t.Errorf("Error marshaling JSON: %v\n", err)
		t.FailNow()
	}

	err = json.Unmarshal(jsonBytes, target)
	if err != nil {
		t.Errorf("Error parsing JSON: %v", err)
		t.FailNow()
	}
}

func CreateKinSchemaFromString(t *testing.T, jsonSrc string) *openapi3.Schema {
	t.Helper()
	schema := openapi3.NewSchema()
	err := json.Unmarshal([]byte(jsonSrc), schema)
	if err != nil {
		t.Errorf("Error parsing JSON: %v", err)
		t.FailNow()
	}
	return schema
}

func CreateSchemaFromString(t *testing.T, jsonSrc string) *Schema {
	t.Helper()
	return NewSchemaFromKin(CreateKinSchemaFromString(t, jsonSrc), nil)
}

func CreateLibSchemaFromString(t *testing.T, ymlSchema string) *base2.SchemaProxy {
	t.Helper()
	// unmarshal raw bytes
	var node yaml.Node
	_ = yaml.Unmarshal([]byte(ymlSchema), &node)

	// build out the low-level model
	var lowSchema base.SchemaProxy
	_ = low.BuildModel(node.Content[0], &lowSchema)
	_ = lowSchema.Build(node.Content[0], nil)

	// build the high level schema proxy
	return base2.NewSchemaProxy(&low.NodeReference[*base.SchemaProxy]{
		Value: &lowSchema,
	})
}

func AssertJSONEqual(t *testing.T, expected, actual any) {
	t.Helper()
	expectedJSON, _ := json.Marshal(expected)
	actualJSON, _ := json.Marshal(actual)

	// Compare JSON representations
	assert.Equal(t, string(expectedJSON), string(actualJSON), "JSON representations should match")
}

func GetJSONPair(expected, actual any) (string, string) {
	expectedJSON, _ := json.Marshal(expected)
	actualJSON, _ := json.Marshal(actual)

	return string(expectedJSON), string(actualJSON)
}

func SetupApp(appDir string) (*Router, error) {
	cfg := MustConfig(appDir)
	err := MustFileStructure(cfg.App.Paths)
	if err != nil {
		return nil, err
	}
	_ = SaveFile(cfg.App.Paths.ConfigFile, []byte(""))

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

func NewTestReplaceContext(schema any) *ReplaceContext {
	return &ReplaceContext{
		Faker:  faker.New(),
		Schema: schema,
	}
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

// Custom transport to simulate an error during reading
type mockTransportWithReadError struct{}

func (t *mockTransportWithReadError) RoundTrip(req *http.Request) (*http.Response, error) {
	// Create a response with a body that returns an error when read
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/plain"}},
		Body:       &mockBodyWithReadError{},
	}, nil
}

type mockBodyWithReadError struct{}

func (b *mockBodyWithReadError) Close() error {
	return nil
}

func (b *mockBodyWithReadError) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF // Simulate an error
}
