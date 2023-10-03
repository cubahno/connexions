package connexions

import (
	"encoding/json"
	"github.com/cubahno/connexions/openapi"
	"github.com/cubahno/connexions/openapi/providers/kin"
	"github.com/cubahno/connexions/replacers"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/jaswdr/faker"
	base2 "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
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

func CreateSchemaFromString(t *testing.T, jsonSrc string) *openapi.Schema {
	t.Helper()
	return kin.NewSchemaFromKin(CreateKinSchemaFromString(t, jsonSrc), nil)
}

func CreateLibSchemaFromString(t *testing.T, ymlSchema string) *base2.SchemaProxy {
	t.Helper()
	// unmarshal raw bytes
	var node yaml.Node
	_ = yaml.Unmarshal([]byte(ymlSchema), &node)

	// build out the low-level model
	var lowSchema base.SchemaProxy
	_ = low.BuildModel(node.Content[0], &lowSchema)
	_ = lowSchema.Build(nil, node.Content[0], nil)

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

func NewTestReplaceContext(schema any) *replacers.ReplaceContext {
	return &replacers.ReplaceContext{
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
